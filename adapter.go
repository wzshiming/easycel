package easycel

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type adapter struct {
}

func newAdapter() *adapter {
	return &adapter{}
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
	value := val.Interface()
	switch c := value.(type) {
	case CelVal:
		return c.CelVal(), nil
	case ref.Val:
		return c, nil
	}
	return nil, fmt.Errorf("unsupported type conversion kind %s", val.Kind())
}
