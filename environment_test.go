package easycel_test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"

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
			src:   "msg.list[0].message",
			types: []any{Message{}},
			vars: map[string]any{
				"msg": Message{
					List: []*Message{
						{Message: "hello"},
					},
				},
			},
			want: types.String("hello"),
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
			src:   "unix(msg.time)",
			types: []any{Message{}, Meta{}, Timestamp{}},
			conversions: []any{
				func(t Timestamp) types.Timestamp {
					return types.Timestamp{t.Time}
				},
			},
			funcs: map[string][]any{
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
					func(s string) types.String {
						return "hello " + types.String(s)
					},
				},
			},
			want: types.String("hello CEL"),
		},
		{
			src: "sayHi('CEL')",
			funcs: map[string][]any{
				"sayHi": {
					func(s string) string {
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
					func(s *string) types.String {
						return "hello " + types.String(*s)
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
		{
			src: "Number(1) + 1",
			funcs: map[string][]any{
				"Number": {
					func(i types.Int) Number {
						return Number{int(i)}
					},
				},
			},
			types: []any{Number{}},
			want:  Number{2},
		},
		{
			src: "Number(2) - 1",
			funcs: map[string][]any{
				"Number": {
					func(i types.Int) Number {
						return Number{int(i)}
					},
				},
			},
			types: []any{Number{}},
			want:  Number{1},
		},
		{
			src: "-Number(1)",
			funcs: map[string][]any{
				"Number": {
					func(i types.Int) Number {
						return Number{int(i)}
					},
				},
			},
			types: []any{Number{}},
			want:  Number{-1},
		},
		{
			src: "Number(2) * 2",
			funcs: map[string][]any{
				"Number": {
					func(i types.Int) Number {
						return Number{int(i)}
					},
				},
			},
			types: []any{Number{}},
			want:  Number{4},
		},
		{
			src: "Number(10) / 2",
			funcs: map[string][]any{
				"Number": {
					func(i types.Int) Number {
						return Number{int(i)}
					},
				},
			},
			types: []any{Number{}},
			want:  Number{5},
		},
		{
			src: "Number(10) == Number(5)",
			funcs: map[string][]any{
				"Number": {
					func(i types.Int) Number {
						return Number{int(i)}
					},
				},
			},
			types: []any{Number{}},
			want:  types.False,
		},
		{
			src: "Number(10) != Number(5)",
			funcs: map[string][]any{
				"Number": {
					func(i types.Int) Number {
						return Number{int(i)}
					},
				},
			},
			types: []any{Number{}},
			want:  types.True,
		},
		{
			src: "Number(10) > Number(5)",
			funcs: map[string][]any{
				"Number": {
					func(i types.Int) Number {
						return Number{int(i)}
					},
				},
			},
			types: []any{Number{}},
			want:  types.True,
		},
		{
			src: "Number(10) < Number(5)",
			funcs: map[string][]any{
				"Number": {
					func(i types.Int) Number {
						return Number{int(i)}
					},
				},
			},
			types: []any{Number{}},
			want:  types.False,
		},
		{
			src: `size(m)`,
			vars: map[string]any{
				"m": Map{
					Map: map[string]string{
						"foo": "bar",
					},
				},
			},
			types: []any{Map{}},
			want:  types.Int(1),
		},
		{
			src: `"foo" in m`,
			vars: map[string]any{
				"m": Map{
					Map: map[string]string{
						"foo": "bar",
					},
				},
			},
			types: []any{Map{}},
			want:  types.True,
		},
		{
			src: `m["foo"]`,
			vars: map[string]any{
				"m": Map{
					Map: map[string]string{
						"foo": "bar",
					},
				},
			},
			types: []any{Map{}},
			want:  types.String("bar"),
		},
		// TODO
		//{
		//	src: `m.foo`,
		//	vars: map[string]any{
		//		"m": Map{
		//			Map: map[string]string{
		//				"foo": "bar",
		//			},
		//		},
		//	},
		//	types: []any{Map{}},
		//	want:  types.String("bar"),
		//},
		//{
		//	src: `has(m.foo)`,
		//	vars: map[string]any{
		//		"m": Map{
		//			Map: map[string]string{
		//				"foo": "bar",
		//			},
		//		},
		//	},
		//	types: []any{Map{}},
		//	want:  types.True,
		//},
		{
			src: `s + s`,
			vars: map[string]any{
				"s": Slice{
					Slice: []string{"foo", "bar"},
				},
			},
			types: []any{Slice{}},
			want: Slice{
				Slice: []string{"foo", "bar", "foo", "bar"},
			},
		},
		{
			src: `size(s)`,
			vars: map[string]any{
				"s": Slice{
					Slice: []string{"foo", "bar"},
				},
			},
			types: []any{Slice{}},
			want:  types.Int(2),
		},
		{
			src: `"foo" in s`,
			vars: map[string]any{
				"s": Slice{
					Slice: []string{"foo", "bar"},
				},
			},
			types: []any{Slice{}},
			want:  types.True,
		},
		{
			src: `s[0]`,
			vars: map[string]any{
				"s": Slice{
					Slice: []string{"foo", "bar"},
				},
			},
			types: []any{Slice{}},
			want:  types.String("foo"),
		},
		{
			src:   `easycel_test.Message{ message: "hello" }`,
			types: []any{Message{}},
			want:  Message{Message: "hello"},
		},
		{
			src:   `easycel_test.Message{ message: "hello" }.message`,
			types: []any{Message{}},
			want:  types.String("hello"),
		},
		// TODO
		//{
		//	src:   `easycel_test.Message{ message: "hello" }["message"]`,
		//	types: []any{Message{}},
		//	want:  types.String("hello"),
		//},
		{
			src:   `{ "message": "hello" }`,
			types: []any{Message{}},
			want: map[ref.Val]ref.Val{
				types.String("message"): types.String("hello"),
			},
		},
		{
			src:   `{ "message": "hello" }.message`,
			types: []any{Message{}},
			want:  types.String("hello"),
		},
		{
			src:   `{ "message": "hello" }["message"]`,
			types: []any{Message{}},
			want:  types.String("hello"),
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
	Meta    Meta       `json:"meta"`
	Message string     `json:"message"`
	Next    *Message   `json:"next"`
	Time    Timestamp  `json:"time"`
	List    []*Message `json:"list"`
	_       int
}

type Meta struct {
	Name string `json:"name"`
}

type Timestamp struct {
	time.Time
}

type Number struct {
	Number int
}

var (
	NumberType = cel.ObjectType("my_number",
		traits.AdderType,
		traits.SubtractorType,
		traits.NegatorType,
		traits.MultiplierType,
		traits.DividerType,
		traits.ComparerType,
	)
)

func (n Number) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return nil, fmt.Errorf("unsupported conversion from 'number' to '%v'", typeDesc)
}

func (n Number) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("type conversion error from '%s' to '%s'", NumberType, typeValue)
}

func (n Number) Equal(other ref.Val) ref.Val {
	v, ok := other.(Number)
	if !ok {
		return types.False
	}
	return types.Bool(v.Number == n.Number)
}

func (n Number) Type() ref.Type {
	return NumberType
}

func (n Number) Value() any {
	return n.Number
}

func (n Number) Add(other ref.Val) ref.Val {
	return Number{n.Number + int(other.(types.Int))}
}

func (n Number) Subtract(other ref.Val) ref.Val {
	return Number{n.Number - int(other.(types.Int))}
}

func (n Number) Negate() ref.Val {
	return Number{-n.Number}
}

func (n Number) Multiply(other ref.Val) ref.Val {
	return Number{n.Number * int(other.(types.Int))}
}

func (n Number) Divide(other ref.Val) ref.Val {
	return Number{n.Number / int(other.(types.Int))}
}

func (n Number) Compare(other ref.Val) ref.Val {
	v, ok := other.(Number)
	if !ok {
		return types.MaybeNoSuchOverloadErr(other)
	}
	if n.Number < v.Number {
		return types.Int(-1)
	}
	if n.Number > v.Number {
		return types.Int(1)
	}
	return types.Int(0)
}

var (
	MapType = cel.ObjectType("my_map",
		traits.ContainerType,
		traits.IndexerType,
		traits.SizerType,
	)
)

type Map struct {
	Map map[string]string
}

func (m Map) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return nil, fmt.Errorf("unsupported conversion from 'map' to '%v'", typeDesc)
}

func (m Map) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("type conversion error from '%s' to '%s'", MapType, typeValue)
}

func (m Map) Equal(other ref.Val) ref.Val {
	v, ok := other.(Map)
	if !ok {
		return types.False
	}
	return types.Bool(reflect.DeepEqual(v.Map, m.Map))
}

func (m Map) Type() ref.Type {
	return MapType
}

func (m Map) Value() any {
	return m.Map
}

func (m Map) Size() ref.Val {
	return types.Int(len(m.Map))
}

func (m Map) Contains(index ref.Val) ref.Val {
	key, ok := index.(types.String)
	if !ok {
		return types.MaybeNoSuchOverloadErr(index)
	}
	_, found := m.Map[string(key)]
	return types.Bool(found)
}

func (m Map) Get(index ref.Val) ref.Val {
	key, ok := index.(types.String)
	if !ok {
		return types.MaybeNoSuchOverloadErr(index)
	}
	val, found := m.Map[string(key)]
	if !found {
		return types.String("")
	}
	return types.String(val)
}

var (
	SliceType = cel.ObjectType("my_slice",
		traits.AdderType,
		traits.ContainerType,
		traits.IndexerType,
		traits.SizerType,
	)
)

type Slice struct {
	Slice []string
}

func (s Slice) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return nil, fmt.Errorf("unsupported conversion from 'slice' to '%v'", typeDesc)
}

