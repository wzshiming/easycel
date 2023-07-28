package easycel

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

var (
	structTypeTraitMask = traits.FieldTesterType | traits.IndexerType
)

func newStructType(tagName string, refType reflect.Type) (Type, error) {
	if refType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("unsupported reflect.Type %v, must be reflect.Struct", refType)
	}
	return &structType{
		tagName:  tagName,
		typeName: rawTypeName(refType),
		refType:  refType,
	}, nil
}

type structType struct {
	tagName  string
	typeName string
	refType  reflect.Type
}

// HasTrait implements the ref.Type interface method.
func (t *structType) HasTrait(trait int) bool {
	return structTypeTraitMask&trait == trait
}

// TypeName implements the ref.Type interface method.
func (t *structType) TypeName() string {
	return t.typeName
}

// ConvertToNative implements ref.Val.ConvertToNative.
func (t *structType) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return nil, fmt.Errorf("type conversion error for type to '%v'", typeDesc)
}

// ConvertToType implements ref.Val.ConvertToType.
func (t *structType) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case types.TypeType:
		return t
	}
	return types.NewErr("type conversion error from '%s' to '%s'", types.TypeType, typeVal)
}

// Equal returns true of both type names are equal to each other.
func (t *structType) Equal(other ref.Val) ref.Val {
	otherType, ok := other.(ref.Type)
	return types.Bool(ok && t.TypeName() == otherType.TypeName())
}

// Type implements the ref.Val interface method.
func (t *structType) Type() ref.Type {
	return types.TypeType
}

// Value implements the ref.Val interface method.
func (t *structType) Value() any {
	return t.typeName
}

// TagName implements the structType interface method.
func (t *structType) TagName() string {
	return t.tagName
}

// GetRawType returns the underlying reflect.Type.
func (t *structType) GetRawType() reflect.Type {
	return t.refType
}
