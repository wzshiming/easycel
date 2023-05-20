package easycel_test

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"

	"github.com/wzshiming/easycel"
	"github.com/wzshiming/easycel/typeutils"
)

func TestAll(t *testing.T) {
	tests := []struct {
		src         string
		conversions map[reflect.Type]func(adapter ref.TypeAdapter, value reflect.Value) ref.Val
		types       map[string]any
		funcs       map[string][]any
		methods     map[string][]any
		vars        map[string]any
		want        ref.Val
	}{
		{
			src:  "1 + 1",
			want: types.Int(2),
		},
		{
			src: "1 + v",
			vars: map[string]any{
				"v": 1,
			},
			want: types.Int(2),
		},
		{
			src: "msg.next.message",
			vars: map[string]any{
				"msg": Message{
					Message: "hello",
					Next:    &Message{Message: "world"},
				},
			},
			conversions: map[reflect.Type]func(adapter ref.TypeAdapter, value reflect.Value) ref.Val{
				reflect.TypeOf(Message{}): func(adapter ref.TypeAdapter, value reflect.Value) ref.Val {
					return typeutils.NewStructWithJSONTag(adapter, value.Interface().(Message))
				},
			},
			want: types.String("world"),
		},
		{
			src: "msg",
			vars: map[string]any{
				"msg": typeutils.NewStructWithJSONTag(nil, Message{
					Message: "hello",
				}),
			},
			want: typeutils.NewStructWithJSONTag(nil, Message{
				Message: "hello",
			}),
		},
		{
			src: "msg.message",
			vars: map[string]any{
				"msg": typeutils.NewStructWithJSONTag(nil, Message{
					Message: "hello",
				}),
			},
			want: types.String("hello"),
		},
		{
			src: "sayHi('CEL')",
			funcs: map[string][]any{
				"sayHi": {
					func(s types.String) types.String {
						return "hello " + s
					},
				},
			},
			want: types.String("hello CEL"),
		},
		{
			src: "sayHi('CEL')",
			funcs: map[string][]any{
				"sayHi": {
					func(s types.String) (types.String, error) {
						return "hello " + s, nil
					},
				},
			},
			want: types.String("hello CEL"),
		},
		{
			src: "sayHi('xx') + ' ' + sayHi(1)",
			funcs: map[string][]any{
				"sayHi": {
					func(s types.String) types.String {
						return "hello " + s
					},
					func(i types.Int) types.String {
						return types.String("hello " + strconv.FormatInt(i.Value().(int64), 10))
					},
				},
			},
			want: types.String("hello xx hello 1"),
		},
		{
			src: "Say(msg)",
			vars: map[string]any{
				"msg": typeutils.NewStructWithJSONTag(nil, Message{
					Message: "hello",
				}),
			},
			funcs: map[string][]any{
				"Say": {
					func(msg typeutils.Struct[Message]) types.String {
						return types.String(msg.Value().(Message).Message)
					},
				},
			},
			want: types.String("hello"),
		},
		{
			src: "msg.Say()",
			vars: map[string]any{
				"msg": typeutils.NewStructWithJSONTag(nil, Message{
					Message: "hello",
				}),
			},
			methods: map[string][]any{
				"Say": {
					func(msg typeutils.Struct[Message]) types.String {
						return types.String(msg.Value().(Message).Message)
					},
				},
			},
			want: types.String("hello"),
		},
		{
			src:  "{'k':'v'}",
			vars: map[string]any{},
			want: types.NewStringStringMap(nil, map[string]string{"k": "v"}),
		},
		{
			src: "NewMessage({'message':'hello'})",
			funcs: map[string][]any{
				"NewMessage": {
					func(msg traits.Mapper) typeutils.Struct[Message] {
						return typeutils.NewStructWithJSONTag(nil, Message{Message: msg.Get(types.String("message")).Value().(string)})
					},
				},
			},
			want: typeutils.NewStructWithJSONTag(nil, Message{Message: "hello"}),
		},
		{
			src: "NewMessage('hello')",
			funcs: map[string][]any{
				"NewMessage": {
					func(msg types.String) typeutils.Struct[Message] {
						return typeutils.NewStructWithJSONTag(nil, Message{Message: string(msg)})
					},
					func(msg types.Int) typeutils.Struct[Message] {
						return typeutils.NewStructWithJSONTag(nil, Message{Message: strconv.FormatInt(int64(msg), 10)})
					},
				},
			},
			want: typeutils.NewStructWithJSONTag(nil, Message{Message: "hello"}),
		},
		{
			src: "NewMessage(100)",
			funcs: map[string][]any{
				"NewMessage": {
					func(msg types.String) typeutils.Struct[Message] {
						return typeutils.NewStructWithJSONTag(nil, Message{Message: string(msg)})
					},
					func(msg types.Int) typeutils.Struct[Message] {
						return typeutils.NewStructWithJSONTag(nil, Message{Message: strconv.FormatInt(int64(msg), 10)})
					},
				},
			},
			want: typeutils.NewStructWithJSONTag(nil, Message{Message: "100"}),
		},
		{
			src: "foo.bar.baz",
			vars: map[string]any{
				"foo": map[string]any{
					"bar": map[string]any{
						"baz": "hello",
					},
				},
			},
			want: types.String("hello"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			registry := easycel.NewRegistry("test")

			for name, typ := range tt.types {
				err := registry.RegisterType(name, typ)
				if err != nil {
					t.Fatal(err)
				}
			}

			for name, funcs := range tt.funcs {
				for _, fun := range funcs {
					err := registry.RegisterFunction(name, fun)
					if err != nil {
						t.Fatal(err)
					}
				}
			}

			for name, methods := range tt.methods {
				for _, method := range methods {
					err := registry.RegisterMethod(name, method)
					if err != nil {
						t.Fatal(err)
					}
				}
			}

			for name, conversion := range tt.conversions {
				err := registry.RegisterConversion(name, conversion)
				if err != nil {
					t.Fatal(err)
				}
			}

			env, err := easycel.NewEnvironmentWithExtensions(cel.Lib(registry))
			if err != nil {
				t.Fatal(err)
			}

			program, err := env.Program(tt.src)
			if err != nil {
				t.Fatal(err)
			}

			got, _, err := program.Eval(tt.vars)
			if err != nil {
				t.Fatal(err)
			}

			if got.Equal(tt.want) != types.True {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

type Message struct {
	Message string   `json:"message"`
	Next    *Message `json:"next"`
}
