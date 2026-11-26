package hyperex

import (
	"context"
	"errors"
	"math"
	"testing"
)

func TestNonFiniteErrorSurvivesConstantFolding(t *testing.T) {
	_, err := New(WithNonFinite(NonFiniteError)).Eval(context.Background(), `(1e308 * 1e308) > 0`)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("got %v, want ErrNonFinite", err)
	}
}

func TestConstantFoldingUsesFloatBitEquality(t *testing.T) {
	got, err := Eval(context.Background(), `-0.0 == 0.0`)
	if err != nil || got != false {
		t.Fatalf("got=(%#v, %v), want (false, nil)", got, err)
	}
}

func TestConstantFoldingMatchesNaNComparisonSemantics(t *testing.T) {
	nan := `(1e308 * 1e308 - 1e308 * 1e308)`
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{nan + ` == ` + nan, true},
		{nan + ` <= ` + nan, true},
		{nan + ` >= ` + nan, true},
		{nan + ` < ` + nan, false},
	} {
		got, err := Eval(context.Background(), tc.expr)
		if err != nil || got != tc.want {
			t.Fatalf("%s: got (%#v, %v), want (%t, nil)", tc.expr, got, err, tc.want)
		}
	}
}

func TestSimplifyDoesNotEmitNonFiniteLiterals(t *testing.T) {
	text, err := Simplify(`1e308 * 1e308`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(text); err != nil {
		t.Fatalf("Simplify returned unparseable %q: %v", text, err)
	}
}

func TestAlgebraicSimplificationPreservesNumericSemantics(t *testing.T) {
	eng := New(WithConstantFolding(false))
	for _, tc := range []struct {
		expr string
		want any
	}{
		{`2 * 1.0`, float64(2)},
		{`2 / 1.0`, float64(2)},
		{`0 + .0`, float64(0)},
		{`(1.5 + 0) * 0`, float64(0)},
	} {
		got, err := eng.Eval(context.Background(), tc.expr)
		if err != nil || got != tc.want {
			t.Fatalf("%s: got (%#v, %T, %v), want (%#v, %T, nil)", tc.expr, got, got, err, tc.want, tc.want)
		}
	}
}

func TestAlgebraicSimplificationDoesNotHideErrorsOrNaN(t *testing.T) {
	eng := New(WithConstantFolding(false))
	if _, err := eng.Eval(context.Background(), `"a" * 1.0`); err == nil {
		t.Fatal("string times float unexpectedly succeeded")
	}
	if _, err := eng.Eval(context.Background(), `(0 + 0) / (0 + 0)`); !errors.Is(err, ErrDivisionByZero) {
		t.Fatalf("got %v, want ErrDivisionByZero", err)
	}
	got, err := eng.Eval(context.Background(), `(1e308 * 1e308) * 0`)
	if err != nil {
		t.Fatal(err)
	}
	if f, ok := got.(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("got %#v (%T), want NaN", got, got)
	}
}

func TestAlgebraicSimplificationPreservesFloatModeTypes(t *testing.T) {
	eng := New(WithConstantFolding(false), WithNumericMode(NumericFloat64))
	for _, expr := range []string{`2 * 1`, `2 ** 1`, `(1 + 2) / 1`} {
		got, err := eng.Eval(context.Background(), expr)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		if _, ok := got.(float64); !ok {
			t.Fatalf("%s: got %#v (%T), want float64", expr, got, got)
		}
	}
}

func TestPowerZeroPreservesFloatType(t *testing.T) {
	eng := New(WithConstantFolding(false))
	node, err := newParser(`0 + .0`).parse()
	if err != nil {
		t.Fatal(err)
	}
	optimized := eng.optimizer.optimize(node)
	if _, ok := optimized.(*floatLit); !ok {
		t.Fatalf("optimizer returned %T (%v), want floatLit", optimized, optimized)
	}
	got, err := eng.Eval(context.Background(), `.0 ** 0`)
	if err != nil || got != float64(1) {
		t.Fatalf("got (%#v, %T, %v), want (1.0, float64, nil)", got, got, err)
	}
}

func TestMultiplicationIdentityRejectsUnsupportedTypes(t *testing.T) {
	for _, expr := range []string{`true * 1`, `nil * 1`, `[1] * 1`} {
		if _, err := Eval(context.Background(), expr); err == nil {
			t.Fatalf("%s unexpectedly succeeded", expr)
		}
	}
}

func TestASTToStringParenthesizesMinIntExpression(t *testing.T) {
	node := &binaryExpr{Op: "*", Left: &intLit{Value: math.MinInt64}, Right: &intLit{Value: -1}}
	text := ASTToString(node)
	got, err := Eval(context.Background(), text)
	if err != nil {
		t.Fatalf("%q: %v", text, err)
	}
	if got != float64(-math.MinInt64) {
		t.Fatalf("%q -> %#v, want %v", text, got, -float64(math.MinInt64))
	}
}
