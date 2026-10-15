package hyperex

import (
	"context"
	"testing"
)

// =============================================================================
// PARSER TESTS
// =============================================================================

func TestParser_Arithmetic(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"2 + 2", 4},
		{"10 - 3", 7},
		{"4 * 5", 20},
		{"20 / 4", 5},
		{"17 % 5", 2},
		{"-5", -5},
		{"-(-5)", 5},
		{"2 + 3 * 4", 14},
		{"(2 + 3) * 4", 20},
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

func TestParser_FloatArithmetic(t *testing.T) {
	tests := []struct {
		expr   string
		result float64
	}{
		{"2.5 + 2.5", 5.0},
		{"10.0 / 4", 2.5},
		{"2 ** 10", 1024.0},
		{"2.0 ** 0.5", 1.4142135623730951},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Float() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Comparison(t *testing.T) {
	tests := []struct {
		expr   string
		result bool
	}{
		{"1 == 1", true},
		{"1 != 2", true},
		{"1 < 2", true},
		{"2 <= 2", true},
		{"3 > 2", true},
		{"3 >= 3", true},
		{"1 == 2", false},
		{`"abc" == "abc"`, true},
		{`"abc" != "def"`, true},
		{`"a" < "b"`, true},
		{`"b" > "a"`, true},
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

func TestParser_Logical(t *testing.T) {
	tests := []struct {
		expr   string
		result bool
	}{
		{"true && true", true},
		{"true && false", false},
		{"false && true", false},
		{"true || false", true},
		{"false || true", true},
		{"false || false", false},
		{"!true", false},
		{"!false", true},
		{"!!true", true},
		{"true and true", true},
		{"true or false", true},
		{"not false", true},
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

func TestParser_Bitwise(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"5 & 3", 1},
		{"(5 | 3)", 7},
		{"5 ^ 3", 6},
		{"~0", -1},
		{"~(-1)", 0},
		{"1 << 4", 16},
		{"16 >> 2", 4},
		{"8 >> 1", 4},
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

func TestParser_Ternary(t *testing.T) {
	tests := []struct {
		expr   string
		result any
	}{
		{"true ? 1 : 2", int64(1)},
		{"false ? 1 : 2", int64(2)},
		{"(1 > 0) ? \"yes\" : \"no\"", "yes"},
		{"(1 < 0) ? \"yes\" : \"no\"", "no"},
		{"true ? (true ? 1 : 2) : 3", int64(1)},
		{"false ? 1 : (true ? 2 : 3)", int64(2)},
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

func TestParser_Coalesce(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		env    map[string]any
		result int64
	}{
		{"x ?? 0", map[string]any{"x": nil}, 0},
		{"x ?? 0", map[string]any{"x": int64(42)}, 42},
		{"x ?? y ?? 0", map[string]any{"x": nil, "y": nil}, 0},
		{"x ?? y ?? 0", map[string]any{"x": nil, "y": int64(10)}, 10},
		{"x ?? y ?? z", map[string]any{"x": nil, "y": nil, "z": int64(5)}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Int() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_InOperator(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(1), int64(2), int64(3)},
		"str": "hello world",
		"m":   map[string]any{"foo": 1, "bar": 2},
	}

	tests := []struct {
		expr   string
		result bool
	}{
		{"2 in arr", true},
		{"5 in arr", false},
		{`"world" in str`, true},
		{`"xyz" in str`, false},
		{`"foo" in m`, true},
		{`"baz" in m`, false},
		{"!(1 in arr)", false},
		{"!(5 in arr)", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Range(t *testing.T) {
	tests := []struct {
		expr  string
		count int
	}{
		{"(1..5)", 5},
		{"(0..3)", 4},
		{"(10..10)", 1},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			arr := result.([]any)
			if len(arr) != tt.count {
				t.Errorf("got len %d, want %d", len(arr), tt.count)
			}
		})
	}
}

func TestParser_ArrayLiteral(t *testing.T) {
	tests := []struct {
		expr  string
		count int
	}{
		{"[]", 0},
		{"[1]", 1},
		{"[1, 2, 3]", 3},
		{"[1, 2, 3,]", 3}, // trailing comma
		{"[[1, 2], [3, 4]]", 2},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			arr := result.([]any)
			if len(arr) != tt.count {
				t.Errorf("got len %d, want %d", len(arr), tt.count)
			}
		})
	}
}

func TestParser_MapLiteral(t *testing.T) {
	tests := []struct {
		expr string
		keys []string
	}{
		{"{}", nil},
		{"{foo: 1}", []string{"foo"}},
		{`{"foo": 1, "bar": 2}`, []string{"foo", "bar"}},
		{"{foo: 1,}", []string{"foo"}}, // trailing comma
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			m := result.(map[string]any)
			if len(m) != len(tt.keys) {
				t.Errorf("got len %d, want %d", len(m), len(tt.keys))
			}
			for _, k := range tt.keys {
				if _, ok := m[k]; !ok {
					t.Errorf("missing key %q", k)
				}
			}
		})
	}
}

func TestParser_IndexAccess(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(10), int64(20), int64(30)},
		"str": "hello",
	}

	tests := []struct {
		expr   string
		result any
	}{
		{"arr[0]", int64(10)},
		{"arr[1]", int64(20)},
		{"arr[-1]", int64(30)},
		{"arr[-2]", int64(20)},
		{"str[0]", "h"},
		{"str[-1]", "o"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_SliceAccess(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(0), int64(1), int64(2), int64(3), int64(4)},
		"str": "hello",
	}

	tests := []struct {
		expr  string
		count int
	}{
		{"arr[1:3]", 2},
		{"arr[:2]", 2},
		{"arr[3:]", 2},
		{"arr[:]", 5},
		{"str[1:3]", 2},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			switch v := result.(type) {
			case []any:
				if len(v) != tt.count {
					t.Errorf("got len %d, want %d", len(v), tt.count)
				}
			case string:
				if len(v) != tt.count {
					t.Errorf("got len %d, want %d", len(v), tt.count)
				}
			}
		})
	}
}

func TestParser_MemberAccess(t *testing.T) {
	engine := New()
	env := map[string]any{
		"user": map[string]any{
			"name": "Alice",
			"age":  int64(30),
		},
	}

	tests := []struct {
		expr   string
		result any
	}{
		{"user.name", "Alice"},
		{"user.age", int64(30)},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_OptionalChain(t *testing.T) {
	engine := New()
	env := map[string]any{
		"user": map[string]any{"name": "Alice"},
		"null": nil,
	}

	tests := []struct {
		expr   string
		isNil  bool
		result string
	}{
		{"user?.name", false, "Alice"},
		{"null?.name", true, ""},
		{"user?.missing", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if tt.isNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else {
				if result != tt.result {
					t.Errorf("got %v, want %v", result, tt.result)
				}
			}
		})
	}
}

func TestParser_Let(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"let x = 5; x + 1", 6},
		{"let x = 10; let y = 20; x + y", 30},
		{"let a = 2; let b = 3; a * b", 6},
		{"let n = 100; n / 2", 50},
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

func TestParser_NotIn(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(1), int64(2), int64(3)},
		"str": "hello world",
		"m":   map[string]any{"foo": 1, "bar": 2},
	}

	tests := []struct {
		expr   string
		result bool
	}{
		{"1 not in arr", false},
		{"5 not in arr", true},
		{"10 not in arr", true},
		{`"xyz" not in str`, true},
		{`"world" not in str`, false},
		{`"baz" not in m`, true},
		{`"foo" not in m`, false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Regex(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		result bool
	}{
		{`"hello" ~= "^h"`, true},
		{`"hello" ~= "o$"`, true},
		{`"hello" ~= "^x"`, false},
		{`"test123" ~= "[0-9]+"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Errors(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"incomplete_plus", "1 +"},
		{"unbalanced_paren", "(1 + 2"},
		{"double_op", "1 + + 1"},
		{"incomplete_array", "[1, 2,"},
		{"incomplete_map", "{foo:"},
		{"invalid_primary", "1 + @"},
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
// AST TESTS
// =============================================================================

func TestAST_Walk(t *testing.T) {
	parser := newParser("1 + 2 * 3")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	count := 0
	Walk(ast, func(n Node) bool {
		count++
		return true
	})

	if count < 5 { // Should visit multiple nodes
		t.Errorf("Walk visited %d nodes, expected more", count)
	}
}

func TestAST_WalkStop(t *testing.T) {
	parser := newParser("1 + 2 * 3")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	count := 0
	Walk(ast, func(n Node) bool {
		count++
		return count < 2 // Stop after 2 nodes
	})

	// Walk should stop early
	if count < 1 {
		t.Errorf("Walk visited %d nodes, expected at least 1", count)
	}
}

func TestAST_NodeStrings(t *testing.T) {
	nodes := []Node{
		&intLit{Value: 42},
		&floatLit{Value: 3.14},
		&stringLit{Value: "hello"},
		&boolLit{Value: true},
		&nilLit{},
		&ident{Name: "x"},
		&binaryExpr{Op: "+", Left: &intLit{Value: 1}, Right: &intLit{Value: 2}},
		&unaryExpr{Op: "-", Operand: &intLit{Value: 5}},
		&ternaryExpr{Cond: &boolLit{Value: true}, Then: &intLit{Value: 1}, Else: &intLit{Value: 2}},
		&arrayLit{Elements: []Node{&intLit{Value: 1}}},
		&mapLit{Keys: []Node{&stringLit{Value: "a"}}, Values: []Node{&intLit{Value: 1}}},
		&rangeExpr{Start: &intLit{Value: 1}, End: &intLit{Value: 5}},
		&coalesceExpr{Left: &ident{Name: "x"}, Right: &intLit{Value: 0}},
	}

	for _, n := range nodes {
		s := n.String()
		if s == "" {
			t.Errorf("empty string for %T", n)
		}
	}
}
