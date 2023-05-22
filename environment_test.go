package easycel_test

import (
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/wzshiming/easycel"
)

func TestAll(t *testing.T) {
	tests := []struct {
		src         string
		conversions []any
		types       []any
		funcs       map[string][]any
		methods     map[string][]any
		vars        map[string]any
		want        any
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
			src:   "msg.next.message",
			types: []any{Message{}},
			vars: map[string]any{
				"msg": Message{
					Message: "hello",
					Next:    &Message{Message: "world"},
				},
			},
			want: types.String("world"),
		},
		{
			src: "msg",
			conversions: []any{
				func(msg Message) types.String {
					return types.String(msg.Message)
				},
			},
			vars: map[string]any{
				"msg": Message{
					Message: "hello",
				},
			},
			want: types.String("hello"),
		},
		{
			src:   "msg",
			types: []any{Message{}},
			vars: map[string]any{
				"msg": Message{
					Message: "hello",
				},
			},
			want: Message{
				Message: "hello",
			},
		},
		{
			src:   "msg.message",
			types: []any{Message{}},
			vars: map[string]any{
				"msg": Message{
					Message: "hello",
				},
			},
			want: types.String("hello"),
		},
		{
			src:   "msg.meta.name",
			types: []any{Message{}, Meta{}},
			vars: map[string]any{
				"msg": Message{
					Meta: Meta{
						Name: "Meta",
					},
				},
			},
			want: types.String("Meta"),
		},
		{
			src:   "msg.time",
			types: []any{Message{}, Meta{}},
			conversions: []any{
				func(t Timestamp) types.Timestamp {
					return types.Timestamp{t.Time}
				},
			},
			vars: map[string]any{
				"msg": Message{
					Time: Timestamp{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			want: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			src:   "msg.time.unix()",
			types: []any{Message{}, Meta{}},
			conversions: []any{
				func(t Timestamp) types.Timestamp {
					return types.Timestamp{t.Time}
				},
			},
			methods: map[string][]any{
				"unix": {
					func(t types.Timestamp) int64 {
						return t.Unix()
					},
				},
			},
			vars: map[string]any{
				"msg": Message{
					Time: Timestamp{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			want: types.Int(1577836800),
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
			src:   "Say(msg)",
			types: []any{Message{}},
			vars: map[string]any{
				"msg": Message{
					Message: "hello",
				},
			},
			funcs: map[string][]any{
				"Say": {
					func(index Message) types.String {
						return types.String(index.Message)
					},
				},
			},
			want: types.String("hello"),
		},
		{
			src: "msg.Say()",
			vars: map[string]any{
				"msg": Message{
					Message: "hello",
				},
			},
			methods: map[string][]any{
				"Say": {
					func(index Message) types.String {
						return types.String(index.Message)
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
					func(index map[ref.Val]ref.Val) Message {
						return Message{
							Message: string(index[types.String("message")].(types.String)),
						}
					},
				},
			},
			want: Message{Message: "hello"},
		},
		{
			src: "NewMessage('hello')",
			funcs: map[string][]any{
				"NewMessage": {
					func(msg types.String) Message {
						return Message{Message: string(msg)}
					},
					func(msg types.Int) Message {
						return Message{Message: strconv.FormatInt(int64(msg), 10)}
					},
				},
			},
			want: Message{Message: "hello"},
		},
		{
			src: "NewMessage('hello')",
			funcs: map[string][]any{
				"NewMessage": {
					func(msg string) Message {
						return Message{Message: string(msg)}
					},
					func(msg int) Message {
						return Message{Message: strconv.FormatInt(int64(msg), 10)}
					},
				},
			},
			want: Message{Message: "hello"},
		},
		{
			src: "NewMessage(100)",
			funcs: map[string][]any{
				"NewMessage": {
					func(msg types.String) Message {
						return Message{Message: string(msg)}
					},
					func(msg types.Int) Message {
						return Message{Message: strconv.FormatInt(int64(msg), 10)}
					},
				},
			},
			want: Message{Message: "100"},
		},
		{
			src: "time('2020-01-01T00:00:00Z')",
			funcs: map[string][]any{
				"time": {
					func(s string) (time.Time, error) {
						return time.Parse(time.RFC3339, s)
					},
				},
			},
			want: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			src: "time('2020-01-01T00:00:00Z')",
			funcs: map[string][]any{
				"time": {
					func(s string) (types.Timestamp, error) {
						t, err := time.Parse(time.RFC3339, s)
						return types.Timestamp{Time: t}, err
					},
				},
			},
			want: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			src: "time('2020-01-01T00:00:00Z').unix()",
			funcs: map[string][]any{
				"time": {
					func(s string) (types.Timestamp, error) {
						t, err := time.Parse(time.RFC3339, s)
						return types.Timestamp{Time: t}, err
					},
				},
			},
			methods: map[string][]any{
				"unix": {
					func(t types.Timestamp) int64 {
						return t.Unix()
					},
				},
			},
			want: types.Int(1577836800),
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
			registry := easycel.NewRegistry("test", easycel.WithTagName("json"))

			for _, typ := range tt.types {
				err := registry.RegisterType(typ)
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

			for _, conversion := range tt.conversions {
				err := registry.RegisterConversion(conversion)
				if err != nil {
					t.Fatal(err)
				}
			}

			for name, value := range tt.vars {
				err := registry.RegisterVariable(name, value)
				if err != nil {
					t.Fatal(err)
				}
			}
			env, err := easycel.NewEnvironment(cel.Lib(registry))
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

			if want, ok := tt.want.(ref.Val); ok {
				if got.Equal(want) != types.True {
					t.Errorf("got %#v, want %#v", got, tt.want)
				}
			} else {
				gotVal := got.Value()
				wantVal := tt.want
				if !reflect.DeepEqual(gotVal, wantVal) {
					t.Errorf("got %#v, want %#v", gotVal, wantVal)
				}
			}
		})
	}
}

type Message struct {
	Meta    Meta      `json:"meta"`
	Message string    `json:"message"`
	Next    *Message  `json:"next"`
	Time    Timestamp `json:"time"`
}

type Meta struct {
	Name string `json:"name"`
}

type Timestamp struct {
	time.Time
}
