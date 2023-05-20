package typeutils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/wzshiming/easycel"
)

type Struct[T comparable] struct {
	adapter ref.TypeAdapter
	entries map[string]string
	tagName string
	typ     reflect.Type
	value   reflect.Value
	refType ref.Type
	val     T
}

func NewStructWithJSONTag[T comparable](adapter ref.TypeAdapter, val T) Struct[T] {
	return NewStruct[T](adapter, val, "json")
}

func NewStruct[T comparable](adapter ref.TypeAdapter, val T, tagName string) Struct[T] {
	value := reflect.ValueOf(val)
	typ := value.Type()

	refType := easycel.GetTypeValue(reflect.TypeOf(Struct[T]{}))
	entries := GetStructFieldMap(typ, tagName)
	return Struct[T]{
		adapter: adapter,
		entries: entries,
		tagName: tagName,
		typ:     typ,
		value:   value,
		refType: refType,
		val:     val,
	}
}

func (s Struct[T]) Type() ref.Type {
	return s.refType
}

func (s Struct[T]) Get(index ref.Val) ref.Val {
	if index.Type() != types.StringType {
		return types.NewErr("struct index must be string")
	}
	key := index.Value().(string)
	key, ok := s.entries[key]
	if !ok {
		return types.NewErr("struct field not found")
	}
	val := s.value.FieldByName(key)
	if !val.IsValid() {
		return types.NewErr("struct field is invalid")
	}
	if s.adapter == nil {
		return easycel.NativeToValue(val.Interface())
	}
	return s.adapter.NativeToValue(val.Interface())
}

func (s Struct[T]) Value() any {
	return s.val
}

func (s Struct[T]) ConvertToNative(typeDesc reflect.Type) (any, error) {
	if s.typ.AssignableTo(typeDesc) {
		return s.val, nil
	}
	if reflect.TypeOf(s).AssignableTo(typeDesc) {
		return s, nil
	}
	return nil, fmt.Errorf("type conversion error from '%v' to '%v'", s.typ, typeDesc)
}

func (s Struct[T]) ConvertToType(typeValue ref.Type) ref.Val {
	if s.refType.TypeName() == typeValue.TypeName() {
		return s
	}
	return types.NewErr("type conversion error from '%v' to '%v'", s.typ, typeValue)
}

func (s Struct[T]) Equal(other ref.Val) ref.Val {
	if s.refType.TypeName() != other.Type().TypeName() {
		return types.False
	}
	if !s.value.Equal(reflect.ValueOf(other.Value())) {
		return types.False
	}
	return types.True
}

func fieldNameWithTag(field reflect.StructField, tagName string) (name string, exported bool) {
	value, ok := field.Tag.Lookup(tagName)
	if !ok {
		return field.Name, true
	}

	name = strings.Split(value, ",")[0]
	if name == "-" {
		return "", false
	}

	if name == "" {
		name = field.Name
	}
	return name, true
}
