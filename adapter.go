package easycel

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type adapter struct {
	adapter ref.TypeAdapter
}

func newAdapter(a ref.TypeAdapter) *adapter {
	return &adapter{
		adapter: a,
	}
}

func (t *adapter) TypeToPbType(typ reflect.Type) (*exprpb.Type, error) {
	switch typ.Kind() {
	case reflect.Ptr:
		return t.TypeToPbType(typ.Elem())
	case reflect.Bool:
		return decls.Bool, nil
	case reflect.Float64, reflect.Float32:
		return decls.Double, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return decls.Int, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return decls.Uint, nil
	case reflect.String:
		return decls.String, nil
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return decls.Bytes, nil
		}
		fallthrough
	case reflect.Array:
		elem, err := t.TypeToPbType(typ.Elem())
		if err != nil {
			return nil, err
		}
		return decls.NewListType(elem), nil
	case reflect.Map:
		key, err := t.TypeToPbType(typ.Elem())
		if err != nil {
			return nil, err
		}
		val, err := t.TypeToPbType(typ.Elem())
		if err != nil {
			return nil, err
		}
		return decls.NewMapType(key, val), nil
	case reflect.Struct, reflect.Interface:
		typeName := getUniqTypeName("object", typ)
		return decls.NewObjectType(typeName), nil
	}

	return nil, fmt.Errorf("unsupported type conversion kind %s", typ.Kind())
}

func (t *adapter) ValueToCelValue(val reflect.Value) (ref.Val, error) {
	v, ok := val.Interface().(ref.Val)
	if ok {
		return v, nil
	}
	switch val.Kind() {
	case reflect.Bool:
		return types.Bool(val.Bool()), nil
	case reflect.Float64, reflect.Float32:
		return types.Double(val.Float()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return types.Int(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return types.Uint(val.Uint()), nil
	case reflect.String:
		return types.String(val.String()), nil
	case reflect.Slice:
		if val.Type().Elem().Kind() == reflect.Uint8 {
			return types.Bytes(val.Bytes()), nil
		}
		fallthrough
	case reflect.Array:
		sz := val.Len()
		elts := make([]ref.Val, 0, sz)
		for i := 0; i < sz; i++ {
			v, err := t.ValueToCelValue(val.Index(i))
			if err != nil {
				return nil, err
			}
			elts = append(elts, v)
		}
		return types.NewRefValList(t.adapter, elts), nil
	case reflect.Map:
		entries := map[ref.Val]ref.Val{}
		iter := val.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()
			kv, err := t.ValueToCelValue(k)
			if err != nil {
				return nil, err
			}
			vv, err := t.ValueToCelValue(v)
			if err != nil {
				return nil, err
			}
			entries[kv] = vv
		}
		return types.NewDynamicMap(t.adapter, entries), nil
	}
	return nil, fmt.Errorf("unsupported type conversion kind %s", val.Kind())
}
