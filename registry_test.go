package easycel

import (
	"reflect"
	"testing"
)

func Benchmark_nativeCall(b *testing.B) {
	fun := func(a, b int) (int, error) {
		return a + b, nil
	}

	for i := 0; i < b.N; i++ {
		_, _ = fun(1, 2)
	}
}

func Benchmark_assertCall(b *testing.B) {
	fun := func(a, b any) (any, error) {
		return a.(int) + b.(int), nil
	}

	for i := 0; i < b.N; i++ {
		_, _ = fun(1, 2)
	}
}

func Benchmark_reflectCall(b *testing.B) {
	fun := func(a, b int) (int, error) {
		return a + b, nil
	}

	args := []reflect.Value{reflect.ValueOf(1), reflect.ValueOf(2)}
	f := reflect.ValueOf(fun)
	for i := 0; i < b.N; i++ {
		_, _ = reflectFuncCall(f, args)
	}
}
