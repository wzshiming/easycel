package easycel_test

import (
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"

	"github.com/wzshiming/easycel"
)

type MappingFields struct {
	Any   any            `json:"any"`
	Dict  map[string]any `json:"dict"`
	Items []any          `json:"items"`
	Ptr   *any           `json:"ptr"`
	Func  *func()        `json:"func"`
}

func TestMappingInterfaceFieldsAreDyn(t *testing.T) {
	v := MappingFields{Any: "x", Dict: map[string]any{"k": 1}, Items: []any{"a"}}
	registry := easycel.NewRegistry("test", easycel.WithTagName("json"))
	if err := registry.RegisterType(v); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterVariable("v", v); err != nil {
		t.Fatal(err)
	}
	env, err := easycel.NewEnvironment(cel.Lib(registry))
	if err != nil {
		t.Fatal(err)
	}
	for _, src := range []string{`v.any == "x"`, `v.dict.k == 1`, `v.items[0] == "a"`} {
		program, err := env.Program(src)
		if err != nil {
			t.Errorf("%s: %v", src, err)
			continue
		}
		got, _, err := program.Eval(map[string]any{"v": v})
		if err != nil || got != types.True {
			t.Errorf("%s = %v, %v, want true", src, got, err)
		}
	}
}

func TestMappingFindStructFieldTypeHasType(t *testing.T) {
	registry := easycel.NewRegistry("test", easycel.WithTagName("json"))
	if err := registry.RegisterType(MappingFields{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"any", "dict", "items", "ptr", "func"} {
		if field, found := registry.FindStructFieldType("easycel_test.MappingFields", name); found && field.Type == nil {
			t.Errorf("FindStructFieldType(%q) found the field without a type", name)
		}
	}
}

func TestMappingNamedByteSliceValueIsBytes(t *testing.T) {
	type B []byte
	registry := easycel.NewRegistry("test")
	if err := registry.RegisterFunction("f", func() B { return B("ab") }); err != nil {
		t.Fatal(err)
	}
	env, err := easycel.NewEnvironment(cel.Lib(registry))
	if err != nil {
		t.Fatal(err)
	}
	program, err := env.Program(`f() == b"ab"`)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := program.Eval(map[string]any{})
	if err != nil || got != types.True {
		t.Errorf(`f() == b"ab" = %v, %v, want true`, got, err)
	}
}

func TestMappingSliceOfNamedByteValueIsList(t *testing.T) {
	type Byte byte
	registry := easycel.NewRegistry("test")
	if err := registry.RegisterFunction("f", func() []Byte { return []Byte{1, 2} }); err != nil {
		t.Fatal(err)
	}
	env, err := easycel.NewEnvironment(cel.Lib(registry))
	if err != nil {
		t.Fatal(err)
	}
	program, err := env.Program(`f() == [1u, 2u]`)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := program.Eval(map[string]any{})
	if err != nil || got != types.True {
		t.Errorf(`f() == [1u, 2u] = %v, %v, want true`, got, err)
	}
}
