package hyperex

import (
	"context"
	"math"
	"strings"
	"testing"
)

// =============================================================================
// BUILTIN FUNCTION TESTS
// =============================================================================

func TestBuiltin_Math(t *testing.T) {
	tests := []struct {
		expr   string
		result float64
	}{
		{"abs(-5)", 5},
		{"abs(5)", 5},
		{"abs(-3.14)", 3.14},
		{"ceil(3.2)", 4},
		{"ceil(3.9)", 4},
		{"floor(3.9)", 3},
		{"floor(3.2)", 3},
		{"round(3.4)", 3},
		{"round(3.5)", 4},
		{"sqrt(16)", 4},
		{"sqrt(2)", 1.4142135623730951},
		{"pow(2, 3)", 8},
		{"pow(4, 0.5)", 2},
		{"log(1)", 0},
		{"exp(0)", 1},
		{"sin(0)", 0},
		{"cos(0)", 1},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			got := NewValue(result).Float()
			if got != tt.result {
				t.Errorf("got %v, want %v", got, tt.result)
			}
		})
	}
}

func TestBuiltin_MinMax(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"min(1, 2, 3)", 1},
		{"min([3, 1, 2])", 1},
		{"max(1, 2, 3)", 3},
		{"max([1, 3, 2])", 3},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Int() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestBuiltin_Strings(t *testing.T) {
	tests := []struct {
		expr   string
		result any
	}{
		{`len("hello")`, int64(5)},
		{`upper("hello")`, "HELLO"},
		{`lower("HELLO")`, "hello"},
		{`trim("  hello  ")`, "hello"},
		{`trimLeft("  hello")`, "hello"},
		{`trimRight("hello  ")`, "hello"},
		{`trimLeft("xxhello", "x")`, "hello"},
		{`split("a,b,c", ",")`, []any{"a", "b", "c"}},
		{`contains("hello", "ell")`, true},
		{`contains("hello", "xyz")`, false},
		{`startsWith("hello", "he")`, true},
		{`startsWith("hello", "lo")`, false},
		{`endsWith("hello", "lo")`, true},
		{`endsWith("hello", "he")`, false},
		{`replace("hello", "l", "L")`, "heLLo"},
		{`substr("hello", 1)`, "ello"},
		{`substr("hello", 1, 3)`, "ell"},
		{`substr("hello", -2)`, "lo"},
		{`indexOf("hello", "l")`, int64(2)},
		{`indexOf("hello", "x")`, int64(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			switch expected := tt.result.(type) {
			case []any:
				arr := result.([]any)
				if len(arr) != len(expected) {
					t.Errorf("got %v, want %v", result, tt.result)
				}
			default:
				if result != tt.result {
					t.Errorf("got %v, want %v", result, tt.result)
				}
			}
		})
	}
}

func TestBuiltin_Join(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{"a", "b", "c"},
	}
	result, err := engine.Eval(context.Background(), `join(arr, "-")`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "a-b-c" {
		t.Errorf("got %v, want a-b-c", result)
	}
}

func TestBuiltin_Arrays(t *testing.T) {
	tests := []struct {
		name  string
		expr  string
		check func(any) bool
	}{
		{"first", "first([1, 2, 3])", func(v any) bool { return NewValue(v).Int() == 1 }},
		{"last", "last([1, 2, 3])", func(v any) bool { return NewValue(v).Int() == 3 }},
		{"first_empty", "first([])", func(v any) bool { return v == nil }},
		{"last_empty", "last([])", func(v any) bool { return v == nil }},
		{"take", "take([1, 2, 3, 4], 2)", func(v any) bool { return len(v.([]any)) == 2 }},
		{"take_more", "take([1, 2], 5)", func(v any) bool { return len(v.([]any)) == 2 }},
		{"skip", "skip([1, 2, 3, 4], 2)", func(v any) bool { return len(v.([]any)) == 2 }},
		{"skip_all", "skip([1, 2], 5)", func(v any) bool { return len(v.([]any)) == 0 }},
		{"reverse", "reverse([1, 2, 3])", func(v any) bool {
			arr := v.([]any)
			return len(arr) == 3 && arr[0] == int64(3)
		}},
		{"sort", "sort([3, 1, 2])", func(v any) bool {
			arr := v.([]any)
			return len(arr) == 3 && arr[0] == int64(1)
		}},
		{"unique", "unique([1, 2, 2, 3, 3, 3])", func(v any) bool { return len(v.([]any)) == 3 }},
		{"concat", "concat([1, 2], [3, 4])", func(v any) bool { return len(v.([]any)) == 4 }},
		{"concat_single", "concat([1], 2, [3])", func(v any) bool { return len(v.([]any)) == 3 }},
		{"flatten", "flatten([[1, 2], [3, [4, 5]]])", func(v any) bool { return len(v.([]any)) == 5 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if !tt.check(result) {
				t.Errorf("check failed for %v", result)
			}
		})
	}
}

