package hyperex

import (
	"context"
	"testing"
)

// =============================================================================
// OPTIMIZER TESTS
// =============================================================================

func TestOptimizer_ConstantFolding(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"2 + 3", 5},
		{"2 * 3 + 4", 10},
		{"10 / 2 - 1", 4},
		{"2 ** 3", 8},
		{"10 % 3", 1},
		{"5 & 3", 1},
		{"5 | 3", 7},
		{"5 ^ 3", 6},
		{"1 << 4", 16},
		{"16 >> 2", 4},
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

func TestOptimizer_StringFolding(t *testing.T) {
	result, err := Eval(context.Background(), `"hello" + " " + "world"`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("got %v", result)
	}
}

func TestOptimizer_BooleanFolding(t *testing.T) {
	tests := []struct {
		expr   string
		result bool
	}{
		{"true && true", true},
		{"true && false", false},
		{"false || true", true},
		{"!false", true},
		{"1 == 1", true},
		{"1 != 2", true},
		{"1 < 2", true},
		{"2 <= 2", true},
		{"3 > 2", true},
		{"3 >= 3", true},
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

func TestOptimizer_TernaryFolding(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"true ? 1 : 2", 1},
		{"false ? 1 : 2", 2},
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

func TestOptimizer_CoalesceFolding(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"nil ?? 5", 5},
		{"3 ?? 5", 3},
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

func TestOptimizer_AlgebraicSimplifications(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		env    map[string]any
		result int64
	}{
		{"x + 0", map[string]any{"x": int64(5)}, 5},
		{"0 + x", map[string]any{"x": int64(5)}, 5},
		{"x - 0", map[string]any{"x": int64(5)}, 5},
		{"x * 1", map[string]any{"x": int64(5)}, 5},
		{"1 * x", map[string]any{"x": int64(5)}, 5},
		{"x * 0", map[string]any{"x": int64(5)}, 0},
		{"x / 1", map[string]any{"x": int64(5)}, 5},
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

func TestOptimizer_LogicalSimplifications(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		env    map[string]any
		result bool
	}{
		{"x && true", map[string]any{"x": true}, true},
		{"true && x", map[string]any{"x": true}, true},
		{"x && false", map[string]any{"x": true}, false},
		{"x || false", map[string]any{"x": true}, true},
		{"false || x", map[string]any{"x": true}, true},
		{"x || true", map[string]any{"x": false}, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestOptimizer_StrengthReduction(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		env    map[string]any
		result int64
	}{
		{"x * 2", map[string]any{"x": int64(5)}, 10},  // x << 1
		{"x * 4", map[string]any{"x": int64(5)}, 20},  // x << 2
		{"x * 8", map[string]any{"x": int64(5)}, 40},  // x << 3
		{"x % 2", map[string]any{"x": int64(5)}, 1},   // x & 1
		{"x % 4", map[string]any{"x": int64(5)}, 1},   // x & 3
		{"x ** 2", map[string]any{"x": int64(5)}, 25}, // x * x
		{"x ** 1", map[string]any{"x": int64(5)}, 5},  // x
		{"x ** 0", map[string]any{"x": int64(5)}, 1},  // 1
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

func TestOptimizer_UnarySimplifications(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		env    map[string]any
		result any
	}{
		{"!!x", map[string]any{"x": true}, true},
		{"-(-x)", map[string]any{"x": int64(5)}, int64(5)},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Domain edge proofs -> optimizer_test.go
// ---------------------------------------------------------------------------

func TestModeB_Optimizer_NumericFloat64Folds(t *testing.T) {
	// Drive numericZeroLit / numericOneLit float branches + strength reduction.
	eng := New(WithNumericMode(NumericFloat64))
	ctx := context.Background()
	env := map[string]any{"x": float64(3), "y": float64(4)}
	for _, expr := range []string{
		`x - x`,
		`x * 0`,
		`0 * x`,
		`x / x`,
		`x ** 0`,
		`x ** 1`,
		`x ** 2`,
		`x * 2`,
		`2 * x`,
		`x / 2`,
		`~~1`,
		`~~(1+2)`,
		`-(1+2)`,
		`1 + 0`,
		`0 + x`,
		`x - 0`,
		`x / 1`,
		`x / 1.0`,
	} {
		if _, err := eng.Eval(ctx, expr, env); err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
	}

	opt := defaultOptimizer()
	ast, err := newParser(`~~(1 + 2 - 3)`).parse()
	if err != nil {
		t.Fatal(err)
	}
	_ = opt.optimize(ast)

	// timeUTC non-integer float args (month…sec)
	_, err = eng.Eval(ctx, `timeUTC(2024.0, 6.5, 1, 0, 0, 0)`)
	if err == nil {
		t.Fatal("timeUTC month 6.5 want error")
	}
	for _, expr := range []string{
		`timeUTC(2024, 6.5, 1, 0, 0, 0)`,
		`timeUTC(2024, 6, 1.5, 0, 0, 0)`,
		`timeUTC(2024, 6, 1, 0.5, 0, 0)`,
		`timeUTC(2024, 6, 1, 0, 0.5, 0)`,
		`timeUTC(2024, 6, 1, 0, 0, 0.5)`,
		`timeHour()`,
		`timeMinute()`,
		`timeWeekday()`,
		`durationMs("x")`,
	} {
		if _, e := New().Eval(ctx, expr); e == nil {
			t.Fatalf("%s: expected error", expr)
		}
	}

	// quality nil-map / numberToQualityCode non-number direct
	nilMap := Value{kind: kindMap, ptr: (map[string]any)(nil)}
	_, err = builtinV([]Value{nilMap})
	if err == nil {
		t.Fatal("v(nil map) want error")
	}
	_, err = builtinQ([]Value{nilMap})
	if err == nil {
		t.Fatal("q(nil map) want error")
	}
	_, err = extractQualityCode([]Value{nilMap})
	if err == nil {
		t.Fatal("extractQualityCode nil map want error")
	}
	_, err = numberToQualityCode(Bool(true))
	if err == nil {
		t.Fatal("numberToQualityCode bool want error")
	}

	// funcValuesEqual remaining branches
	if funcValuesEqual(nil, nil) != true {
		t.Fatal("nil funcs equal")
	}
	if funcValuesEqual(nil, &builtinFunc{name: "x"}) {
		t.Fatal("nil vs func")
	}
	f := func() {}
	if !funcValuesEqual(f, f) {
		t.Fatal("same reflect.Func")
	}
	var p *int
	if !funcValuesEqual(p, (*int)(nil)) {
		t.Fatal("nil pointers equal")
	}
	x := 1
	if funcValuesEqual(&x, &x) != true {
		t.Fatal("same pointer")
	}
	if anyValuesEqual(nil, 1) {
		t.Fatal("nil vs value")
	}
	if !anyValuesEqual(nil, nil) {
		t.Fatal("nil any")
	}
	ch := make(chan int)
	if !anyValuesEqual(ch, ch) {
		t.Fatal("same chan")
	}
	type withFunc struct{ F func() }
	if anyValuesEqual(withFunc{}, withFunc{}) {
		t.Fatal("non-comparable struct")
	}

	// TokenPunct path via façade
	toks, err := Tokenize(`(a)`)
	if err != nil {
		t.Fatal(err)
	}
	foundPunct := false
	for _, tok := range toks {
		if tok.Type == TokenPunct {
			foundPunct = true
		}
	}
	if !foundPunct {
		t.Fatalf("want punct in (a): %+v", toks)
	}

	// WithMaxSourceBytes zero / negative no-op path if any
	_ = New(WithMaxSourceBytes(0))
	_ = New(WithMaxSourceBytes(-1))
}
