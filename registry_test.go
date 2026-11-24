package easycel_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/wzshiming/easycel"
)

// registryAdapter adapts every value to a CEL string that names it.
type registryAdapter struct{}

func (registryAdapter) NativeToValue(value any) ref.Val {
	return types.String(fmt.Sprint("adapted ", value))
}

// registryProvider finds every identifier; it implements no other method of types.Provider.
type registryProvider struct {
	types.Provider
}

func (registryProvider) FindIdent(identName string) (ref.Val, bool) {
	return types.String("found " + identName), true
}

type registryPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func TestRegistryWithTypeAdapterReceivesNativeToValue(t *testing.T) {
	registry := easycel.NewRegistry("test", easycel.WithTypeAdapter(registryAdapter{}))
	if got := registry.NativeToValue(1); got != types.String("adapted 1") {
		t.Errorf("got %v, want adapted 1", got)
	}
}

func TestRegistryWithTypeProviderReceivesFindIdent(t *testing.T) {
	registry := easycel.NewRegistry("test", easycel.WithTypeProvider(registryProvider{}))
	if got, found := registry.FindIdent("x"); !found || got != types.String("found x") {
		t.Errorf("got %v, %v, want found x, true", got, found)
	}
}

func TestRegistryFindStructFieldNamesUsesTagNames(t *testing.T) {
	registry := easycel.NewRegistry("test", easycel.WithTagName("json"))
	if err := registry.RegisterType(registryPoint{}); err != nil {
		t.Fatal(err)
	}
	got, found := registry.FindStructFieldNames("easycel_test.registryPoint")
	if !found || !reflect.DeepEqual(got, []string{"x", "y"}) {
		t.Errorf("got %v, %v, want [x y], true", got, found)
	}
}
