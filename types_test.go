package easycel

import (
	"reflect"
	"testing"

	"github.com/google/cel-go/common/types"
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
