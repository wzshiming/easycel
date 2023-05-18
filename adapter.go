package easycel

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// convertToExprType converts the Golang reflect.Type to a protobuf exprpb.Type.
func convertToExprType(refType reflect.Type) (*exprpb.Type, bool) {
	switch refType.Kind() {
	case reflect.Pointer:
		return convertToExprType(refType.Elem())
	case reflect.Bool:
		return decls.Bool, true
	case reflect.Float32, reflect.Float64:
		return decls.Double, true
	case reflect.Int64:
		if refType == durationType {
			return decls.Duration, true
		}
		return decls.Int, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return decls.Int, true
	case reflect.String:
		return decls.String, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return decls.Uint, true
	case reflect.Slice:
		refElem := refType.Elem()
		if refElem == byteType {
			return decls.Bytes, true
		}
		elemType, ok := convertToExprType(refElem)
		if !ok {
			return nil, false
		}
		return decls.NewListType(elemType), true
	case reflect.Array:
		elemType, ok := convertToExprType(refType.Elem())
		if !ok {
			return nil, false
		}
		return decls.NewListType(elemType), true
	case reflect.Map:
		keyType, ok := convertToExprType(refType.Key())
		if !ok {
			return nil, false
		}
		elemType, ok := convertToExprType(refType.Elem())
		if !ok {
			return nil, false
		}
		return decls.NewMapType(keyType, elemType), true
	case reflect.Struct:
		if refType == timestampType {
			return decls.Timestamp, true
		}
		return decls.Dyn, true
	case reflect.Interface:
		return decls.Dyn, true
	}
	return nil, false
}

// convertToRefVal converts the Golang reflect.Value to a CEL ref.Val.
func convertToRefVal(a ref.TypeAdapter, val reflect.Value, tagName string) (ref.Val, error) {
	i := val.Interface()
	if i == nil {
		return types.NullValue, nil
	}
	switch v := i.(type) {
	case ref.Val:
		return v, nil
	case map[string]interface{}:
		return types.NewStringInterfaceMap(a, v), nil
	case map[string]string:
		return types.NewStringStringMap(a, v), nil
	case []string:
		return types.NewStringList(a, v), nil
	}

	switch val.Kind() {
	case reflect.Pointer:
		if val.IsNil() {
			return types.NullValue, nil
		}
		return convertToRefVal(a, val.Elem(), tagName)
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
		if val.IsNil() {
			return types.NullValue, nil
		}
		if val.Type().Elem().Kind() == reflect.Uint8 {
			return types.Bytes(val.Bytes()), nil
		}
		fallthrough
	case reflect.Array:
		sz := val.Len()
		elems := make([]ref.Val, 0, sz)
		for i := 0; i < sz; i++ {
			v, err := convertToRefVal(a, val.Index(i), tagName)
			if err != nil {
				return nil, err
			}
			elems = append(elems, v)
		}
		return types.NewRefValList(a, elems), nil
	case reflect.Map:
		if val.IsNil() {
			return types.NullValue, nil
		}
		entries := map[ref.Val]ref.Val{}
		iter := val.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()
			kv, err := convertToRefVal(a, k, tagName)
			if err != nil {
				return nil, err
			}
			vv, err := convertToRefVal(a, v, tagName)
			if err != nil {
				return nil, err
			}
			entries[kv] = vv
		}
		return types.NewRefValMap(a, entries), nil
	case reflect.Struct:
		entries := map[ref.Val]ref.Val{}
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}
			f := val.Field(i)
			if !f.CanInterface() {
				continue
			}

			name := ""
			if tagName == "" {
				name = field.Name
			} else {
				var ok bool
				name, ok = fieldName(field, tagName)
				if !ok {
					continue
				}
			}

			v, err := convertToRefVal(a, f, tagName)
			if err != nil {
				return nil, err
			}
			entries[types.String(name)] = v
		}

		typeName := getUniqTypeName("object", typ)
		t := types.NewTypeValue(typeName,
			traits.IndexerType,
		)
		v := types.NewRefValMap(a, entries)

		return structTypeValue{
			entries: entries,
			Val:     v,
			typ:     t,
			val:     val.Interface(),
		}, nil
	case reflect.Interface:
		if val.IsNil() {
			return types.NullValue, nil
		}
		return convertToRefVal(a, val.Elem(), tagName)
	}

	v := a.NativeToValue(val)
	if !types.IsError(v) {
		return v, nil
	}
	return nil, fmt.Errorf("unsupported type conversion kind %s", val.Kind())
}

type structTypeValue struct {
	entries map[ref.Val]ref.Val
	ref.Val
	typ ref.Type
	val any
}

func (s structTypeValue) Type() ref.Type {
	return s.typ
}

func (s structTypeValue) Get(index ref.Val) ref.Val {
	return s.entries[index]
}

func (s structTypeValue) Value() any {
	return s.val
}

func fieldName(field reflect.StructField, tagName string) (name string, exported bool) {
	name, ok := field.Tag.Lookup(tagName)
	if ok && name == "-" {
		return "", false
	}
	if name != "" {
		name = strings.SplitN(name, ",", 2)[0]
	}

	if name == "" {
		name = field.Name
	}

	return name, true
}
