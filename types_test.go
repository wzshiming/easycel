package easycel

import (
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

func TestGetTrait(t *testing.T) {
	type args struct {
		typ reflect.Type
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "bool",
			args: args{
				typ: reflect.TypeOf(types.Bool(false)),
			},
			want: traits.ComparerType | traits.NegatorType,
		},
		{
			name: "int",
			args: args{
				typ: reflect.TypeOf(types.Int(0)),
			},
			want: traits.AdderType | traits.ComparerType | traits.DividerType | traits.ModderType | traits.MultiplierType | traits.NegatorType | traits.SubtractorType,
		},
		{
			name: "string",
			args: args{
				typ: reflect.TypeOf(types.String("")),
			},
			want: traits.AdderType | traits.ComparerType | traits.MatcherType | traits.ReceiverType | traits.SizerType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTrait(tt.args.typ); got != tt.want {
				t.Errorf("getTrait() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mappingBytes []byte

type mappingByte byte

type mappingAdder struct{}

func (mappingAdder) Add(ref.Val) ref.Val { return nil }

// mappingTable has a Go type for each rule of the mapping, with its CEL type or nil if it is not supported.
var mappingTable = []struct {
	typ   reflect.Type
	want  *cel.Type
	trait int
}{
	{reflect.TypeOf(false), cel.BoolType, 0},
	{reflect.TypeOf(int(0)), cel.IntType, 0},
	{reflect.TypeOf(int8(0)), cel.IntType, 0},
	{reflect.TypeOf(int16(0)), cel.IntType, 0},
	{reflect.TypeOf(int32(0)), cel.IntType, 0},
	{reflect.TypeOf(int64(0)), cel.IntType, 0},
	{reflect.TypeOf(uint(0)), cel.UintType, 0},
	{reflect.TypeOf(uint8(0)), cel.UintType, 0},
	{reflect.TypeOf(uint16(0)), cel.UintType, 0},
	{reflect.TypeOf(uint32(0)), cel.UintType, 0},
	{reflect.TypeOf(uint64(0)), cel.UintType, 0},
	{reflect.TypeOf(float32(0)), cel.DoubleType, 0},
	{reflect.TypeOf(float64(0)), cel.DoubleType, 0},
	{reflect.TypeOf(""), cel.StringType, 0},
	{reflect.TypeOf(time.Duration(0)), cel.DurationType, 0},
	{reflect.TypeOf(types.Duration{}), cel.DurationType, 0},
	{reflect.TypeOf(time.Time{}), cel.TimestampType, 0},
	{reflect.TypeOf(types.Timestamp{}), cel.TimestampType, 0},
	{reflect.TypeOf([]byte(nil)), cel.BytesType, 0},
	{reflect.TypeOf(mappingBytes(nil)), cel.BytesType, 0},
	{reflect.TypeOf([]mappingByte(nil)), cel.ListType(cel.UintType), 0},
	{reflect.TypeOf([]string(nil)), cel.ListType(cel.StringType), 0},
	{reflect.TypeOf([2]byte{}), cel.ListType(cel.UintType), 0},
	{reflect.TypeOf(map[string]int(nil)), cel.MapType(cel.StringType, cel.IntType), 0},
	{reflect.TypeOf((*int)(nil)), cel.NullableType(cel.IntType), 0},
	{reflect.TypeOf(mappingAdder{}), cel.ObjectType("easycel.mappingAdder"), traits.AdderType},
	{reflect.TypeOf((*any)(nil)).Elem(), cel.DynType, 0},
	{reflect.TypeOf([]any(nil)), cel.ListType(cel.DynType), 0},
	{reflect.TypeOf(map[string]any(nil)), cel.MapType(cel.StringType, cel.DynType), 0},
	{reflect.TypeOf((chan int)(nil)), nil, 0},
	{reflect.TypeOf((func())(nil)), nil, 0},
	{reflect.TypeOf(complex64(0)), nil, 0},
	{reflect.TypeOf(complex128(0)), nil, 0},
	{reflect.TypeOf(uintptr(0)), nil, 0},
	{reflect.TypeOf(unsafe.Pointer(nil)), nil, 0},
	{reflect.TypeOf([]chan int(nil)), nil, 0},
	{reflect.TypeOf([1]func(){}), nil, 0},
	{reflect.TypeOf(map[string]complex64(nil)), nil, 0},
	{reflect.TypeOf(map[uintptr]string(nil)), nil, 0},
	{reflect.TypeOf((*func())(nil)), nil, 0},
}

func TestMappingConvertToCelType(t *testing.T) {
	for _, tt := range mappingTable {
		t.Run(tt.typ.String(), func(t *testing.T) {
			got, ok := convertToCelType(tt.typ)
			if ok != (tt.want != nil) || ok && (got.Kind() != tt.want.Kind() || got.String() != tt.want.String() || !got.HasTrait(tt.trait)) {
				t.Errorf("convertToCelType() = %v of kind %d, %v, want %v of kind %d with trait %d", got, got.Kind(), ok, tt.want, tt.want.Kind(), tt.trait)
			}
		})
	}
}

func TestMappingCachedPerGoType(t *testing.T) {
	for _, tt := range mappingTable {
		first, _ := convertToCelType(tt.typ)
		if second, _ := convertToCelType(tt.typ); second != first {
			t.Errorf("convertToCelType(%v) returned a new instance on the second call", tt.typ)
		}
	}
}

func TestMappingIsSupportedTypeMatchesMapping(t *testing.T) {
	for _, tt := range mappingTable {
		if _, ok := convertToCelType(tt.typ); isSupportedType(tt.typ) != ok {
			t.Errorf("isSupportedType(%v) = %v, convertToCelType says %v", tt.typ, !ok, ok)
		}
	}
}
