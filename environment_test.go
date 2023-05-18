package easycel_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/go-cmp/cmp"

	"github.com/wzshiming/easycel"
)

type SayHi struct {
	Hi     string `json:"hi"`
	Target Target `json:"target"`
	Next   *SayHi `json:"next"`
}

type Target struct {
	Name string `json:"name"`
}

func TestAll(t *testing.T) {
	registry, err := easycel.NewRegistry(easycel.WithTagName("json"))
	if err != nil {
		t.Fatal(err)
	}

	global := map[string]interface{}{
		"to_timestamp": func(str string) (time.Time, error) {
			return time.Parse(time.RFC3339, str)
		},
		"vars.string": "xx",
		"vars.int":    1,
		"vars.uint":   uint(1),
		"vars.double": 1.0,
		"vars.sayhi": SayHi{
			Hi: "Hi",
			Target: Target{
				Name: "CEL",
			},
			Next: &SayHi{
				Hi: "Hello",
				Target: Target{
					Name: "Golang",
				},
			},
		},
		"SayHi": func(s SayHi) string {
			return s.Hi
		},
		"vars.map": map[string]string{
			"kv": "vv",
		},
		"vars.list":     []string{"v0", "v1"},
		"vars.to_upper": strings.ToUpper,
		"vars.add": func(s1, s2 string) string {
			return fmt.Sprintf("%s.%s", s1, s2)
		},
		"vars.fix": func() string {
			return "fix"
		},
		"xxx": map[string]interface{}{
			"yyy": map[string]interface{}{
				"zzz": map[string]interface{}{
					"kkk": "vvv",
				},
			},
		},
	}

	for n, v := range global {
		err := registry.Register(n, v)
		if err != nil {
			t.Fatal(err)
		}
	}
	env, err := easycel.NewEnvironmentWithExtensions(registry.CompileOptions()...)
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
		want           any
		wantProgramErr bool
		wantEvalErr    bool
	}{
		{
			args: args{
				src: `xxx.yyy.zzz`,
			},
			want: map[string]interface{}{
				"kkk": "vvv",
			},
		},
		{
			args: args{
				src: `vars.int + 1`,
			},
			want: int64(2),
		},
		{
			args: args{
				src: `vars.uint + uint(1)`,
			},
			want: uint64(2),
		},
		{
			args: args{
				src: `vars.double + 1.1`,
			},
			want: 2.1,
		},
		{
			args: args{
				src: `vars.string + 'str'`,
			},
			want: "xxstr",
		},
		{
			args: args{
				src: `vars.sayhi.hi`,
			},
			want: "Hi",
		},
		{
			args: args{
				src: `SayHi(vars.sayhi)`,
			},
			want: "Hi",
		},
		{
			args: args{
				src: `vars.sayhi.SayHi()`,
			},
			want: "Hi",
		},
		{
			args: args{
				src: `vars.sayhi.next.SayHi()`,
			},
			want: "Hello",
		},
		{
			args: args{
				src: `vars.sayhi.hi + " " + vars.sayhi.target.name`,
			},
			want: "Hi CEL",
		},
		{
			args: args{
				src: `size(vars.string)`,
			},
			want: int64(2),
		},
		{
			args: args{
				src: `vars.list[0]`,
			},
			want: "v0",
		},
		{
			args: args{
				src: `vars.list + ['v2']`,
			},
			want: []interface{}{"v0", "v1", "v2"},
		},
		{
			args: args{
				src: `vars.add(vars.map['kv'], 'xx')`,
			},
			want: "vv.xx",
		},
		{
			args: args{
				src: `vars.to_upper('xx')`,
			},
			want: "XX",
		},
		{
			args: args{
				src: `vars.fix()`,
			},
			want: "fix",
		},
		{
			args: args{
				src: `'z' in ['x', 'y', 'z']`,
			},
			want: true,
		},
		{
			args: args{
				src: `to_timestamp('2020-01-01T00:00:00Z')`,
			},
			want: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
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
			gotVal, _, err := prg.Eval(m)
			if (err != nil) != tt.wantEvalErr {
				t.Errorf("Eval() error = %v, wantErr %v", err, tt.wantEvalErr)
				return
			}

			if types.IsError(gotVal) {
				t.Errorf("Eval() error = %v", gotVal)
				return
			}

			got := gotVal.Value()

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Eval() diff = %v", diff)
			}
		})
	}
}

func BenchmarkAll(b *testing.B) {
	registry, err := easycel.NewRegistry()
	if err != nil {
		b.Fatal(err)
	}

	var testdatas = []string{
		`1`,
		`2 + 2`,
		`2 - 2`,
		`2 * 2`,
		`2 / 2`,
		`2 % 2`,
		`2 + 2 * 2`,
		`2 + 2 * 2 + 2`,
	}

	for _, data := range testdatas {
		b.Run(data, func(b *testing.B) {
			env, err := easycel.NewEnvironment(registry.CompileOptions()...)
			if err != nil {
				b.Fatal(err)
			}
			prg, err := env.Program(data)
			if err != nil {
				b.Fatal(err)
				return
			}

			m := map[string]interface{}{}
			for i := 0; i < b.N; i++ {
				_, _, err := prg.Eval(m)
				if err != nil {
					b.Fatal(err)
					return
				}
			}
		})
	}
}
