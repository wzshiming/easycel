package easycel

import (
	"reflect"

	"github.com/google/cel-go/cel"
)

// convertToCelType converts the Golang reflect.Type to CEL type
func convertToCelType(refType reflect.Type) (*cel.Type, bool) {
	switch refType.Kind() {
	case reflect.Pointer:
		ptrType, ok := convertToCelType(refType.Elem())
		if !ok {
			return nil, false
		}
		return cel.NullableType(ptrType), true
	case reflect.Bool:
		return cel.BoolType, true
	case reflect.Float32, reflect.Float64:
		return cel.DoubleType, true
	case reflect.Int64:
		if refType == durationType || refType == typesDurationType {
			return cel.DurationType, true
		}
		return cel.IntType, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return cel.IntType, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cel.UintType, true
	case reflect.String:
		return cel.StringType, true
	case reflect.Slice:
		refElem := refType.Elem()
		if refElem == byteType {
			return cel.BytesType, true
		}
		elemType, ok := convertToCelType(refElem)
		if !ok {
			return nil, false
		}
		return cel.ListType(elemType), true
	case reflect.Array:
		elemType, ok := convertToCelType(refType.Elem())
		if !ok {
			return nil, false
		}
		return cel.ListType(elemType), true
	case reflect.Map:
		keyType, ok := convertToCelType(refType.Key())
		if !ok {
			return nil, false
		}
		elemType, ok := convertToCelType(refType.Elem())
		if !ok {
			return nil, false
		}
		return cel.MapType(keyType, elemType), true
	case reflect.Struct:
		if refType == timestampType || refType == typesTimestampType {
			return cel.TimestampType, true
		}
		return cel.ObjectType(rawTypeName(refType)), true
	case reflect.Interface:
		return cel.DynType, true
	}
	return nil, false
}
