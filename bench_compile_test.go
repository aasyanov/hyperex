package hyperex

import (
	"context"
	"testing"
)

// =============================================================================
// COMPILATION BENCHMARKS
// =============================================================================

func BenchmarkCompile_Simple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Compile("2 + 2")
	}
}

func BenchmarkCompile_Medium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Compile("(x + y) * (z - w) / 2")
	}
}

func BenchmarkCompile_Complex(b *testing.B) {
	expr := "(a + b) * (c - d) / e + f ** 2 - g % h + (i > j ? k : l)"
	for i := 0; i < b.N; i++ {
		Compile(expr)
	}
}

func BenchmarkCompile_VeryComplex(b *testing.B) {
	expr := `user.age >= 18 && user.status == "active" && (user.role == "admin" || user.permissions.canEdit) && len(user.tags) > 0`
	for i := 0; i < b.N; i++ {
		Compile(expr)
	}
}

func BenchmarkCompile_WithOptimizer(b *testing.B) {
	engine := New(WithConstantFolding(true), WithStrengthReduction(true))
	expr := "2 + 3 * 4 - 1 + 5 * 2 + 8 / 2"
	for i := 0; i < b.N; i++ {
		engine.Compile(expr)
	}
}

func BenchmarkCompile_WithoutOptimizer(b *testing.B) {
	engine := New(WithOptimize(false))
	expr := "2 + 3 * 4 - 1 + 5 * 2 + 8 / 2"
	for i := 0; i < b.N; i++ {
		engine.Compile(expr)
	}
}

// =============================================================================
// COMPARISON: COMPILE VS EVAL
// =============================================================================

func BenchmarkCompare_CompileVsEval_Simple(b *testing.B) {
	expr := "x + y"
	env := map[string]any{"x": int64(10), "y": int64(20)}

	b.Run("Eval", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Eval(context.Background(), expr, env)
		}
	})

	b.Run("Compiled", func(b *testing.B) {
		program := MustCompile(expr)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			program.Eval(context.Background(), env)
		}
	})
}

func BenchmarkCompare_CompileVsEval_Complex(b *testing.B) {
	expr := "(a + b) * (c - d) / e + f ** 2 - g % h"
	env := map[string]any{
		"a": int64(10), "b": int64(20), "c": int64(30), "d": int64(5),
		"e": int64(5), "f": float64(3), "g": int64(17), "h": int64(5),
	}

	b.Run("Eval", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Eval(context.Background(), expr, env)
		}
	})

	b.Run("Compiled", func(b *testing.B) {
		program := MustCompile(expr)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			program.Eval(context.Background(), env)
		}
	})
}

// =============================================================================
// CONSTANT FOLDING EFFECTIVENESS
// =============================================================================

func BenchmarkOptimizer_Effectiveness(b *testing.B) {
	// Expression with many constants that can be folded
	expr := "2 + 3 * 4 - 1 + 5 * 2 + 8 / 2 - 3 + 7 * 2"

	b.Run("WithOptimizer", func(b *testing.B) {
		engine := New(WithConstantFolding(true))
		program := engine.MustCompile(expr)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			program.Eval(context.Background(), nil)
		}
	})

	b.Run("WithoutOptimizer", func(b *testing.B) {
		engine := New(WithOptimize(false))
		program := engine.MustCompile(expr)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			program.Eval(context.Background(), nil)
		}
	})
}

func BenchmarkOptimizer_StrengthReduction(b *testing.B) {
	// Expression with operations that can use strength reduction
	b.Run("WithReduction", func(b *testing.B) {
		engine := New(WithStrengthReduction(true))
		program := engine.MustCompile("x * 8 + y * 4 + z % 8")
		env := map[string]any{"x": int64(10), "y": int64(20), "z": int64(50)}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			program.Eval(context.Background(), env)
		}
	})

	b.Run("WithoutReduction", func(b *testing.B) {
		engine := New(WithStrengthReduction(false))
		program := engine.MustCompile("x * 8 + y * 4 + z % 8")
		env := map[string]any{"x": int64(10), "y": int64(20), "z": int64(50)}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			program.Eval(context.Background(), env)
		}
	})
}
