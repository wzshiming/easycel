package easycel_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/wzshiming/easycel"
)

// convertVal is a CEL string whose conversion to any Go type returns nil and err.
type convertVal struct {
	types.String
	err error
}

func (v convertVal) ConvertToNative(reflect.Type) (any, error) {
	return nil, v.err
}

// convertEval evaluates src with Message registered and v declared as a CEL string bound to val.
func convertEval(t *testing.T, src string, val ref.Val) (ref.Val, error) {
	t.Helper()
	registry := easycel.NewRegistry("test", easycel.WithTagName("json"))
	if err := registry.RegisterType(Message{}); err != nil {
		t.Fatal(err)
	}
	env, err := easycel.NewEnvironment(cel.Lib(registry), cel.Variable("v", cel.StringType))
	if err != nil {
		t.Fatal(err)
	}
	program, err := env.Program(src)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := program.Eval(map[string]any{"v": val})
	return got, err
}

func TestConvertUntypedNilSetsFieldToZero(t *testing.T) {
	got, err := convertEval(t, `easycel_test.Message{message: v}.message == ""`, convertVal{String: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got != types.True {
		t.Errorf("got %v, want true", got)
	}
}

func TestConvertErrorTextIsKeptVerbatim(t *testing.T) {
	want := "conversion failed at 100%"
	_, err := convertEval(t, `easycel_test.Message{message: v}`, convertVal{err: errors.New(want)})
	if err == nil || err.Error() != want {
		t.Errorf("got error %v, want %q", err, want)
	}
}

func TestConvertNullSetsPointerFieldToNil(t *testing.T) {
	got, err := convertEval(t, `easycel_test.Message{next: null}.next == null`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != types.True {
		t.Errorf("got %v, want true", got)
	}
}

func TestConvertNullToNonPointerFieldFails(t *testing.T) {
	got, err := convertEval(t, `easycel_test.Message{meta: null}`, nil)
	if err == nil {
		t.Errorf("got %v, want an error", got)
	}
}
