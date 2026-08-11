package hyperex

import (
	"context"
	"fmt"
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
// BUILTIN FUNCTION BENCHMARKS
// =============================================================================

func BenchmarkEval_Builtin_Len(b *testing.B) {
	program := MustCompile("len(arr)")
	env := map[string]any{"arr": make([]any, 100)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Abs(b *testing.B) {
	program := MustCompile("abs(x)")
	env := map[string]any{"x": int64(-42)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_MinMax(b *testing.B) {
	program := MustCompile("min(a, b, c) + max(a, b, c)")
	env := map[string]any{"a": int64(10), "b": int64(20), "c": int64(15)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Sum(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("sum(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Avg(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("avg(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Upper(b *testing.B) {
	program := MustCompile("upper(s)")
	env := map[string]any{"s": "hello world"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Contains(b *testing.B) {
	program := MustCompile(`contains(s, "world")`)
	env := map[string]any{"s": "hello world"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Trim(b *testing.B) {
	program := MustCompile("trim(s)")
	env := map[string]any{"s": "   hello world   "}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Split(b *testing.B) {
	program := MustCompile(`split(s, ",")`)
	env := map[string]any{"s": "a,b,c,d,e,f,g,h,i,j"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_First(b *testing.B) {
	arr := make([]any, 1000)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("first(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Last(b *testing.B) {
	arr := make([]any, 1000)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("last(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Sort(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(100 - i)
	}
	program := MustCompile("sort(arr)")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := map[string]any{"arr": arr}
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Reverse(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("reverse(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Unique(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i % 10) // Many duplicates
	}
	program := MustCompile("unique(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Builtin_Math(b *testing.B) {
	program := MustCompile("sqrt(x) + sin(y) + cos(z)")
	env := map[string]any{"x": 16.0, "y": 0.5, "z": 0.5}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// CUSTOM FUNCTION BENCHMARKS
// =============================================================================

func BenchmarkEval_CustomFunc_NoArgs(b *testing.B) {
	engine := New(WithFunction("answer", func() int { return 42 }))
	program := engine.MustCompile("answer()")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

func BenchmarkEval_CustomFunc_OneArg(b *testing.B) {
	engine := New(WithFunction("double", func(x int) int { return x * 2 }))
	program := engine.MustCompile("double(x)")
	env := map[string]any{"x": 21}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_CustomFunc_TwoArgs(b *testing.B) {
	engine := New(WithFunction("add", func(a, b int) int { return a + b }))
	program := engine.MustCompile("add(x, y)")
	env := map[string]any{"x": 10, "y": 20}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_CustomFunc_Variadic(b *testing.B) {
	engine := New(WithFunction("sumAll", func(nums ...int) int {
		sum := 0
		for _, n := range nums {
			sum += n
		}
		return sum
	}))
	program := engine.MustCompile("sumAll(1, 2, 3, 4, 5)")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

func BenchmarkEval_CustomFunc_String(b *testing.B) {
	engine := New(WithFunction("greet", func(name string) string {
		return "Hello, " + name + "!"
	}))
	program := engine.MustCompile(`greet(name)`)
	env := map[string]any{"name": "World"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_CustomFunc_WithError(b *testing.B) {
	engine := New(WithFunction("divide", func(a, b int) (int, error) {
		if b == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return a / b, nil
	}))
	program := engine.MustCompile("divide(x, y)")
	env := map[string]any{"x": 100, "y": 5}
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
// REAL-WORLD SCENARIO BENCHMARKS
// =============================================================================

func BenchmarkScenario_BusinessRule(b *testing.B) {
	// Typical business rule: check user eligibility
	program := MustCompile(`user.age >= 18 && user.active && user.balance > 0 && !(user.blocked)`)
	env := map[string]any{
		"user": map[string]any{
			"age":     int64(25),
			"active":  true,
			"balance": int64(100),
			"blocked": false,
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_Pricing(b *testing.B) {
	// Pricing calculation with discounts
	program := MustCompile(`
		price * quantity * (1 - discount / 100) + (quantity > 10 ? 0 : shipping)
	`)
	env := map[string]any{
		"price":    float64(29.99),
		"quantity": int64(5),
		"discount": float64(10),
		"shipping": float64(5.99),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_Validation(b *testing.B) {
	// Form validation rules
	program := MustCompile(`
		len(email) > 0 && contains(email, "@") && 
		len(password) >= 8 && 
		age >= 13 && age <= 120
	`)
	env := map[string]any{
		"email":    "user@example.com",
		"password": "securepass123",
		"age":      int64(25),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_DataTransform(b *testing.B) {
	// Data transformation
	program := MustCompile(`
		{
			fullName: user.firstName + " " + user.lastName,
			isAdult: user.age >= 18,
			score: (quiz1 + quiz2 + quiz3) / 3
		}
	`)
	env := map[string]any{
		"user": map[string]any{
			"firstName": "John",
			"lastName":  "Doe",
			"age":       int64(25),
		},
		"quiz1": float64(85),
		"quiz2": float64(90),
		"quiz3": float64(88),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_ArrayProcessing(b *testing.B) {
	// Array aggregation
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i * 10)
	}
	program := MustCompile("sum(arr) + avg(arr) + len(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_StringManipulation(b *testing.B) {
	program := MustCompile(`
		upper(trim(name)) + " - " + string(len(description)) + " chars"
	`)
	env := map[string]any{
		"name":        "  product name  ",
		"description": "This is a long product description that needs to be measured.",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
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

// =============================================================================
// NEW FEATURE BENCHMARKS
// =============================================================================

func BenchmarkEval_Let(b *testing.B) {
	program := MustCompile("let x = 10; let y = 20; x + y")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

func BenchmarkEval_LetChained(b *testing.B) {
	program := MustCompile("let a = 1; let b = a + 1; let c = b + 1; let d = c + 1; d")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

func BenchmarkEval_NotIn_Array(b *testing.B) {
	program := MustCompile("x not in arr")
	env := map[string]any{"x": int64(50), "arr": []any{int64(10), int64(20), int64(30)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_NotIn_String(b *testing.B) {
	program := MustCompile(`sub not in str`)
	env := map[string]any{"sub": "xyz", "str": "hello world"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkTypeChecker_Simple(b *testing.B) {
	tc := NewTypeChecker()
	parser := newParser("1 + 2 * 3")
	ast, _ := parser.parse()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc.Infer(ast)
	}
}

func BenchmarkTypeChecker_WithEnv(b *testing.B) {
	type Env struct {
		X int     `expr:"x"`
		Y float64 `expr:"y"`
	}
	tc := NewTypeChecker().WithEnvType(Env{})
	parser := newParser("x + int(y)")
	ast, _ := parser.parse()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc.Infer(ast)
	}
}

func BenchmarkTypeChecker_Complex(b *testing.B) {
	type Env struct {
		User struct {
			Age    int    `expr:"age"`
			Name   string `expr:"name"`
			Active bool   `expr:"active"`
		} `expr:"user"`
	}
	tc := NewTypeChecker().WithEnvType(Env{})
	parser := newParser("user.age >= 18 && user.active")
	ast, _ := parser.parse()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc.Infer(ast)
	}
}

func BenchmarkCompile_WithTypeCheck(b *testing.B) {
	type Env struct {
		X int `expr:"x"`
		Y int `expr:"y"`
	}
	engine := New(WithTypeCheck(Env{}))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Compile("x + y")
	}
}

func BenchmarkCompile_WithoutTypeCheck(b *testing.B) {
	engine := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Compile("x + y")
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
// NEW BUILTIN BENCHMARKS
// =============================================================================

func BenchmarkBuiltin_Clamp(b *testing.B) {
	program := MustCompile("clamp(x, 0, 100)")
	env := map[string]any{"x": int64(150)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Between(b *testing.B) {
	program := MustCompile("between(x, 0, 100)")
	env := map[string]any{"x": int64(50)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_HasKey(b *testing.B) {
	program := MustCompile(`hasKey(m, "key")`)
	env := map[string]any{"m": map[string]any{"key": "value", "other": 123}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Empty(b *testing.B) {
	program := MustCompile("empty(s)")
	env := map[string]any{"s": ""}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Coalesce(b *testing.B) {
	program := MustCompile("coalesce(a, b, c)")
	env := map[string]any{"a": nil, "b": nil, "c": "found"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Chunk(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("chunk(arr, 10)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Pluck(b *testing.B) {
	users := make([]any, 100)
	for i := range users {
		users[i] = map[string]any{"name": fmt.Sprintf("User%d", i), "age": int64(i + 20)}
	}
	program := MustCompile(`pluck(users, "name")`)
	env := map[string]any{"users": users}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_GroupBy(b *testing.B) {
	users := make([]any, 100)
	roles := []string{"admin", "user", "moderator", "guest"}
	for i := range users {
		users[i] = map[string]any{
			"name": fmt.Sprintf("User%d", i),
			"role": roles[i%len(roles)],
		}
	}
	program := MustCompile(`groupBy(users, u => u.role)`)
	env := map[string]any{"users": users}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_GroupBy_WithMap(b *testing.B) {
	users := make([]any, 100)
	roles := []string{"admin", "user", "moderator", "guest"}
	for i := range users {
		users[i] = map[string]any{
			"name": fmt.Sprintf("User%d", i),
			"role": roles[i%len(roles)],
		}
	}
	program := MustCompile(`groupBy(users, u => u.role) | map(g => {"role": g.key, "count": len(g.items)})`)
	env := map[string]any{"users": users}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_SortBy(b *testing.B) {
	users := make([]any, 100)
	for i := range users {
		users[i] = map[string]any{
			"name": fmt.Sprintf("User%d", i),
			"age":  int64((i * 7) % 100), // pseudo-random ages
		}
	}
	program := MustCompile(`sortBy(users, u => u.age)`)
	env := map[string]any{"users": users}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_SortBy_Descending(b *testing.B) {
	users := make([]any, 100)
	for i := range users {
		users[i] = map[string]any{
			"name": fmt.Sprintf("User%d", i),
			"age":  int64((i * 7) % 100),
		}
	}
	program := MustCompile(`sortBy(users, u => u.age, true)`)
	env := map[string]any{"users": users}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// LAMBDA & HIGHER-ORDER FUNCTION BENCHMARKS
// =============================================================================

func BenchmarkEval_Lambda_Simple(b *testing.B) {
	program := MustCompile("let f = x => x * 2; f(21)")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

func BenchmarkEval_Lambda_TwoArgs(b *testing.B) {
	program := MustCompile("let add = (a, b) => a + b; add(10, 20)")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

func BenchmarkEval_Lambda_Closure(b *testing.B) {
	program := MustCompile("let x = 10; let f = y => x + y; f(5)")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

func BenchmarkBuiltin_Map_Lambda(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("map(arr, x => x * 2)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Filter_Lambda(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("filter(arr, x => x % 2 == 0)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Reduce_Lambda(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("reduce(arr, (acc, x) => acc + x, 0)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Find_Lambda(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("find(arr, x => x > 50)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_FindIndex_Lambda(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("findIndex(arr, x => x > 50)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// PIPE OPERATOR BENCHMARKS
// =============================================================================

func BenchmarkEval_Pipe_Simple(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("arr | sum")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Pipe_Chain(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("arr | reverse | first")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// SPREAD OPERATOR BENCHMARKS
// =============================================================================

func BenchmarkEval_Spread_Simple(b *testing.B) {
	program := MustCompile("[...a, ...b]")
	env := map[string]any{
		"a": []any{int64(1), int64(2), int64(3)},
		"b": []any{int64(4), int64(5), int64(6)},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Spread_Large(b *testing.B) {
	arr1 := make([]any, 50)
	arr2 := make([]any, 50)
	for i := range arr1 {
		arr1[i] = int64(i)
		arr2[i] = int64(i + 50)
	}
	program := MustCompile("[...a, ...b]")
	env := map[string]any{"a": arr1, "b": arr2}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Spread_Mixed(b *testing.B) {
	program := MustCompile("[1, ...a, 2, ...b, 3]")
	env := map[string]any{
		"a": []any{int64(10), int64(20)},
		"b": []any{int64(30), int64(40)},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// DESTRUCTURING BENCHMARKS
// =============================================================================

func BenchmarkEval_Destructure_Array(b *testing.B) {
	program := MustCompile("let [a, b, c] = arr; a + b + c")
	env := map[string]any{"arr": []any{int64(1), int64(2), int64(3)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Destructure_Object(b *testing.B) {
	program := MustCompile("let {x, y, z} = obj; x + y + z")
	env := map[string]any{"obj": map[string]any{"x": int64(1), "y": int64(2), "z": int64(3)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Destructure_WithDefaults(b *testing.B) {
	program := MustCompile("let {x, y = 10} = obj; x + y")
	env := map[string]any{"obj": map[string]any{"x": int64(5)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Destructure_Alias(b *testing.B) {
	program := MustCompile("let {x: a, y: b} = obj; a + b")
	env := map[string]any{"obj": map[string]any{"x": int64(10), "y": int64(20)}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// RANGE OPERATOR BENCHMARKS
// =============================================================================

func BenchmarkEval_Range_Small(b *testing.B) {
	program := MustCompile("1..10")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

func BenchmarkEval_Range_Large(b *testing.B) {
	program := MustCompile("1..100")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

func BenchmarkBuiltin_Range_WithStep(b *testing.B) {
	program := MustCompile("range(0, 100, 2)")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

// =============================================================================
// ADDITIONAL BUILTIN BENCHMARKS
// =============================================================================

func BenchmarkBuiltin_Ceil(b *testing.B) {
	program := MustCompile("ceil(x)")
	env := map[string]any{"x": 3.7}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Floor(b *testing.B) {
	program := MustCompile("floor(x)")
	env := map[string]any{"x": 3.7}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Round(b *testing.B) {
	program := MustCompile("round(x)")
	env := map[string]any{"x": 3.5}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Log(b *testing.B) {
	program := MustCompile("log(x)")
	env := map[string]any{"x": 100.0}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Exp(b *testing.B) {
	program := MustCompile("exp(x)")
	env := map[string]any{"x": 2.0}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Lower(b *testing.B) {
	program := MustCompile("lower(s)")
	env := map[string]any{"s": "HELLO WORLD"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_StartsWith(b *testing.B) {
	program := MustCompile(`startsWith(s, "hello")`)
	env := map[string]any{"s": "hello world"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_EndsWith(b *testing.B) {
	program := MustCompile(`endsWith(s, "world")`)
	env := map[string]any{"s": "hello world"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Replace(b *testing.B) {
	program := MustCompile(`replace(s, "world", "Go")`)
	env := map[string]any{"s": "hello world"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Substr(b *testing.B) {
	program := MustCompile("substr(s, 0, 5)")
	env := map[string]any{"s": "hello world"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_IndexOf(b *testing.B) {
	program := MustCompile(`indexOf(s, "world")`)
	env := map[string]any{"s": "hello world"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Join(b *testing.B) {
	program := MustCompile(`join(arr, ", ")`)
	env := map[string]any{"arr": []any{"a", "b", "c", "d", "e"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Take(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("take(arr, 10)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Skip(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("skip(arr, 90)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Concat(b *testing.B) {
	arr1 := make([]any, 50)
	arr2 := make([]any, 50)
	for i := range arr1 {
		arr1[i] = int64(i)
		arr2[i] = int64(i + 50)
	}
	program := MustCompile("concat(a, b)")
	env := map[string]any{"a": arr1, "b": arr2}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Flatten(b *testing.B) {
	program := MustCompile("flatten(arr)")
	env := map[string]any{
		"arr": []any{
			[]any{int64(1), int64(2)},
			[]any{int64(3), int64(4)},
			[]any{int64(5), int64(6)},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_All(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = true
	}
	program := MustCompile("all(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Any(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = i == 50 // Only one true
	}
	program := MustCompile("any(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_None(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = false
	}
	program := MustCompile("none(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Count(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("count(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Keys(b *testing.B) {
	program := MustCompile("keys(m)")
	env := map[string]any{
		"m": map[string]any{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Values(b *testing.B) {
	program := MustCompile("values(m)")
	env := map[string]any{
		"m": map[string]any{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Type(b *testing.B) {
	program := MustCompile("type(x)")
	env := map[string]any{"x": int64(42)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_IsNil(b *testing.B) {
	program := MustCompile("isNil(x)")
	env := map[string]any{"x": nil}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Default(b *testing.B) {
	program := MustCompile("default(x, 100)")
	env := map[string]any{"x": nil}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Repeat(b *testing.B) {
	program := MustCompile(`repeat("ab", 50)`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), nil)
	}
}

func BenchmarkBuiltin_Zip(b *testing.B) {
	arr1 := make([]any, 50)
	arr2 := make([]any, 50)
	for i := range arr1 {
		arr1[i] = int64(i)
		arr2[i] = fmt.Sprintf("item%d", i)
	}
	program := MustCompile("zip(a, b)")
	env := map[string]any{"a": arr1, "b": arr2}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Compact(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		if i%3 == 0 {
			arr[i] = nil
		} else {
			arr[i] = int64(i)
		}
	}
	program := MustCompile("compact(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Truthy(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		switch i % 4 {
		case 0:
			arr[i] = nil
		case 1:
			arr[i] = false
		case 2:
			arr[i] = int64(0)
		default:
			arr[i] = int64(i)
		}
	}
	program := MustCompile("truthy(arr)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_ToJSON(b *testing.B) {
	program := MustCompile("toJSON(obj)")
	env := map[string]any{
		"obj": map[string]any{
			"name": "test",
			"age":  int64(25),
			"tags": []any{"a", "b", "c"},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_FromJSON(b *testing.B) {
	program := MustCompile(`fromJSON(s)`)
	env := map[string]any{"s": `{"name":"test","age":25,"tags":["a","b","c"]}`}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkBuiltin_Sprintf(b *testing.B) {
	program := MustCompile(`sprintf("Hello %s, you are %d years old", name, age)`)
	env := map[string]any{"name": "Alice", "age": int64(25)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
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

// =============================================================================
// UNICODE BENCHMARKS
// =============================================================================

func BenchmarkEval_Unicode_String(b *testing.B) {
	program := MustCompile("len(s)")
	env := map[string]any{"s": "Привет мир! 你好世界 🎉"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkEval_Unicode_Identifier(b *testing.B) {
	program := MustCompile("数量 + цена")
	env := map[string]any{"数量": int64(10), "цена": int64(20)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

// =============================================================================
// COMPLEX REAL-WORLD SCENARIOS
// =============================================================================

func BenchmarkScenario_FilterMapReduce(b *testing.B) {
	arr := make([]any, 100)
	for i := range arr {
		arr[i] = int64(i)
	}
	program := MustCompile("reduce(map(filter(arr, x => x % 2 == 0), x => x * 2), (a, b) => a + b, 0)")
	env := map[string]any{"arr": arr}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_PipelineProcessing(b *testing.B) {
	users := make([]any, 50)
	for i := range users {
		users[i] = map[string]any{
			"name":   fmt.Sprintf("User%d", i),
			"age":    int64(18 + i%50),
			"active": i%3 != 0,
		}
	}
	program := MustCompile(`filter(users, u => u.active && u.age >= 21) | len`)
	env := map[string]any{"users": users}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_DestructureAndCompute(b *testing.B) {
	program := MustCompile(`
		let {width, height, depth = 1} = box;
		width * height * depth
	`)
	env := map[string]any{
		"box": map[string]any{"width": int64(10), "height": int64(20)},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_SpreadAndAggregate(b *testing.B) {
	program := MustCompile("sum([...arr1, ...arr2, ...arr3])")
	env := map[string]any{
		"arr1": []any{int64(1), int64(2), int64(3)},
		"arr2": []any{int64(4), int64(5), int64(6)},
		"arr3": []any{int64(7), int64(8), int64(9)},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_ComplexValidation(b *testing.B) {
	program := MustCompile(`
		let {email, password, age, country} = user;
		len(email) > 0 && contains(email, "@") &&
		len(password) >= 8 &&
		age >= 18 && age <= 120 &&
		country in ["US", "UK", "CA", "AU"]
	`)
	env := map[string]any{
		"user": map[string]any{
			"email":    "test@example.com",
			"password": "securepass123",
			"age":      int64(25),
			"country":  "US",
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_SalesAnalytics(b *testing.B) {
	// Complex scenario: group sales by region, calculate stats, sort by total
	sales := make([]any, 100)
	regions := []string{"East", "West", "North", "South"}
	for i := range sales {
		sales[i] = map[string]any{
			"id":     int64(i),
			"region": regions[i%len(regions)],
			"amount": int64((i*17)%1000 + 100),
			"rep":    fmt.Sprintf("Rep%d", i%10),
		}
	}
	program := MustCompile(`
		groupBy(sales, s => s.region) | map(g => {
			"region": g.key,
			"total": sum(g.items | pluck("amount")),
			"count": len(g.items),
			"avg": round(avg(g.items | pluck("amount")))
		})
	`)
	env := map[string]any{"sales": sales}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_LeaderboardRanking(b *testing.B) {
	// Sort players by score descending, then take top 10
	players := make([]any, 100)
	for i := range players {
		players[i] = map[string]any{
			"name":  fmt.Sprintf("Player%d", i),
			"score": int64((i*31)%10000 + 1000),
			"level": int64(i%50 + 1),
		}
	}
	program := MustCompile(`
		sortBy(players, p => p.score, true) | take(10) | map(p => {
			"name": p.name,
			"score": p.score,
			"badge": p.score > 9000 ? "gold" : p.score > 5000 ? "silver" : "bronze"
		})
	`)
	env := map[string]any{"players": players}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_InventoryReport(b *testing.B) {
	// Group products by category, find low stock items, sort by urgency
	products := make([]any, 100)
	categories := []string{"Electronics", "Clothing", "Food", "Books", "Toys"}
	for i := range products {
		products[i] = map[string]any{
			"sku":          fmt.Sprintf("SKU%04d", i),
			"name":         fmt.Sprintf("Product%d", i),
			"category":     categories[i%len(categories)],
			"stock":        int64(i % 50),
			"reorderLevel": int64(20),
		}
	}
	program := MustCompile(`
		let lowStock = products | filter(p => p.stock <= p.reorderLevel);
		groupBy(lowStock, p => p.category) | map(g => {
			"category": g.key,
			"count": len(g.items),
			"items": sortBy(g.items, i => i.stock) | take(3) | pluck("name")
		})
	`)
	env := map[string]any{"products": products}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
	}
}

func BenchmarkScenario_UserSegmentation(b *testing.B) {
	// Segment users by activity, sort each segment, calculate metrics
	users := make([]any, 100)
	for i := range users {
		users[i] = map[string]any{
			"id":           int64(i),
			"name":         fmt.Sprintf("User%d", i),
			"lastLogin":    int64(i % 30),
			"purchaseCount": int64(i % 20),
			"totalSpent":   int64((i*37)%5000 + 100),
		}
	}
	program := MustCompile(`
		let segments = [
			{"name": "VIP", "filter": u => u.totalSpent > 3000},
			{"name": "Active", "filter": u => u.lastLogin < 7 && u.totalSpent <= 3000},
			{"name": "AtRisk", "filter": u => u.lastLogin >= 7 && u.lastLogin < 30},
			{"name": "Churned", "filter": u => u.lastLogin >= 30}
		];
		let vip = users | filter(u => u.totalSpent > 3000) | sortBy(u => u.totalSpent, true);
		{
			"vipCount": len(vip),
			"vipRevenue": sum(vip | pluck("totalSpent")),
			"topVIP": first(vip)?.name ?? "None"
		}
	`)
	env := map[string]any{"users": users}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		program.Eval(context.Background(), env)
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