func TestBuiltin_Aggregation(t *testing.T) {
	tests := []struct {
		expr   string
		result any
	}{
		{"sum([1, 2, 3, 4, 5])", int64(15)},
		{"sum([1.5, 2.5])", float64(4.0)},
		// Test overflow handling: sum should switch to float on overflow (consistent with + operator)
		{"sum([9223372036854775807, 1])", float64(9223372036854775808.0)}, // MaxInt64 + 1 overflows
		{"avg([2, 4, 6])", float64(4.0)},
		{"avg([])", nil},
		{"count([1, 2, 3])", int64(3)},
		{"all([true, true, true])", true},
		{"all([true, false, true])", false},
		{"all([1, 2, 3])", true},
		{"all([1, 0, 3])", false},
		{"any([false, false, true])", true},
		{"any([false, false, false])", false},
		{"none([false, false, false])", true},
		{"none([false, true, false])", false},
		{"one([false, true, false])", true},
		{"one([true, true, false])", false},
		{"one([false, false, false])", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.result, tt.result)
			}
		})
	}
}

func TestBuiltin_TypeConversion(t *testing.T) {
	tests := []struct {
		expr   string
		result any
	}{
		{`int(3.7)`, int64(3)},
		{`int("42")`, int64(42)},
		{`float(3)`, float64(3)},
		{`float("3.14")`, float64(3.14)},
		{`string(42)`, "42"},
		{`string(3.14)`, "3.14"},
		{`string(true)`, "true"},
		{`bool(1)`, true},
		{`bool(0)`, false},
		{`bool("")`, false},
		{`bool("hi")`, true},
		{`type(42)`, "int"},
		{`type("hello")`, "string"},
		{`type([1, 2])`, "array"},
		{`type({foo: 1})`, "map"},
		{`type(nil)`, "nil"},
		{`isNil(nil)`, true},
		{`isNil(0)`, false},
		{`isNil("")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestBuiltin_KeysValues(t *testing.T) {
	engine := New()
	env := map[string]any{
		"m": map[string]any{"a": 1, "b": 2},
	}

	// Test keys
	result, err := engine.Eval(context.Background(), "keys(m)", env)
	if err != nil {
		t.Fatalf("keys error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 2 {
		t.Errorf("keys: got %d items", len(arr))
	}

	// Test values
	result, err = engine.Eval(context.Background(), "values(m)", env)
	if err != nil {
		t.Fatalf("values error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 2 {
		t.Errorf("values: got %d items", len(arr))
	}
}

func TestBuiltin_Sprintf(t *testing.T) {
	result, err := Eval(context.Background(), `sprintf("Hello, %s! You have %d messages.", "Alice", 5)`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "Hello, Alice! You have 5 messages." {
		t.Errorf("got %v", result)
	}
}

func TestBuiltin_ErrorCases(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"abs_no_args", "abs()"},
		{"min_no_args", "min()"},
		{"len_no_args", "len()"},
		{"first_not_array", "first(42)"},
		{"sum_not_array", "sum(42)"},
		{"keys_not_map", "keys([1,2])"},
		{"values_not_map", "values([1,2])"},
		{"join_not_array", "join(42, \"-\")"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Eval(context.Background(), tt.expr)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

// =============================================================================
// NEW BUILTIN FUNCTIONS TESTS
// =============================================================================

func TestBuiltin_Clamp(t *testing.T) {
	tests := []struct {
		expr     string
		expected int64
	}{
		{"clamp(5, 0, 10)", 5},
		{"clamp(-5, 0, 10)", 0},
		{"clamp(15, 0, 10)", 10},
		{"clamp(0, 0, 10)", 0},
		{"clamp(10, 0, 10)", 10},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalInt(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_Between(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{"between(5, 0, 10)", true},
		{"between(-5, 0, 10)", false},
		{"between(15, 0, 10)", false},
		{"between(0, 0, 10)", true},
		{"between(10, 0, 10)", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalBool(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_HasKey(t *testing.T) {
	tests := []struct {
		expr     string
		env      map[string]any
		expected bool
	}{
		{`hasKey(m, "a")`, map[string]any{"m": map[string]any{"a": 1, "b": 2}}, true},
		{`hasKey(m, "c")`, map[string]any{"m": map[string]any{"a": 1, "b": 2}}, false},
		{`hasKey(m, "")`, map[string]any{"m": map[string]any{"": "empty key"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalBool(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_Empty(t *testing.T) {
	tests := []struct {
		expr     string
		env      map[string]any
		expected bool
	}{
		{`empty("")`, nil, true},
		{`empty("hello")`, nil, false},
		{`empty([])`, nil, true},
		{`empty([1, 2])`, nil, false},
		{`empty(x)`, map[string]any{"x": nil}, true},
		{`empty(x)`, map[string]any{"x": map[string]any{}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalBool(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_NotEmpty(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{`notEmpty("")`, false},
		{`notEmpty("hello")`, true},
		{`notEmpty([])`, false},
		{`notEmpty([1, 2])`, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalBool(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_Default(t *testing.T) {
	tests := []struct {
		expr     string
		env      map[string]any
		expected int64
	}{
		{`default(x, 42)`, map[string]any{"x": nil}, 42},
		{`default(x, 42)`, map[string]any{"x": 10}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalInt(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_Coalesce(t *testing.T) {
	tests := []struct {
		expr     string
		env      map[string]any
		expected string
	}{
		{`coalesce(a, b, "default")`, map[string]any{"a": nil, "b": nil}, "default"},
		{`coalesce(a, b, "default")`, map[string]any{"a": nil, "b": "found"}, "found"},
		{`coalesce(a, b, "default")`, map[string]any{"a": "first", "b": "second"}, "first"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalString(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_Repeat(t *testing.T) {
	result, err := Eval(context.Background(), `repeat("x", 3)`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr))
	}
	for _, v := range arr {
		if NewValue(v).String() != "x" {
			t.Errorf("expected 'x', got %v", v)
		}
	}
}

func TestBuiltin_Range(t *testing.T) {
	tests := []struct {
		expr     string
		expected []int64
	}{
		{"range(1, 5)", []int64{1, 2, 3, 4, 5}},
		{"range(0, 3)", []int64{0, 1, 2, 3}},
		{"range(1, 10, 2)", []int64{1, 3, 5, 7, 9}},
		{"range(10, 1, -2)", []int64{10, 8, 6, 4, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			arr := NewValue(result).Array()
			if len(arr) != len(tt.expected) {
				t.Errorf("length: got %d, want %d", len(arr), len(tt.expected))
				return
			}
			for i, v := range arr {
				if NewValue(v).Int() != tt.expected[i] {
					t.Errorf("[%d]: got %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestBuiltin_Zip(t *testing.T) {
	result, err := Eval(context.Background(), `zip([1, 2, 3], ["a", "b", "c"])`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	if len(arr) != 3 {
		t.Errorf("expected 3 tuples, got %d", len(arr))
	}
	// First tuple should be [1, "a"]
	tuple := NewValue(arr[0]).Array()
	if len(tuple) != 2 {
		t.Errorf("expected tuple of 2, got %d", len(tuple))
	}
}

func TestBuiltin_Chunk(t *testing.T) {
	result, err := Eval(context.Background(), `chunk([1, 2, 3, 4, 5], 2)`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	if len(arr) != 3 { // [[1,2], [3,4], [5]]
		t.Errorf("expected 3 chunks, got %d", len(arr))
	}
}

func TestBuiltin_Compact(t *testing.T) {
	// compact removes only nil values
	result, err := Eval(context.Background(), `compact([1, nil, 2, false, 3, "", 0])`, map[string]any{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	// Should keep: 1, 2, false, 3, "", 0 (removes only nil)
	if len(arr) != 6 {
		t.Errorf("expected 6 elements, got %d: %v", len(arr), arr)
	}
}

func TestBuiltin_Truthy(t *testing.T) {
	// truthy removes all falsy values (nil, false, 0, "")
	result, err := Eval(context.Background(), `truthy([1, nil, 2, false, 3, "", 0])`, map[string]any{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	// Should keep: 1, 2, 3 (removes nil, false, "", 0)
	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got %d: %v", len(arr), arr)
	}
}

func TestBuiltin_Pluck(t *testing.T) {
	env := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "age": 30},
			map[string]any{"name": "Bob", "age": 25},
			map[string]any{"name": "Charlie", "age": 35},
		},
	}
	result, err := Eval(context.Background(), `pluck(users, "name")`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	if len(arr) != 3 {
		t.Errorf("expected 3 names, got %d", len(arr))
	}
	if NewValue(arr[0]).String() != "Alice" {
		t.Errorf("expected Alice, got %v", arr[0])
	}
}

func TestBuiltin_GroupBy(t *testing.T) {
	env := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "role": "admin"},
			map[string]any{"name": "Bob", "role": "user"},
			map[string]any{"name": "Charlie", "role": "admin"},
			map[string]any{"name": "Diana", "role": "user"},
		},
	}

	// Basic groupBy
	result, err := Eval(context.Background(), `groupBy(users, u => u.role)`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	if len(arr) != 2 {
		t.Errorf("expected 2 groups, got %d", len(arr))
	}

	// GroupBy with map transformation
	result2, err := Eval(context.Background(), `groupBy(users, u => u.role) | map(g => {"role": g.key, "count": len(g.items)})`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr2 := NewValue(result2).Array()
	if len(arr2) != 2 {
		t.Errorf("expected 2 groups, got %d", len(arr2))
	}

	// Check first group
	first := NewValue(arr2[0]).Map()
	if first["role"] != "admin" {
		t.Errorf("expected first role to be admin, got %v", first["role"])
	}
	if NewValue(first["count"]).Int() != 2 {
		t.Errorf("expected admin count to be 2, got %v", first["count"])
	}
}

func TestBuiltin_GroupBy_Errors(t *testing.T) {
	// Not an array
	_, err := Eval(context.Background(), `groupBy(123, x => x)`)
	if err == nil {
		t.Error("expected error for non-array")
	}

	// Not a function
	_, err = Eval(context.Background(), `groupBy([1,2,3], "not a function")`)
	if err == nil {
		t.Error("expected error for non-function")
	}
}

func TestBuiltin_SortBy(t *testing.T) {
	env := map[string]any{
		"users": []any{
			map[string]any{"name": "Charlie", "age": 30},
			map[string]any{"name": "Alice", "age": 25},
			map[string]any{"name": "Bob", "age": 35},
		},
	}

	// Sort ascending by age
	result, err := Eval(context.Background(), `sortBy(users, u => u.age) | pluck("name")`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	expected := []string{"Alice", "Charlie", "Bob"}
	for i, name := range expected {
		if NewValue(arr[i]).String() != name {
			t.Errorf("position %d: expected %s, got %v", i, name, arr[i])
		}
	}

	// Sort descending by age
	result2, err := Eval(context.Background(), `sortBy(users, u => u.age, true) | pluck("name")`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr2 := NewValue(result2).Array()
	expected2 := []string{"Bob", "Charlie", "Alice"}
	for i, name := range expected2 {
		if NewValue(arr2[i]).String() != name {
			t.Errorf("position %d: expected %s, got %v", i, name, arr2[i])
		}
	}

	// Sort by string field
	result3, err := Eval(context.Background(), `sortBy(users, u => u.name) | pluck("name")`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr3 := NewValue(result3).Array()
	expected3 := []string{"Alice", "Bob", "Charlie"}
	for i, name := range expected3 {
		if NewValue(arr3[i]).String() != name {
			t.Errorf("position %d: expected %s, got %v", i, name, arr3[i])
		}
	}
}

func TestBuiltin_SortBy_Errors(t *testing.T) {
	// Not an array
	_, err := Eval(context.Background(), `sortBy(123, x => x)`)
	if err == nil {
		t.Error("expected error for non-array")
	}

	// Not a function
	_, err = Eval(context.Background(), `sortBy([1,2,3], "not a function")`)
	if err == nil {
		t.Error("expected error for non-function")
	}

	// Wrong arg count
	_, err = Eval(context.Background(), `sortBy([1,2,3])`)
	if err == nil {
		t.Error("expected error for wrong arg count")
	}
}

func TestBuiltin_ToJSON(t *testing.T) {
	result, err := EvalString(context.Background(), `toJSON({"a": 1, "b": 2})`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Result should be valid JSON
	if !strings.Contains(result, `"a"`) || !strings.Contains(result, `"b"`) {
		t.Errorf("expected JSON with a and b keys, got %s", result)
	}
}

func TestBuiltin_FromJSON(t *testing.T) {
	result, err := Eval(context.Background(), `fromJSON('{"name": "test", "value": 42}')`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	m := NewValue(result).Map()
	if m["name"] != "test" {
		t.Errorf("expected name=test, got %v", m["name"])
	}
}

func TestBuiltin_SizeRemoved(t *testing.T) {
	if IsBuiltin("size") {
		t.Fatal("size must not be a builtin")
	}
	_, err := Eval(context.Background(), `size([1, 2, 3])`)
	if err == nil {
		t.Fatal("expected undefined size")
	}
}

func TestAST_WalkAllNodes(t *testing.T) {
	// Parse a complex expression to cover more Walk paths
	exprs := []string{
		"[1, 2, 3]",
		"{a: 1, b: 2}",
		"(1..5)",
		"a ?? b",
		"x ? 1 : 2",
	}

	for _, expr := range exprs {
		parser := newParser(expr)
		ast, err := parser.parse()
		if err != nil {
			continue // Some may not parse
		}

		visited := false
		Walk(ast, func(n Node) bool {
			visited = true
			return true
		})

		if !visited {
			t.Errorf("Walk didn't visit any nodes for %s", expr)
		}
	}
}

func TestAST_Transform(t *testing.T) {
	parser := newParser("1 + 2")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Double all integers
	result := Transform(ast, func(n Node) Node {
		if lit, ok := n.(*intLit); ok {
			return &intLit{Value: lit.Value * 2}
		}
		return n
	})

	// Verify transformation
	bin, ok := result.(*binaryExpr)
	if !ok {
		t.Fatal("expected binaryExpr")
	}
	left, ok := bin.Left.(*intLit)
	if !ok || left.Value != 2 {
		t.Errorf("left: got %v, want 2", left)
	}
	right, ok := bin.Right.(*intLit)
	if !ok || right.Value != 4 {
		t.Errorf("right: got %v, want 4", right)
	}
}

func TestAST_MoreNodeStrings(t *testing.T) {
	nodes := []Node{
		&callExpr{Func: &ident{Name: "foo"}, Args: []Node{&intLit{Value: 1}}},
		&memberExpr{Object: &ident{Name: "obj"}, Property: &ident{Name: "field"}},
		&indexExpr{Object: &ident{Name: "arr"}, Index: &intLit{Value: 0}},
		&sliceExpr{Object: &ident{Name: "arr"}, Low: &intLit{Value: 0}, High: &intLit{Value: 2}},
		&pipeExpr{Left: &ident{Name: "arr"}, Right: &ident{Name: "sum"}},
		&lambdaExpr{Params: []string{"x"}, Body: &ident{Name: "x"}},
		&letExpr{Name: "x", Value: &intLit{Value: 5}, Body: &ident{Name: "x"}},
	}

	for _, n := range nodes {
		s := n.String()
		if s == "" {
			t.Errorf("empty string for %T", n)
		}
	}
}

func TestValue_MapEquals(t *testing.T) {
	m1 := Map(map[string]any{"a": int64(1), "b": int64(2)})
	m2 := Map(map[string]any{"a": int64(1), "b": int64(2)})
	m3 := Map(map[string]any{"a": int64(1), "c": int64(3)})
	m4 := Map(map[string]any{"a": int64(1)})

	if !m1.Equal(m2) {
		t.Error("m1 should equal m2")
	}
	if m1.Equal(m3) {
		t.Error("m1 should not equal m3")
	}
	if m1.Equal(m4) {
		t.Error("m1 should not equal m4 (different size)")
	}
}

func TestValue_Interface(t *testing.T) {
	tests := []struct {
		value    Value
		expected any
	}{
		{Nil, nil},
		{Bool(true), true},
		{Int(42), int64(42)},
		{Float(3.14), 3.14},
		{String("hello"), "hello"},
	}

	for _, tt := range tests {
		if tt.value.Interface() != tt.expected {
			t.Errorf("got %v, want %v", tt.value.Interface(), tt.expected)
		}
	}
}

func TestValue_ArrayMap(t *testing.T) {
	arr := Array([]any{1, 2, 3})
	if arr.Array() == nil {
		t.Error("Array() returned nil")
	}
	if Int(42).Array() != nil {
		t.Error("Int.Array() should return nil")
	}

	m := Map(map[string]any{"a": 1})
	if m.Map() == nil {
		t.Error("Map() returned nil")
	}
	if Int(42).Map() != nil {
		t.Error("Int.Map() should return nil")
	}
}

func TestValue_GetStructNil(t *testing.T) {
	v := Value{kind: kindAny, ptr: nil}
	_, ok := v.Get("field")
	if ok {
		t.Error("expected false for nil ptr")
	}
}

func TestValue_CompareIncompatible(t *testing.T) {
	// Comparing incompatible types
	arr := Array([]any{1})
	m := Map(map[string]any{"a": 1})

	// Should not panic
	_ = arr.Compare(m)
}

func TestVM_SubErrors(t *testing.T) {
	engine := New()
	_, err := engine.Eval(context.Background(), `"a" - "b"`)
	if err == nil {
		t.Error("expected error for string subtraction")
	}
}

func TestVM_MulErrors(t *testing.T) {
	engine := New()
	_, err := engine.Eval(context.Background(), `[1] * [2]`)
	if err == nil {
		t.Error("expected error for array multiplication")
	}
}

func TestVM_AddErrors(t *testing.T) {
	engine := New()
	env := map[string]any{
		"f": func() {},
	}
	_, err := engine.Eval(context.Background(), "f + f", env)
	if err == nil {
		t.Error("expected error for function addition")
	}
}

func TestSliceErrors(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(1), int64(2)},
	}

	// Invalid slice bounds now return empty array (Python-style behavior)
	result, err := engine.Eval(context.Background(), "arr[5:1]", env)
	if err != nil {
		t.Errorf("unexpected error for invalid slice bounds: %v", err)
	}
	arr, ok := result.([]any)
	if !ok || len(arr) != 0 {
		t.Errorf("expected empty array for invalid slice bounds, got %v", result)
	}
}

func TestLambda_Simple(t *testing.T) {
	// Test lambda parsing
	parser := newParser("(x) => x + 1")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if _, ok := ast.(*lambdaExpr); !ok {
		t.Error("expected lambdaExpr")
	}
}

func TestLambda_NoParams(t *testing.T) {
	parser := newParser("() => 42")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if lambda, ok := ast.(*lambdaExpr); ok {
		if len(lambda.Params) != 0 {
			t.Error("expected 0 params")
		}
	} else {
		t.Error("expected lambdaExpr")
	}
}

func TestLambda_MultiParams(t *testing.T) {
	parser := newParser("(x, y) => x + y")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if lambda, ok := ast.(*lambdaExpr); ok {
		if len(lambda.Params) != 2 {
			t.Errorf("expected 2 params, got %d", len(lambda.Params))
		}
	} else {
		t.Error("expected lambdaExpr")
	}
}

func TestGrouping(t *testing.T) {
	// Test that grouping works
	result, err := Eval(context.Background(), "(1 + 2) * 3")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(9) {
		t.Errorf("got %v, want 9", result)
	}
}

func TestSliceStep(t *testing.T) {
	// Slice with step currently just ignores step
	engine := New()
	env := map[string]any{
		"arr": []any{int64(0), int64(1), int64(2), int64(3)},
	}

	// Just verify it parses and runs without error
	_, err := engine.Eval(context.Background(), "arr[0:2]", env)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMapComputedKey(t *testing.T) {
	engine := New()
	env := map[string]any{
		"key": "foo",
	}

	result, err := engine.Eval(context.Background(), `{[key]: 42}`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	m := result.(map[string]any)
	if m["foo"] != int64(42) {
		t.Errorf("got %v", m)
	}
}

func TestBuiltin_SubstrEdgeCases(t *testing.T) {
	tests := []struct {
		expr   string
		result string
	}{
		{`substr("hello", 10)`, ""},          // start beyond end
		{`substr("hello", 0, 100)`, "hello"}, // length beyond end
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestBuiltin_TakeSkipEdges(t *testing.T) {
	tests := []struct {
		expr  string
		count int
	}{
		{"take([1,2,3], -1)", 0},
		{"skip([1,2,3], -1)", 3},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if len(result.([]any)) != tt.count {
				t.Errorf("got len %d, want %d", len(result.([]any)), tt.count)
			}
		})
	}
}

func TestNewValue_SliceTypes(t *testing.T) {
	// []Value conversion
	values := []Value{Int(1), Int(2)}
	v := NewValue(values)
	if !v.IsArray() {
		t.Error("expected array")
	}
}

func TestNewValue_FuncType(t *testing.T) {
	fn := func() int { return 42 }
	v := NewValue(fn)
	if !v.IsFunc() {
		t.Error("expected func")
	}
}

func TestValue_AnyType(t *testing.T) {
	type CustomStruct struct {
		X int
	}

	v := NewValue(CustomStruct{X: 5})
	typeName := v.Type()
	if typeName == "" {
		t.Error("expected non-empty type name")
	}
}

func TestVariadicFunction(t *testing.T) {
	engine := New(
		WithFunction("sumAll", func(nums ...int) int {
			sum := 0
			for _, n := range nums {
				sum += n
			}
			return sum
		}),
	)

	result, err := engine.Eval(context.Background(), "sumAll(1, 2, 3, 4, 5)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 15 {
		t.Errorf("got %v, want 15", result)
	}
}

func TestFunctionWithError(t *testing.T) {
	engine := New(
		WithFunction("divide", func(a, b int) (int, error) {
			if b == 0 {
				return 0, ErrDivisionByZero
			}
			return a / b, nil
		}),
	)

	// Success case
	result, err := engine.Eval(context.Background(), "divide(10, 2)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 5 {
		t.Errorf("got %v, want 5", result)
	}

	// Error case
	_, err = engine.Eval(context.Background(), "divide(10, 0)")
	if err == nil {
		t.Error("expected error")
	}
}

// ---------------------------------------------------------------------------
// Edge contracts for builtins (range overflow, fromJSON ints, ceil/floor range)
// ---------------------------------------------------------------------------

func TestRange_OverflowProtection(t *testing.T) {
	// Test that opRange doesn't allow overflow-based DoS
	_, err := Eval(context.Background(), "0..9223372036854775807", nil) // MaxInt64
	if err == nil {
		t.Error("Expected error for huge range, got nil")
	}

	// Test that builtinRange also has overflow protection
	_, err = Eval(context.Background(), "range(0, 9223372036854775807)", nil)
	if err == nil {
		t.Error("Expected error for huge range in builtin, got nil")
	}

	// Valid ranges should still work
	result, err := Eval(context.Background(), "1..10", nil)
	if err != nil {
		t.Fatalf("Unexpected error for valid range: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 10 {
		t.Errorf("Expected 10 elements, got %d", len(arr))
	}
}

func TestRange_MaxInt64_DoesNotHang(t *testing.T) {
	// Prior bug: MaxInt64..MaxInt64 would hang due to int64 overflow in i++ loop.
	result, err := Eval(context.Background(), "9223372036854775807..9223372036854775807", nil) // MaxInt64..MaxInt64
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any result, got %T", result)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 element, got %d", len(arr))
	}
	if arr[0].(int64) != int64(9223372036854775807) {
		t.Fatalf("unexpected value: %v", arr[0])
	}

	// builtinRange should also not hang.
	result, err = Eval(context.Background(), "range(9223372036854775807, 9223372036854775807, 1)", nil)
	if err != nil {
		t.Fatalf("unexpected error for builtin range: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 1 || arr[0].(int64) != int64(9223372036854775807) {
		t.Fatalf("unexpected builtin range result: %v", result)
	}
}

func TestRange_NegativeStepOverflow(t *testing.T) {
	// Test range with negative step and potential overflow scenarios
	result, err := Eval(context.Background(), "range(10, 0, -1)", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 11 {
		t.Errorf("Expected 11 elements, got %d", len(arr))
	}
	if arr[0].(int64) != 10 || arr[10].(int64) != 0 {
		t.Errorf("Unexpected range values: first=%v, last=%v", arr[0], arr[len(arr)-1])
	}

	// Empty range when step direction doesn't match
	result, err = Eval(context.Background(), "range(0, 10, -1)", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 0 {
		t.Errorf("Expected empty array for mismatched step, got %d elements", len(arr))
	}
}

func TestFromJSON_IntegerPreservation(t *testing.T) {
	// Test that fromJSON converts whole numbers to int64
	result, err := Eval(context.Background(), `fromJSON("[1, 2, 3]")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 3 {
		t.Fatalf("Expected 3 elements, got %d", len(arr))
	}

	// Check that elements are int64, not float64
	for i, v := range arr {
		if _, ok := v.(int64); !ok {
			t.Errorf("Element %d: expected int64, got %T", i, v)
		}
	}

	// Test with object
	result, err = Eval(context.Background(), `fromJSON("{\"x\": 42, \"y\": 3.14}")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	m := result.(map[string]any)

	// x should be int64 (whole number)
	if _, ok := m["x"].(int64); !ok {
		t.Errorf("x: expected int64, got %T", m["x"])
	}

	// y should remain float64 (has decimal)
	if _, ok := m["y"].(float64); !ok {
		t.Errorf("y: expected float64, got %T", m["y"])
	}

	// Test that int values work in expressions
	result, err = Eval(context.Background(), `let data = fromJSON("[10, 20, 30]"); data[0] + data[1]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 30 {
		t.Errorf("Expected 30, got %v", result)
	}

	// Safe integer boundary behavior:
	// 2^53-1 is representable exactly in float64, so we convert to int64.
	result, err = Eval(context.Background(), `fromJSON("9007199254740991")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if _, ok := result.(int64); !ok {
		t.Errorf("2^53-1: expected int64, got %T (%v)", result, result)
	}

	// 2^53 is outside the safe-integer range; keep as float64 to avoid precision corruption.
	result, err = Eval(context.Background(), `fromJSON("9007199254740992")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if _, ok := result.(float64); !ok {
		t.Errorf("2^53: expected float64, got %T (%v)", result, result)
	}
}

func TestCeilFloorRound_OutOfInt64Range_ReturnsFloat(t *testing.T) {
	// Extremely large numbers should not silently overflow into int64.
	// We keep them as float64.
	tests := []string{
		"ceil(1e30)",
		"floor(1e30)",
		"round(1e30)",
	}
	for _, expr := range tests {
		result, err := Eval(context.Background(), expr, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", expr, err)
		}
		if _, ok := result.(float64); !ok {
			t.Fatalf("%s: expected float64, got %T (%v)", expr, result, result)
		}
		if result.(float64) <= float64(math.MaxInt64) {
			t.Fatalf("%s: expected value > MaxInt64, got %v", expr, result)
		}
	}
}

// ---------------------------------------------------------------------------
// Domain edge proofs -> value_test.go
// ---------------------------------------------------------------------------
