package easycel

import (
	"reflect"

	"github.com/google/cel-go/cel"
)

var typeValueMap cache[reflect.Type, *cel.Type]

// convertToCelType converts the Golang reflect.Type to CEL type
func convertToCelType(refType reflect.Type) (*cel.Type, bool) {
	typ, ok := typeValueMap.load(refType)
	if ok {
		return typ, typ != nil
	}

	switch refType.Kind() {
	case reflect.Pointer:
		if ptrType, ok := convertToCelType(refType.Elem()); ok {
			typ = cel.NullableType(ptrType)
		}
	case reflect.Bool:
		typ = cel.BoolType
	case reflect.Float32, reflect.Float64:
		typ = cel.DoubleType
	case reflect.Int64:
		if refType == durationType {
			typ = cel.DurationType
		} else {
			typ = cel.IntType
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		typ = cel.IntType
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		typ = cel.UintType
	case reflect.String:
		typ = cel.StringType
	case reflect.Slice:
		refElem := refType.Elem()
		if refElem == byteType {
			typ = cel.BytesType
		} else if elemType, ok := convertToCelType(refElem); ok {
			typ = cel.ListType(elemType)
		}
	case reflect.Array:
		if elemType, ok := convertToCelType(refType.Elem()); ok {
			typ = cel.ListType(elemType)
		}
	case reflect.Map:
		keyType, keyOK := convertToCelType(refType.Key())
		elemType, elemOK := convertToCelType(refType.Elem())
		if keyOK && elemOK {
			typ = cel.MapType(keyType, elemType)
		}
	case reflect.Struct:
		// rawTypeName gives time.Time, types.Timestamp and types.Duration the names of well-known types
		typ = cel.ObjectType(rawTypeName(refType), getTrait(refType))
	case reflect.Interface:
		typ = cel.DynType
	}
	typ = typeValueMap.store(refType, typ)
	return typ, typ != nil
}
