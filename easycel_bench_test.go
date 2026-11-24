package easycel_test

import (
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"

	"github.com/wzshiming/easycel"
)

type benchInner struct {
	Name string
}

type benchOuter struct {
	Inner benchInner
}

var benchVars = map[string]any{
	"inner": benchInner{Name: "x"},
	"outer": benchOuter{Inner: benchInner{Name: "x"}},
}

// benchProgram compiles src with every registration the benchmarks use and fails b unless it evaluates to want.
func benchProgram(b *testing.B, src string, want any) (*easycel.Environment, cel.Program) {
	r := easycel.NewRegistry("bench")
	for _, err := range []error{
		r.RegisterType(benchInner{}),
		r.RegisterType(benchOuter{}),
		r.RegisterVariable("inner", benchInner{}),
		r.RegisterVariable("outer", benchOuter{}),
		r.RegisterFunction("inc", func(x int) int { return x + 1 }),
		r.RegisterFunction("add", func(x, y int) int { return x + y }),
		r.RegisterFunction("sum", func(x, y, z int) int { return x + y + z }),
		r.RegisterFunction("addVal", func(x, y types.Int) types.Int { return x + y }),
		r.RegisterMethod("label", func(in benchInner) string { return in.Name }),
	} {
		if err != nil {
			b.Fatal(err)
		}
	}
	env, err := easycel.NewEnvironment(cel.Lib(r))
	if err != nil {
		b.Fatal(err)
	}
	prg, err := env.Program(src)
	if err != nil {
		b.Fatal(err)
	}
	got, _, err := prg.Eval(benchVars)
	if err != nil {
		b.Fatal(err)
	}
	if got.Value() != want {
		b.Fatalf("%s = %v, want %v", src, got.Value(), want)
	}
	return env, prg
}

// benchEval times the evaluation of src once benchProgram has checked its result.
func benchEval(b *testing.B, src string, want any) {
	_, prg := benchProgram(b, src, want)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prg.Eval(benchVars)
	}
}

// BenchmarkCallUnary measures a call of a Go function with one argument.
func BenchmarkCallUnary(b *testing.B) {
	benchEval(b, "inc(1000)", int64(1001))
}

// BenchmarkCallBinary measures a call of a Go function with two arguments.
func BenchmarkCallBinary(b *testing.B) {
	benchEval(b, "add(1000, 2000)", int64(3000))
}

// BenchmarkCallTernary measures a call of a Go function with three arguments.
func BenchmarkCallTernary(b *testing.B) {
	benchEval(b, "sum(1000, 2000, 3000)", int64(6000))
}

// BenchmarkCallMethod measures a Go function called as a method of a struct variable.
func BenchmarkCallMethod(b *testing.B) {
	benchEval(b, "inner.label()", "x")
}

// BenchmarkCallRefVal measures a call of a Go function whose arguments and result are cel-go values.
func BenchmarkCallRefVal(b *testing.B) {
	benchEval(b, "addVal(1000, 2000)", int64(3000))
}

// BenchmarkStructField measures a two-level field access on a struct variable.
func BenchmarkStructField(b *testing.B) {
	benchEval(b, "outer.Inner.Name", "x")
}

// BenchmarkStructLiteral measures the construction of a struct from a CEL literal.
func BenchmarkStructLiteral(b *testing.B) {
	benchEval(b, "easycel_test.benchInner{Name: 'x'}", benchInner{Name: "x"})
}

// BenchmarkCompile measures Environment.Program on a field access and a function call.
func BenchmarkCompile(b *testing.B) {
	const src = "outer.Inner.Name == 'x' && add(1, 2) == 3"
	env, _ := benchProgram(b, src, true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env.Program(src)
	}
}
