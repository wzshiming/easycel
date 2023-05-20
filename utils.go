package easycel

import (
	"reflect"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

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
		if refType == durationType {
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
		if refType == timestampType {
			return cel.TimestampType, true
		}
		return cel.ObjectType(typeName(refType)), true
	case reflect.Interface:
		return cel.DynType, true
	}
	return nil, false
}

func NativeToValue(i any) ref.Val {
	return generalAdapter{}.NativeToValue(i)
}

type generalAdapter struct {
	baseAdapter ref.TypeAdapter
}

func (a generalAdapter) NativeToValue(i any) ref.Val {
	if i == nil {
		return types.NullValue
	}
	switch v := i.(type) {
	case ref.Val:
		return v
	case map[string]any:
		return types.NewStringInterfaceMap(a, v)
	case map[string]string:
		return types.NewStringStringMap(a, v)
	case []string:
		return types.NewStringList(a, v)
	}

	val := reflect.ValueOf(i)
	switch val.Kind() {
	case reflect.Pointer:
		if val.IsNil() {
			return types.NullValue
		}
		return a.NativeToValue(val.Elem().Interface())
	case reflect.Bool:
		return types.Bool(val.Bool())
	case reflect.Float64, reflect.Float32:
		return types.Double(val.Float())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return types.Int(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return types.Uint(val.Uint())
	case reflect.String:
		return types.String(val.String())
	case reflect.Slice:
		if val.IsNil() {
			return types.NullValue
		}
		if val.Type().Elem().Kind() == reflect.Uint8 {
			return types.Bytes(val.Bytes())
		}
		fallthrough
	case reflect.Array:
		sz := val.Len()
		elems := make([]ref.Val, 0, sz)
		for i := 0; i < sz; i++ {
			v := a.NativeToValue(val.Index(i).Interface())
			elems = append(elems, v)
		}
		return types.NewRefValList(a, elems)
	case reflect.Map:
		if val.IsNil() {
			return types.NullValue
		}
		entries := map[ref.Val]ref.Val{}
		iter := val.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()
			kv := a.NativeToValue(k.Interface())
			vv := a.NativeToValue(v.Interface())
			entries[kv] = vv
		}
		return types.NewRefValMap(a, entries)
	case reflect.Interface:
		if val.IsNil() {
			return types.NullValue
		}
		return a.NativeToValue(val.Elem().Interface())
	case reflect.Struct:
		if val.Type() == timestampType {
			return types.Timestamp{Time: val.Interface().(time.Time)}
		}
	}

	if a.baseAdapter != nil {
		return a.baseAdapter.NativeToValue(i)
	}
	return types.UnsupportedRefValConversionErr(i)
}
