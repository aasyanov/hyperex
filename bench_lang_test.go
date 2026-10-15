package hyperex

import (
	"context"
	"testing"
)

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

func BenchmarkTypeChecker_WithStructEnv(b *testing.B) {
	type Env struct {
		X int     `expr:"x"`
		Y float64 `expr:"y"`
	}
	tc := NewTypeChecker().WithStructEnv(Env{})
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
	tc := NewTypeChecker().WithStructEnv(Env{})
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
