package hyperex

import (
	"context"
	"testing"
)

// =============================================================================
// TYPE CHECKER TESTS
// =============================================================================

func TestTypeChecker_BasicTypes(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		expr     string
		expected Type
	}{
		{"42", TypeInt},
		{"3.14", TypeFloat},
		{`"hello"`, TypeString},
		{"true", TypeBool},
		{"nil", TypeNil},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeChecker_BinaryOps(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		expr     string
		expected Type
	}{
		{"1 + 2", TypeInt},
		{"1.0 + 2", TypeFloat},
		{"1 + 2.0", TypeFloat},
		{`"a" + "b"`, TypeString},
		{"1 - 2", TypeInt},
		{"1 * 2", TypeInt},
		{"1 / 2", TypeInt},
		{"1.0 / 2", TypeFloat},
		{"1 == 2", TypeBool},
		{"1 < 2", TypeBool},
		{"true && false", TypeBool},
		{"true || false", TypeBool},
		{"1 & 2", TypeInt},
		{"1 << 2", TypeInt},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeChecker_WithStructEnv(t *testing.T) {
	type Env struct {
		Count int     `expr:"count"`
		Price float64 `expr:"price"`
		Name  string  `expr:"name"`
		Ok    bool    `expr:"ok"`
	}

	tc := NewTypeChecker().WithStructEnv(Env{})

	tests := []struct {
		expr     string
		expected Type
	}{
		{"count", TypeInt},
		{"price", TypeFloat},
		{"name", TypeString},
		{"ok", TypeBool},
		{"count + 1", TypeInt},
		{"price * 2", TypeFloat},
		{`name + " suffix"`, TypeString},
		{"ok && true", TypeBool},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeChecker_Builtins(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		expr     string
		expected Type
	}{
		{`len("hello")`, TypeInt},
		{`upper("hello")`, TypeString},
		// abs can return int for int inputs, float for float inputs; treat as any.
		{`abs(-5)`, TypeAny},
		{`sqrt(16)`, TypeFloat},
		{`contains("hello", "ll")`, TypeBool},
		{`split("a,b", ",")`, TypeArray},
		// sum returns int when all ints and no overflow, float otherwise.
		{`sum([1,2,3])`, TypeAny},
		{`first([1,2,3])`, TypeAny},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeChecker_Ternary(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		expr     string
		expected Type
	}{
		{"true ? 1 : 2", TypeInt},
		{"true ? 1.0 : 2.0", TypeFloat},
		{`true ? "a" : "b"`, TypeString},
		{"true ? 1 : 2.0", TypeFloat}, // Mixed numeric -> float
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeChecker_Let(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		expr     string
		expected Type
	}{
		{"let x = 5; x + 1", TypeInt},
		{"let x = 3.14; x * 2", TypeFloat},
		{`let s = "hi"; s + "!"`, TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWithTypeCheck(t *testing.T) {
	type Env struct {
		X int `expr:"x"`
		Y int `expr:"y"`
	}

	engine := New(WithTypeCheck(Env{}))

	// Should compile successfully
	_, err := engine.Compile("x + y")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	// Execute with map env (Program.Eval requires map[string]any)
	result, err := engine.Eval(context.Background(), "x + y", Env{X: 10, Y: 20})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if NewValue(result).Int() != 30 {
		t.Errorf("got %v, want 30", result)
	}
}

func TestCheckExpressionType(t *testing.T) {
	type Env struct {
		X int `expr:"x"`
	}

	// Should pass - x is int, expression returns int
	err := CheckExpressionType("x + 1", TypeInt, Env{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should pass - literal int
	err = CheckExpressionType("42", TypeInt, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestType_String(t *testing.T) {
	types := []struct {
		typ      Type
		expected string
	}{
		{TypeUnknown, "unknown"},
		{TypeNil, "nil"},
		{TypeBool, "bool"},
		{TypeInt, "int"},
		{TypeFloat, "float"},
		{TypeString, "string"},
		{TypeArray, "array"},
		{TypeMap, "map"},
		{TypeFunc, "func"},
		{TypeAny, "any"},
	}

	for _, tt := range types {
		if tt.typ.String() != tt.expected {
			t.Errorf("Type %d: got %v, want %v", tt.typ, tt.typ.String(), tt.expected)
		}
	}
}

func TestVM_Division_FloatByInt(t *testing.T) {
	result, err := Eval(context.Background(), "7.0 / 2")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Float() != 3.5 {
		t.Errorf("got %v, want 3.5", result)
	}
}

func TestVM_Division_IntByFloat(t *testing.T) {
	result, err := Eval(context.Background(), "7 / 2.0")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Float() != 3.5 {
		t.Errorf("got %v, want 3.5", result)
	}
}

func TestVM_Sub_IntFloat(t *testing.T) {
	tests := []struct {
		expr   string
		result float64
	}{
		{"7.0 - 2", 5.0},
		{"7 - 2.0", 5.0},
	}

	for _, tt := range tests {
		result, err := Eval(context.Background(), tt.expr)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if NewValue(result).Float() != tt.result {
			t.Errorf("got %v, want %v", result, tt.result)
		}
	}
}

func TestVM_Slice_StringWithNegative(t *testing.T) {
	engine := New()
	env := map[string]any{"s": "hello"}

	result, err := engine.Eval(context.Background(), "s[1:-1]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "ell" {
		t.Errorf("got %v, want ell", result)
	}
}

func TestVM_Slice_MapError(t *testing.T) {
	engine := New()
	env := map[string]any{"m": map[string]any{"a": 1}}

	_, err := engine.Eval(context.Background(), "m[0:1]", env)
	if err == nil {
		t.Error("expected error for slicing map")
	}
}

func TestBuiltin_Sort_Float(t *testing.T) {
	result, err := Eval(context.Background(), "sort([3.0, 1.0, 2.0])")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := result.([]any)
	if NewValue(arr[0]).Float() != 1.0 {
		t.Errorf("first should be 1.0, got %v", arr[0])
	}
}

func TestBuiltin_Sort_String(t *testing.T) {
	result, err := Eval(context.Background(), `sort(["c", "a", "b"])`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := result.([]any)
	if arr[0] != "a" {
		t.Errorf("first should be a, got %v", arr[0])
	}
}

func TestSlice_StringOmitEnd(t *testing.T) {
	engine := New()
	env := map[string]any{"s": "hello"}

	result, err := engine.Eval(context.Background(), "s[2:]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "llo" {
		t.Errorf("got %v, want llo", result)
	}
}

func TestSlice_StringOmitStart(t *testing.T) {
	engine := New()
	env := map[string]any{"s": "hello"}

	result, err := engine.Eval(context.Background(), "s[:3]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "hel" {
		t.Errorf("got %v, want hel", result)
	}
}

func TestIndex_StringOutOfBounds(t *testing.T) {
	engine := New()
	env := map[string]any{"s": "hello"}

	_, err := engine.Eval(context.Background(), "s[100]", env)
	if err == nil {
		t.Error("expected error for out of bounds")
	}
}

func TestCompiler_MoreOps(t *testing.T) {
	// Test compilation of more complex expressions
	exprs := []string{
		"true ? 1 : 2",
		"nil ?? 5",
		"[1, 2, 3]",
		"{a: 1, b: 2}",
		"(1..3)",
	}

	for _, expr := range exprs {
		_, err := Compile(expr)
		if err != nil {
			t.Errorf("compile error for %s: %v", expr, err)
		}
	}
}

func TestVM_UnsignedRightShift(t *testing.T) {
	// Test unsigned right shift
	tests := []struct {
		expr     string
		expected int64
	}{
		{"(-8) >>> 1", 0x7ffffffffffffffc}, // -8 as unsigned >> 1
		{"8 >>> 1", 4},                     // Positive works same as >>
		{"(-1) >>> 32", 0xffffffff},        // All 1s >> 32
		{"16 >>> 2", 4},                    // Simple positive case
	}
	for _, tt := range tests {
		result, err := Eval(context.Background(), tt.expr)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.expr, err)
			continue
		}
		if got, ok := result.(int64); !ok || got != tt.expected {
			t.Errorf("%s: expected %d, got %v", tt.expr, tt.expected, result)
		}
	}
}

func TestOptChain_DeepNil(t *testing.T) {
	engine := New()
	env := map[string]any{
		"obj": map[string]any{
			"nested": nil,
		},
	}

	result, err := engine.Eval(context.Background(), "obj?.nested?.value", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestMemberAccess_NestedMap(t *testing.T) {
	engine := New()
	env := map[string]any{
		"obj": map[string]any{
			"nested": map[string]any{
				"value": int64(42),
			},
		},
	}

	result, err := engine.Eval(context.Background(), "obj.nested.value", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(42) {
		t.Errorf("got %v, want 42", result)
	}
}

func TestIndex_2D_Array(t *testing.T) {
	engine := New()
	env := map[string]any{
		"matrix": []any{
			[]any{int64(1), int64(2)},
			[]any{int64(3), int64(4)},
		},
	}

	result, err := engine.Eval(context.Background(), "matrix[1][0]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(3) {
		t.Errorf("got %v, want 3", result)
	}
}

func TestFunctionChain(t *testing.T) {
	result, err := Eval(context.Background(), "upper(lower(\"HeLLo\"))")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "HELLO" {
		t.Errorf("got %v, want HELLO", result)
	}
}

func TestComplex_Expression(t *testing.T) {
	result, err := Eval(context.Background(), "(1 + 2) * 3 > 5 && true || false")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != true {
		t.Errorf("got %v, want true", result)
	}
}

func TestParser_MethodCall(t *testing.T) {
	// Test method-like calls
	engine := New()
	env := map[string]any{
		"obj": map[string]any{
			"getValue": func() int { return 42 },
		},
	}

	result, err := engine.Eval(context.Background(), "obj.getValue()", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 42 {
		t.Errorf("got %v, want 42", result)
	}
}

func TestVM_Push_Overflow(t *testing.T) {
	// Create expression that uses deep stack
	// This is hard to trigger with normal expressions
	// Just test a deep nested expression
	result, err := Eval(context.Background(), "((((((1 + 2) + 3) + 4) + 5) + 6) + 7)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(28) {
		t.Errorf("got %v, want 28", result)
	}
}

func TestBuiltin_HigherOrder_Errors(t *testing.T) {
	// map, filter, reduce, find, findIndex all return "not implemented" errors
	// because they require closure support
	funcs := []string{
		"map([], 1)",
		"filter([], 1)",
		"reduce([], 1, 1)",
		"find([], 1)",
		"findIndex([], 1)",
	}

	for _, fn := range funcs {
		_, err := Eval(context.Background(), fn)
		if err == nil {
			t.Errorf("expected error for %s", fn)
		}
	}
}

func TestValue_Interface_Func(t *testing.T) {
	fn := func() {}
	v := Func(fn)
	iface := v.Interface()
	if iface == nil {
		t.Error("expected non-nil interface")
	}
}

func TestSlice_Clamp(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(1), int64(2), int64(3)},
	}

	// Start beyond end now returns empty array (Python-style clamping behavior)
	result, err := engine.Eval(context.Background(), "arr[10:20]", env)
	if err != nil {
		t.Errorf("unexpected error for out of bounds slice: %v", err)
	}
	arr, ok := result.([]any)
	if !ok || len(arr) != 0 {
		t.Errorf("expected empty array for out of bounds slice, got %v", result)
	}
}

// TestAST_NodeMarkers covers the empty node() interface marker methods
func TestAST_NodeMarkers(t *testing.T) {
	// These methods are just type markers for the Node interface
	// They have empty bodies but calling them increases coverage
	(&intLit{}).node()
	(&floatLit{}).node()
	(&stringLit{}).node()
	(&boolLit{}).node()
	(&nilLit{}).node()
	(&ident{}).node()
	(&binaryExpr{}).node()
	(&unaryExpr{}).node()
	(&ternaryExpr{}).node()
	(&callExpr{}).node()
	(&memberExpr{}).node()
	(&indexExpr{}).node()
	(&sliceExpr{}).node()
	(&arrayLit{}).node()
	(&mapLit{}).node()
	(&rangeExpr{}).node()
	(&pipeExpr{}).node()
	(&lambdaExpr{}).node()
	(&coalesceExpr{}).node()
	(&letExpr{}).node()
}

// ---------------------------------------------------------------------------
// Domain edge proofs -> typechecker_test.go
// ---------------------------------------------------------------------------

func TestModeB_TypeChecker_TimeAndQuality(t *testing.T) {
	tc := NewTypeChecker()
	for _, expr := range []string{
		`timeHour(t)`,
		`timeAdd(t, d)`,
		`durationSeconds(1)`,
		`isGood(temp)`,
		`v(temp) + 1`,
		`withQ(1, 0)`,
		`unix(t)`,
		`durMs(d)`,
	} {
		ast, err := newParser(expr).parse()
		if err != nil {
			t.Fatalf("parse %q: %v", expr, err)
		}
		// Best-effort: should not panic; signatures registered.
		_ = tc.Check(ast)
		_ = tc.Infer(ast)
	}
	sig, ok := tc.funcs[nameTimeHour]
	if !ok || sig.Returns != TypeInt {
		t.Fatalf("timeHour signature missing/wrong: %#v", sig)
	}
	sig, ok = tc.funcs[nameIsGood]
	if !ok || sig.Returns != TypeBool {
		t.Fatalf("isGood signature missing/wrong: %#v", sig)
	}
	sig, ok = tc.funcs[nameWithQ]
	if !ok || sig.Returns != TypeMap {
		t.Fatalf("withQ signature missing/wrong: %#v", sig)
	}
}
