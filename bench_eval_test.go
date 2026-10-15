package hyperex

import (
	"context"
	"fmt"
	"testing"
)

// =============================================================================
// ARITHMETIC BENCHMARKS
// =============================================================================

func BenchmarkEval_Arithmetic_IntAdd(b *testing.B) {
	program := MustCompile("x + y")
	env := map[string]any{"x": int64(100), "y": int64(200)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Arithmetic_IntSub(b *testing.B) {
	program := MustCompile("x - y")
	env := map[string]any{"x": int64(100), "y": int64(50)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Arithmetic_IntMul(b *testing.B) {
	program := MustCompile("x * y")
	env := map[string]any{"x": int64(100), "y": int64(200)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Arithmetic_IntDiv(b *testing.B) {
	program := MustCompile("x / y")
	env := map[string]any{"x": int64(100), "y": int64(5)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Arithmetic_IntMod(b *testing.B) {
	program := MustCompile("x % y")
	env := map[string]any{"x": int64(100), "y": int64(7)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Arithmetic_Power(b *testing.B) {
	program := MustCompile("x ** y")
	env := map[string]any{"x": int64(2), "y": int64(10)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Arithmetic_Float(b *testing.B) {
	program := MustCompile("x + y * z - w / v")
	env := map[string]any{
		"x": 3.14, "y": 2.71, "z": 1.41, "w": 9.81, "v": 2.0,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Arithmetic_Mixed(b *testing.B) {
	program := MustCompile("x + y * z")
	env := map[string]any{"x": int64(10), "y": 3.14, "z": int64(2)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Arithmetic_Unary(b *testing.B) {
	program := MustCompile("-x + (-y)")
	env := map[string]any{"x": int64(100), "y": int64(50)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Arithmetic_Complex(b *testing.B) {
	program := MustCompile("(a + b) * (c - d) / e + f ** 2")
	env := map[string]any{
		"a": int64(10), "b": int64(20), "c": int64(30),
		"d": int64(5), "e": int64(5), "f": float64(3),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// COMPARISON BENCHMARKS
// =============================================================================

func BenchmarkEval_Comparison_EQ(b *testing.B) {
	program := MustCompile("x == y")
	env := map[string]any{"x": int64(100), "y": int64(100)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Comparison_NE(b *testing.B) {
	program := MustCompile("x != y")
	env := map[string]any{"x": int64(100), "y": int64(200)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Comparison_LT(b *testing.B) {
	program := MustCompile("x < y")
	env := map[string]any{"x": int64(50), "y": int64(100)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Comparison_Chain(b *testing.B) {
	program := MustCompile("x > 0 && x < 100 && y >= 50 && y <= 200")
	env := map[string]any{"x": int64(50), "y": int64(100)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Comparison_String(b *testing.B) {
	program := MustCompile(`s == "hello"`)
	env := map[string]any{"s": "hello"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// LOGICAL BENCHMARKS
// =============================================================================

func BenchmarkEval_Logical_And(b *testing.B) {
	program := MustCompile("a && b")
	env := map[string]any{"a": true, "b": true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Logical_Or(b *testing.B) {
	program := MustCompile("a || b")
	env := map[string]any{"a": false, "b": true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Logical_Not(b *testing.B) {
	program := MustCompile("!a")
	env := map[string]any{"a": true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Logical_ShortCircuit_And(b *testing.B) {
	program := MustCompile("false && expensive")
	env := map[string]any{"expensive": true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Logical_ShortCircuit_Or(b *testing.B) {
	program := MustCompile("true || expensive")
	env := map[string]any{"expensive": true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Logical_Complex(b *testing.B) {
	program := MustCompile("(a && b) || (c && d) || (!e && f)")
	env := map[string]any{
		"a": true, "b": false, "c": true, "d": true, "e": false, "f": true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// BITWISE BENCHMARKS
// =============================================================================

func BenchmarkEval_Bitwise_And(b *testing.B) {
	program := MustCompile("x & y")
	env := map[string]any{"x": int64(0xFF), "y": int64(0x0F)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Bitwise_Or(b *testing.B) {
	program := MustCompile("(x | y)")
	env := map[string]any{"x": int64(0xF0), "y": int64(0x0F)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Bitwise_Xor(b *testing.B) {
	program := MustCompile("x ^ y")
	env := map[string]any{"x": int64(0xFF), "y": int64(0x0F)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Bitwise_ShiftL(b *testing.B) {
	program := MustCompile("x << 4")
	env := map[string]any{"x": int64(1)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Bitwise_ShiftR(b *testing.B) {
	program := MustCompile("x >> 4")
	env := map[string]any{"x": int64(256)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Bitwise_Complex(b *testing.B) {
	program := MustCompile("(x & 0xFF) | (y << 8) | (z << 16)")
	env := map[string]any{"x": int64(0x12), "y": int64(0x34), "z": int64(0x56)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// REGEX (LIKE) BENCHMARKS
// =============================================================================

func BenchmarkEval_Regex_Simple(b *testing.B) {
	program := MustCompile(`text ~= "^hello"`)
	env := map[string]any{"text": "hello world"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Regex_Complex(b *testing.B) {
	program := MustCompile(`email ~= "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"`)
	env := map[string]any{"email": "test@example.com"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Regex_CachedPattern(b *testing.B) {
	// This benchmark shows the benefit of regex caching
	// Same pattern used multiple times should be much faster
	program := MustCompile(`text ~= pattern`)
	env := map[string]any{
		"text":    "hello world",
		"pattern": "^hello",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Regex_VaryingPatterns(b *testing.B) {
	// Test with varying patterns (cache miss scenario)
	program := MustCompile(`text ~= pattern`)
	patterns := []string{"^hello", "world$", "o.*d", "^h.*d$", "[aeiou]+"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := map[string]any{
			"text":    "hello world",
			"pattern": patterns[i%len(patterns)],
		}
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// STRING BENCHMARKS
// =============================================================================

func BenchmarkEval_String_Concat(b *testing.B) {
	program := MustCompile(`a + b + c`)
	env := map[string]any{"a": "Hello, ", "b": "World", "c": "!"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_String_Multiply(b *testing.B) {
	program := MustCompile(`s * 5`)
	env := map[string]any{"s": "ab"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_String_Interpolation(b *testing.B) {
	program := MustCompile(`"Hello, " + name + "! You have " + string(count) + " messages."`)
	env := map[string]any{"name": "Alice", "count": int64(5)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_String_Compare(b *testing.B) {
	program := MustCompile(`a == b`)
	env := map[string]any{"a": "hello world", "b": "hello world"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// CONTROL FLOW BENCHMARKS
// =============================================================================

func BenchmarkEval_Ternary_True(b *testing.B) {
	program := MustCompile("cond ? 1 : 2")
	env := map[string]any{"cond": true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Ternary_False(b *testing.B) {
	program := MustCompile("cond ? 1 : 2")
	env := map[string]any{"cond": false}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Ternary_Nested(b *testing.B) {
	program := MustCompile("a ? (b ? 1 : 2) : (c ? 3 : 4)")
	env := map[string]any{"a": true, "b": false, "c": true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Coalesce_First(b *testing.B) {
	program := MustCompile("x ?? y ?? z")
	env := map[string]any{"x": int64(1), "y": int64(2), "z": int64(3)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Coalesce_Last(b *testing.B) {
	program := MustCompile("x ?? y ?? z")
	env := map[string]any{"x": nil, "y": nil, "z": int64(3)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// DATA STRUCTURE BENCHMARKS
// =============================================================================

func BenchmarkEval_Array_Access(b *testing.B) {
	program := MustCompile("arr[0]")
	env := map[string]any{"arr": []any{int64(1), int64(2), int64(3)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Array_NegativeIndex(b *testing.B) {
	program := MustCompile("arr[-1]")
	env := map[string]any{"arr": []any{int64(1), int64(2), int64(3)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Array_MultiAccess(b *testing.B) {
	program := MustCompile("arr[0] + arr[1] + arr[2]")
	env := map[string]any{"arr": []any{int64(1), int64(2), int64(3)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Array_Slice(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("arr[10:20]")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Array_2D(b *testing.B) {
	program := MustCompile("matrix[1][2]")
	env := map[string]any{
		"matrix": []any{
			[]any{int64(1), int64(2), int64(3)},
			[]any{int64(4), int64(5), int64(6)},
			[]any{int64(7), int64(8), int64(9)},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Map_Access(b *testing.B) {
	program := MustCompile(`m["key"]`)
	env := map[string]any{"m": map[string]any{"key": int64(42)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Map_Member(b *testing.B) {
	program := MustCompile("obj.field")
	env := map[string]any{"obj": map[string]any{"field": int64(42)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Map_Nested(b *testing.B) {
	program := MustCompile("a.b.c.d")
	env := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": map[string]any{
					"d": int64(42),
				},
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_OptionalChain_Hit(b *testing.B) {
	program := MustCompile("obj?.field")
	env := map[string]any{"obj": map[string]any{"field": int64(42)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_OptionalChain_Miss(b *testing.B) {
	program := MustCompile("obj?.field")
	env := map[string]any{"obj": nil}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_In_Array(b *testing.B) {
	program := MustCompile("x in arr")
	env := map[string]any{"x": int64(50), "arr": []any{int64(10), int64(20), int64(50), int64(100)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_In_String(b *testing.B) {
	program := MustCompile(`sub in str`)
	env := map[string]any{"sub": "world", "str": "hello world"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_In_Map(b *testing.B) {
	program := MustCompile(`key in m`)
	env := map[string]any{"key": "foo", "m": map[string]any{"foo": 1, "bar": 2}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// ENVIRONMENT SIZE BENCHMARKS
// =============================================================================

func BenchmarkEval_Env_Small(b *testing.B) {
	program := MustCompile("a + b")
	env := map[string]any{"a": int64(1), "b": int64(2)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Env_Medium(b *testing.B) {
	program := MustCompile("a + b")
	env := make(map[string]any, 100)
	for j := 0; j < 100; j++ {
		env[fmt.Sprintf("var%d", j)] = int64(j)
	}
	env["a"] = int64(1)
	env["b"] = int64(2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Env_Large(b *testing.B) {
	program := MustCompile("a + b")
	env := make(map[string]any, 1000)
	for j := 0; j < 1000; j++ {
		env[fmt.Sprintf("var%d", j)] = int64(j)
	}
	env["a"] = int64(1)
	env["b"] = int64(2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Env_Struct(b *testing.B) {
	type Env struct {
		X int `expr:"x"`
		Y int `expr:"y"`
	}
	engine := New()
	env := Env{X: 10, Y: 20}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Eval(context.Background(), "x + y", env)
	}
}

// =============================================================================
// BITWISE NOT BENCHMARK
// =============================================================================

func BenchmarkEval_Bitwise_Not(b *testing.B) {
	program := MustCompile("~x")
	env := map[string]any{"x": int64(0xFF)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// OPERATOR OVERLOADING BENCHMARK
// =============================================================================

func BenchmarkEval_OperatorOverload(b *testing.B) {
	engine := New(
		WithFunction("customAdd", func(a, b int) int { return a + b + 1 }),
		WithOperator("+", "customAdd"),
	)
	program := engine.MustCompile("x + y")
	env := map[string]any{"x": 10, "y": 20}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}