func (s Slice) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("type conversion error from '%s' to '%s'", SliceType, typeValue)
}

func (s Slice) Equal(other ref.Val) ref.Val {
	v, ok := other.(Slice)
	if !ok {
		return types.False
	}
	return types.Bool(reflect.DeepEqual(v.Slice, s.Slice))
}

func (s Slice) Type() ref.Type {
	return SliceType
}

func (s Slice) Value() any {
	return s.Slice
}

func (s Slice) Size() ref.Val {
	return types.Int(len(s.Slice))
}

func (s Slice) Add(other ref.Val) ref.Val {
	v, ok := other.(Slice)
	if !ok {
		return types.MaybeNoSuchOverloadErr(other)
	}
	return Slice{append(s.Slice, v.Slice...)}
}

func (s Slice) Contains(index ref.Val) ref.Val {
	key, ok := index.(types.String)
	if !ok {
		return types.MaybeNoSuchOverloadErr(index)
	}
	for _, v := range s.Slice {
		if v == string(key) {
			return types.True
		}
	}
	return types.False
}

func (s Slice) Get(index ref.Val) ref.Val {
	key, ok := index.(types.Int)
	if !ok {
		return types.MaybeNoSuchOverloadErr(index)
	}
	if int(key) < len(s.Slice) {
		return types.String(s.Slice[int(key)])
	}
	return types.String("")
}
