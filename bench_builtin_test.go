package hyperex

import (
	"context"
	"fmt"
	"testing"
)

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
