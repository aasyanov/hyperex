package hyperex

import (
	"context"
	"testing"
)

// =============================================================================
// PARALLEL BENCHMARKS
// =============================================================================

func BenchmarkParallel_Simple(b *testing.B) {
	program := MustCompile("x + y")
	b.RunParallel(func(pb *testing.PB) {
		env := map[string]any{"x": int64(10), "y": int64(20)}
		for pb.Next() {
			program.Eval(context.Background(), env)
		}
	})
}

func BenchmarkParallel_Complex(b *testing.B) {
	program := MustCompile("(a + b) * (c - d) / e + f ** 2")
	b.RunParallel(func(pb *testing.PB) {
		env := map[string]any{
			"a": int64(10), "b": int64(20), "c": int64(30),
			"d": int64(5), "e": int64(5), "f": float64(3),
		}
		for pb.Next() {
			program.Eval(context.Background(), env)
		}
	})
}

func BenchmarkParallel_Builtin(b *testing.B) {
	program := MustCompile("sum(arr)")
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	b.RunParallel(func(pb *testing.PB) {
		env := map[string]any{"arr": arr}
		for pb.Next() {
			program.Eval(context.Background(), env)
		}
	})
}

func BenchmarkParallel_CustomFunc(b *testing.B) {
	engine := New(WithFunction("double", func(x int) int { return x * 2 }))
	program := engine.MustCompile("double(x)")
	b.RunParallel(func(pb *testing.PB) {
		env := map[string]any{"x": 21}
		for pb.Next() {
			program.Eval(context.Background(), env)
		}
	})
}

// =============================================================================
// MEMORY BENCHMARKS
// =============================================================================

func BenchmarkMemory_NoAlloc(b *testing.B) {
	program := MustCompile("x + y")
	env := map[string]any{"x": int64(10), "y": int64(20)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkMemory_StringAlloc(b *testing.B) {
	program := MustCompile(`a + b + c`)
	env := map[string]any{"a": "Hello, ", "b": "World", "c": "!"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkMemory_ArrayAlloc(b *testing.B) {
	program := MustCompile("[1, 2, 3, 4, 5]")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

func BenchmarkMemory_MapAlloc(b *testing.B) {
	program := MustCompile("{a: 1, b: 2, c: 3}")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

// =============================================================================
// SANDBOX BENCHMARKS
// =============================================================================

func BenchmarkEval_WithSandbox(b *testing.B) {
	engine := New(WithSandbox(SandboxConfig{
		MaxOps:  100000,
		Timeout: 0, // No timeout for benchmark
	}))
	program := engine.MustCompile("x + y * 2")
	env := map[string]any{"x": int64(10), "y": int64(20)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_WithoutSandbox(b *testing.B) {
	engine := New()
	program := engine.MustCompile("x + y * 2")
	env := map[string]any{"x": int64(10), "y": int64(20)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Sandbox_OpsCheck(b *testing.B) {
	engine := New(WithMaxOps(1000000))
	program := engine.MustCompile("x + y + z + w")
	env := map[string]any{"x": int64(1), "y": int64(2), "z": int64(3), "w": int64(4)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// ANALYTICS BENCHMARKS
// =============================================================================

func BenchmarkAnalyze_Simple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Analyze("x + y * 2")
	}
}

func BenchmarkAnalyze_Complex(b *testing.B) {
	expr := `user.age >= 18 && user.status == "active" && sum(orders) > 100`
	for i := 0; i < b.N; i++ {
		Analyze(expr)
	}
}

func BenchmarkProfileExpr_Simple(b *testing.B) {
	env := map[string]any{"x": int64(10), "y": int64(20)}
	for i := 0; i < b.N; i++ {
		ProfileExpr(context.Background(), "x + y", env)
	}
}

// =============================================================================
// TRANSFORM BENCHMARKS
// =============================================================================

func BenchmarkSimplify_Constants(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Simplify("2 + 3 * 4 + 5 - 1")
	}
}

func BenchmarkSimplify_Algebraic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Simplify("x + 0 + y * 1 + z * 0")
	}
}

func BenchmarkExpand_Macros(b *testing.B) {
	macros := map[string]string{
		"ADULT": "age >= 18",
		"USA":   `country == "US"`,
	}
	for i := 0; i < b.N; i++ {
		Expand("ADULT && USA", macros)
	}
}

func BenchmarkInline_Let(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Inline("let x = 5; let y = x + 1; y * 2")
	}
}

func BenchmarkASTToString(b *testing.B) {
	parser := newParser("x + y * 2 - z / w")
	ast, _ := parser.parse()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ASTToString(ast)
	}
}

func BenchmarkValidate_Valid(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Validate("x + y * 2")
	}
}

func BenchmarkExtractVariables(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ExtractVariables("a + b * c - d / e")
	}
}

// =============================================================================
// vm POOL BENCHMARKS
// =============================================================================

func BenchmarkParallel_VMPool(b *testing.B) {
	program := MustCompile("sum(arr) + avg(arr)")
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	b.RunParallel(func(pb *testing.PB) {
		env := map[string]any{"arr": arr}
		for pb.Next() {
			program.Eval(context.Background(), env)
		}
	})
}

func BenchmarkParallel_HigherOrder(b *testing.B) {
	program := MustCompile("map(arr, x => x * 2)")
	arr := make([]any, 50)
	for i := range arr {
		arr[i] = int64(i)
	}
	b.RunParallel(func(pb *testing.PB) {
		env := map[string]any{"arr": arr}
		for pb.Next() {
			program.Eval(context.Background(), env)
		}
	})
}
