package easycel

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

var (
	testType = types.NewTypeValue("test")
)

type SayHi struct {
	Hi string
}

func (c SayHi) PbType() *exprpb.Type {
	return NewObject(reflect.TypeOf(c))
}

func (c SayHi) CelMethods() []string {
	return []string{"SayHi"}
}

func (c SayHi) SayHi(m types.String) types.String {
	return types.String(fmt.Sprintf("%s %s!", c.Hi, m))
}

func (c SayHi) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	return nil, errors.New("cannot convert wrapper value to native adapter")
}

func (c SayHi) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("cannot convert wrapper value  to CEL adapter")
}

func (c SayHi) Equal(other ref.Val) ref.Val {
	if other.Type() != testType {
		return types.NewErr("cannot compare adapter")
	}
	w, ok := other.(SayHi)
	if !ok {
		return types.NewErr("cannot compare adapter")
	}

	return types.Bool(c.Hi == w.Hi)
}

func (c SayHi) Type() ref.Type {
	return testType
}

func (c SayHi) Value() interface{} {
	return c.Hi
}

func (c SayHi) CelVal() ref.Val {
	return c
}

func TestAll(t *testing.T) {

	provider := NewProvider()
	global := map[string]interface{}{
		"vars.string": types.String("xx"),
		"vars.int":    types.Int(1),
		"vars.uint":   types.Uint(1),
		"vars.double": types.Double(1),
		"vars.sayhi":  SayHi{"Hi"},
		"vars.map": types.NewStringStringMap(provider, map[string]string{
			"kv": "vv",
		}),
		"vars.list": types.NewStringList(provider, []string{"v0", "v1"}),
		"vars.to_upper": func(s1 types.String) types.String {
			return types.String(strings.ToUpper(string(s1)))
		},
		"vars.add": func(s1, s2 types.String) types.String {
			return types.String(fmt.Sprintf("%s.%s", s1, s2))
		},
		"vars.fix": func() types.String {
			return types.String("fix")
		},
	}

	for n, v := range global {
		err := provider.Registry(n, v)
		if err != nil {
			t.Fatal(err)
		}
	}
	env, err := NewEnvironment(provider.CompileOptions()...)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		src  string
		opts []cel.ProgramOption
		vars interface{}
	}
	tests := []struct {
		name           string
		args           args
		want           ref.Val
		wantProgramErr bool
		wantEvalErr    bool
	}{
		{
			args: args{
				src: `vars.int + 1`,
			},
			want: types.Int(2),
		},
		{
			args: args{
				src: `vars.uint + uint(1)`,
			},
			want: types.Uint(2),
		},
		{
			args: args{
				src: `vars.double + 1.1`,
			},
			want: types.Double(2.1),
		},
		{
			args: args{
				src: `vars.string + 'str'`,
			},
			want: types.String("xxstr"),
		},
		{
			args: args{
				src: `vars.sayhi.SayHi("world")`,
			},
			want: types.String("Hi world!"),
		},
		{
			args: args{
				src: `vars.sayhi.SayHi(vars.string)`,
			},
			want: types.String("Hi xx!"),
		},
		{
			args: args{
				src: `vars.sayhi.SayHi("world")`,
				vars: map[string]interface{}{
					"vars.sayhi": SayHi{"Hello"},
				},
			},
			want: types.String("Hello world!"),
		},
		{
			args: args{
				src: `vars.map["kv"]`,
			},
			want: types.String("vv"),
		},
		{
			args: args{
				src: `vars.map["kv0"]`,
			},
			wantEvalErr: true,
			want:        types.NewErr("no such key: kv0"),
		},
		{
			args: args{
				src: `size(vars.string)`,
			},
			want: types.Int(2),
		},
		{
			args: args{
				src: `vars.list[0]`,
			},
			want: types.String("v0"),
		},
		{
			args: args{
				src: `vars.list + ['v2']`,
			},
			want: types.NewDynamicList(provider, []interface{}{"v0", "v1", "v2"}),
		},
		{
			args: args{
				src: `vars.add(vars.map['kv'], 'xx')`,
			},
			want: types.String("vv.xx"),
		},
		{
			args: args{
				src: `vars.to_upper('xx')`,
			},
			want: types.String("XX"),
		},
		{
			args: args{
				src: `vars.fix()`,
			},
			want: types.String("fix"),
		},
		{
			args: args{
				src: `'z' in ['x', 'y', 'z']`,
			},
			want: types.True,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prg, err := env.Program(tt.args.src, tt.args.opts...)
			if (err != nil) != tt.wantProgramErr {
				t.Errorf("Program() error = %v, wantErr %v", err, tt.wantProgramErr)
				return
			}

			m := tt.args.vars
			if m == nil {
				m = map[string]interface{}{}
			}
			got, _, err := prg.Eval(m)
			if (err != nil) != tt.wantEvalErr {
				t.Errorf("Eval() error = %v, wantErr %v", err, tt.wantEvalErr)
				return
			}

			if !reflect.DeepEqual(got.Value(), tt.want.Value()) {
				t.Errorf("Eval() got = %v, want %v", got.Value(), tt.want.Value())
			}
		})
	}
}
