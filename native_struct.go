package easycel

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func newStructObject(adapter types.Adapter, tagName string, val any, refValue reflect.Value) ref.Val {
	valType, err := newStructType(tagName, refValue.Type())
	if err != nil {
		return types.WrapErr(err)
	}
	return &structObject{
		Adapter:  adapter,
		val:      val,
		valType:  valType,
		refValue: refValue,
	}
}

type structObject struct {
	types.Adapter
	val      any
	valType  Type
	refValue reflect.Value
}

// ConvertToNative implements the ref.Val interface method.
func (o *structObject) ConvertToNative(typeDesc reflect.Type) (any, error) {
	if o.refValue.Type() == typeDesc {
		return o.val, nil
	}
	if o.refValue.Kind() == reflect.Pointer && o.refValue.Type().Elem() == typeDesc {
		return o.refValue.Elem().Interface(), nil
	}
	if typeDesc.Kind() == reflect.Pointer && o.refValue.Type() == typeDesc.Elem() {
		ptr := reflect.New(typeDesc.Elem())
		ptr.Elem().Set(o.refValue)
		return ptr.Interface(), nil
	}
	return nil, fmt.Errorf("type conversion error from '%v' to '%v'", o.Type(), typeDesc)
}

// ConvertToType implements the ref.Val interface method.
func (o *structObject) ConvertToType(typeVal ref.Type) ref.Val {
	if typeVal.TypeName() == o.valType.TypeName() {
		return o
	}
	return types.NewErr("type conversion error from '%s' to '%s'", o.Type(), typeVal)
}

// Equal implements the ref.Val interface method.
func (o *structObject) Equal(other ref.Val) ref.Val {
	otherNtv, ok := other.(*structObject)
	if !ok {
		return types.False
	}
	val := o.val
	otherVal := otherNtv.val
	refVal := o.refValue
	otherRefVal := otherNtv.refValue
	if refVal.Kind() != otherRefVal.Kind() {
		if refVal.Kind() == reflect.Pointer {
			val = refVal.Elem().Interface()
		} else if otherRefVal.Kind() == reflect.Pointer {
			otherVal = otherRefVal.Elem().Interface()
		}
	}
	return types.Bool(reflect.DeepEqual(val, otherVal))
}

// Type implements the ref.Val interface method.
func (o *structObject) Type() ref.Type {
	return o.valType
}

// Value implements the ref.Val interface method.
func (o *structObject) Value() any {
	return o.val
}

var structFieldMap = map[string]map[reflect.Type]map[string]int{}

func getStructFieldMap(typ reflect.Type, tagName string) map[string]int {
	if filedMap, ok := structFieldMap[tagName]; ok {
		if entries, ok := filedMap[typ]; ok {
			return entries
		}
	}
	entries := map[string]int{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		name := ""
		if tagName == "" {
			name = field.Name
		} else {
			var ok bool
			name, ok = fieldNameWithTag(field, tagName)
			if !ok {
				continue
			}
		}
		entries[name] = i
	}

	if _, ok := structFieldMap[tagName]; !ok {
		structFieldMap[tagName] = map[reflect.Type]map[string]int{}
	}

	structFieldMap[tagName][typ] = entries
	return entries
}

func isSupportedFieldType(refType reflect.Type) bool {
	switch refType.Kind() {
	case reflect.Ptr:
		return isSupportedType(refType.Elem())
	case reflect.Struct:
		return true
	case reflect.Map:
		return refType.Key().Kind() == reflect.String
	}
	return false
}

func isSupportedType(refType reflect.Type) bool {
	switch refType.Kind() {
	case reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Func, reflect.UnsafePointer, reflect.Uintptr:
		return false
	case reflect.Array, reflect.Slice:
		return isSupportedType(refType.Elem())
	case reflect.Map:
		return isSupportedType(refType.Key()) && isSupportedType(refType.Elem())
	}
	return true
}
