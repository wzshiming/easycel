package easycel_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/wzshiming/easycel"
)

// functionEval compiles src against a registry filled by register and evaluates it with vars.
func functionEval(t *testing.T, register func(*easycel.Registry) error, src string, vars map[string]any) (ref.Val, error) {
	t.Helper()
	registry := easycel.NewRegistry("test")
	if err := register(registry); err != nil {
		t.Fatal(err)
	}
	env, err := easycel.NewEnvironment(cel.Lib(registry))
	if err != nil {
		t.Fatal(err)
	}
	program, err := env.Program(src)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := program.Eval(vars)
	return got, err
}

func TestFunctionWithoutArguments(t *testing.T) {
	got, err := functionEval(t, func(r *easycel.Registry) error {
		return r.RegisterFunction("f", func() string { return "x" })
	}, "f()", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != types.String("x") {
		t.Errorf("got %v, want x", got)
	}
}

func TestFunctionWithThreeArguments(t *testing.T) {
	got, err := functionEval(t, func(r *easycel.Registry) error {
		return r.RegisterFunction("f", func(a, b, c string) string { return a + b + c })
	}, "f('a', 'b', 'c')", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != types.String("abc") {
		t.Errorf("got %v, want abc", got)
	}
}

func TestFunctionErrorResultFailsEvaluation(t *testing.T) {
	_, err := functionEval(t, func(r *easycel.Registry) error {
		return r.RegisterFunction("f", func(s string) (string, error) { return "", errors.New("bad " + s) })
	}, "f('x')", nil)
	if err == nil || err.Error() != "bad x" {
		t.Errorf("got error %v, want bad x", err)
	}
}

func TestFunctionNullToPointerParameterIsNil(t *testing.T) {
	tests := []struct {
		src string
		fn  any
	}{
		{"f(null)", func(p *Message) bool { return p == nil }},
		{"f('a', null)", func(a string, p *Message) bool { return p == nil }},
		{"f('a', 'b', null)", func(a, b string, p *Message) bool { return p == nil }},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			got, err := functionEval(t, func(r *easycel.Registry) error {
				return r.RegisterFunction("f", tt.fn)
			}, tt.src, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got != types.True {
				t.Errorf("got %v, want true", got)
			}
		})
	}
}

func TestFunctionNullToAnyParameterIsStructpbNullValue(t *testing.T) {
	got, err := functionEval(t, func(r *easycel.Registry) error {
		return r.RegisterFunction("f", func(a any) string { return fmt.Sprintf("%T", a) })
	}, "f(null)", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != types.String("structpb.NullValue") {
		t.Errorf("got %v, want structpb.NullValue", got)
	}
}

func TestFunctionUntypedNilArgumentIsZero(t *testing.T) {
	got, err := functionEval(t, func(r *easycel.Registry) error {
		if err := r.RegisterVariable("v", ""); err != nil {
			return err
		}
		return r.RegisterFunction("f", func(s string) bool { return s == "" })
	}, "f(v)", map[string]any{"v": convertVal{String: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != types.True {
		t.Errorf("got %v, want true", got)
	}
}

func TestFunctionVariadicTakesList(t *testing.T) {
	tests := []struct {
		src  string
		want types.Int
	}{
		{"f('a', ['b', 'c'])", 2},
		{"f('a', [])", 0},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			got, err := functionEval(t, func(r *easycel.Registry) error {
				return r.RegisterFunction("f", func(a string, rest ...string) int { return len(rest) })
			}, tt.src, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFunctionVariadicMethodTakesList(t *testing.T) {
	got, err := functionEval(t, func(r *easycel.Registry) error {
		return r.RegisterMethod("f", func(a string, rest ...string) int { return len(rest) })
	}, "'a'.f(['b', 'c'])", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != types.Int(2) {
		t.Errorf("got %v, want 2", got)
	}
}

func TestFunctionVariadicAnyReceivesElements(t *testing.T) {
	var got []any
	_, err := functionEval(t, func(r *easycel.Registry) error {
		return r.RegisterFunction("f", func(args ...any) bool {
			got = args
			return true
		})
	}, "f([1, 'b'])", nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := []any{int64(1), "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}
