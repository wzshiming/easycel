package easycel_test

import (
	"reflect"
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/traits"

	"github.com/wzshiming/easycel"
)

type nativeStructPoint struct {
	X int `json:"x"`
}

var nativeStructRegistry = easycel.NewRegistry("test", easycel.WithTagName("json"))

func nativeStructType(t *testing.T) easycel.Type {
	t.Helper()
	typ := nativeStructRegistry.NativeToValue(nativeStructPoint{}).Type()
	got, ok := typ.(easycel.Type)
	if !ok {
		t.Fatalf("got type %T, want an easycel.Type", typ)
	}
	return got
}

func TestNativeStructObjectConvertsOnlyToItsOwnType(t *testing.T) {
	val := nativeStructRegistry.NativeToValue(nativeStructPoint{X: 1})
	if got := val.ConvertToType(val.Type()); got != val {
		t.Errorf("own type: got %v, want the value itself", got)
	}
	if got := val.ConvertToType(types.StringType); !types.IsError(got) {
		t.Errorf("string: got %v, want an error", got)
	}
}

func TestNativeStructObjectEqualComparesGoValues(t *testing.T) {
	val := nativeStructRegistry.NativeToValue(nativeStructPoint{X: 1})
	if got := val.Equal(nativeStructRegistry.NativeToValue(nativeStructPoint{X: 1})); got != types.True {
		t.Errorf("same point: got %v, want true", got)
	}
	if got := val.Equal(nativeStructRegistry.NativeToValue(nativeStructPoint{X: 2})); got != types.False {
		t.Errorf("other point: got %v, want false", got)
	}
}

func TestNativeStructTypeHasFieldTesterAndIndexerTraits(t *testing.T) {
	typ := nativeStructType(t)
	if !typ.HasTrait(traits.FieldTesterType) || !typ.HasTrait(traits.IndexerType) {
		t.Error("got no field tester or indexer trait, want both")
	}
	if typ.HasTrait(traits.AdderType) {
		t.Error("got the adder trait, want only those two")
	}
}

func TestNativeStructTypeConvertToNativeFails(t *testing.T) {
	if got, err := nativeStructType(t).ConvertToNative(reflect.TypeOf("")); got != nil || err == nil {
		t.Errorf("got %v, %v, want nil and an error", got, err)
	}
}

func TestNativeStructTypeConvertsOnlyToTypeType(t *testing.T) {
	typ := nativeStructType(t)
	if got := typ.ConvertToType(types.TypeType); got != typ {
		t.Errorf("type: got %v, want the type itself", got)
	}
	if got := typ.ConvertToType(types.StringType); !types.IsError(got) {
		t.Errorf("string: got %v, want an error", got)
	}
}

func TestNativeStructTypeEqualComparesTypeNames(t *testing.T) {
	typ := nativeStructType(t)
	if got := typ.Equal(types.NewObjectType("easycel_test.nativeStructPoint")); got != types.True {
		t.Errorf("same name: got %v, want true", got)
	}
	if got := typ.Equal(types.StringType); got != types.False {
		t.Errorf("other name: got %v, want false", got)
	}
}

func TestNativeStructTypeTypeIsTypeType(t *testing.T) {
	if got := nativeStructType(t).Type(); got != types.TypeType {
		t.Errorf("got %v, want type", got)
	}
}

func TestNativeStructTypeValueIsTypeName(t *testing.T) {
	if got := nativeStructType(t).Value(); got != "easycel_test.nativeStructPoint" {
		t.Errorf("got %v, want easycel_test.nativeStructPoint", got)
	}
}

func TestNativeStructTypeTagNameIsRegistryTagName(t *testing.T) {
	if got := nativeStructType(t).TagName(); got != "json" {
		t.Errorf("got %q, want json", got)
	}
}
