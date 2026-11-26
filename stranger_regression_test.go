package hyperex

import (
	"context"
	"errors"
	"math"
	"testing"
)

func TestNestedLetRestoresOuterBinding(t *testing.T) {
	for _, tc := range []struct {
		expr string
		want any
	}{
		{`let x = 1; (let x = 2; x) + x`, int64(3)},
		{`let x = 1; (let [x] = [2]; x) + x`, int64(3)},
		{`let x = 1; (let {x} = {x: 2}; x) + x`, int64(3)},
	} {
		got, err := Eval(context.Background(), tc.expr)
		if err != nil || got != tc.want {
			t.Fatalf("%s: got (%#v, %v), want (%#v, nil)", tc.expr, got, err, tc.want)
		}
	}
}

func TestEqualityHandlesCyclicValues(t *testing.T) {
	first := map[string]any{}
	first["self"] = first
	second := map[string]any{}
	second["self"] = second
	for _, expr := range []string{`x == x`, `x == y`, `x in [x]`} {
		got, err := Eval(context.Background(), expr, map[string]any{"x": first, "y": second})
		if err != nil || got != true {
			t.Fatalf("%s: got (%#v, %v), want (true, nil)", expr, got, err)
		}
	}
}

func TestRangeOperatorHonorsSandboxAndCancellation(t *testing.T) {
	eng := New(WithMaxOps(8))
	if _, err := eng.Eval(context.Background(), `len(0..999999)`); !errors.Is(err, ErrOpsLimitExceeded) {
		t.Fatalf("got %v, want ErrOpsLimitExceeded", err)
	}
}

func TestSecureCoercionInspectsNestedValues(t *testing.T) {
	type secret struct{ Name string }
	eng := New(WithSecureSandbox())
	for _, value := range []any{
		Array([]any{secret{Name: "secret"}}),
		Map(map[string]any{"nested": secret{Name: "secret"}}),
		Array([]any{Map(map[string]any{"nested": secret{Name: "secret"}})}),
	} {
		if _, err := eng.Eval(context.Background(), `toJSON(x)`, map[string]any{"x": value}); err == nil {
			t.Fatalf("secure mode accepted nested secret in %#v", value)
		}
	}
}

func TestInlineAvoidsCaptureAndHonorsInnerShadowing(t *testing.T) {
	for _, tc := range []struct {
		expr string
		env  map[string]any
		want any
	}{
		{`let x = y; (y => x)(1)`, map[string]any{"y": int64(5)}, int64(5)},
		{`let x = 1; let x = len([2,3]); x`, nil, int64(2)},
	} {
		text, err := Inline(tc.expr)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Eval(context.Background(), text, tc.env)
		if err != nil || got != tc.want {
			t.Fatalf("%s -> %s: got (%#v, %v), want (%#v, nil)", tc.expr, text, got, err, tc.want)
		}
	}
}

func TestProfilePreservesCallDepthAndResultHints(t *testing.T) {
	eng := New(WithSandbox(SandboxConfig{MaxCallDepth: 1}))
	prog, err := eng.Compile(`(x => (y => y)(1))(2)`)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, profileErr := prog.Profile(context.Background(), nil); !errors.Is(profileErr, ErrMaxCallDepth) {
		t.Fatalf("Profile got %v, want ErrMaxCallDepth", profileErr)
	}

	hinted, err := Compile(`1`, AsString())
	if err != nil {
		t.Fatal(err)
	}
	_, result, err := hinted.Profile(context.Background(), nil)
	if err != nil || result != "1" {
		t.Fatalf("Profile got (%#v, %v), want (\"1\", nil)", result, err)
	}
}

func TestProfileAppliesResultConversionErrors(t *testing.T) {
	eng := New(WithStrictMode(true))
	prog, err := eng.Compile(`"bad"`, AsInt())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := prog.Profile(context.Background(), nil); !errors.Is(err, ErrInvalidConversion) {
		t.Fatalf("Profile got %v, want ErrInvalidConversion", err)
	}
}

func TestSecureCoercionKeepsValidNestedValues(t *testing.T) {
	eng := New(WithSecureSandbox())
	value := Array([]any{Int(1), Map(map[string]any{"ok": Bool(true)})})
	got, err := eng.Eval(context.Background(), `toJSON(x)`, map[string]any{"x": value})
	if err != nil || got != `[1,{"ok":true}]` {
		t.Fatalf("got (%#v, %v), want valid nested JSON", got, err)
	}
}

func TestDoubleNegationPreservesMinIntPromotion(t *testing.T) {
	eng := New(WithConstantFolding(false))
	got, err := eng.Eval(context.Background(), `- -(-9223372036854775807 - 1)`)
	if err != nil {
		t.Fatal(err)
	}
	if got != float64(math.MinInt64) {
		t.Fatalf("got %#v (%T), want float64 MinInt64", got, got)
	}
}

func TestTypeCheckerValidatesFunctionArguments(t *testing.T) {
	eng := New(WithTypeCheck(struct{}{}))
	if _, err := eng.Compile(`take(1, 2)`); err == nil {
		t.Fatal("take(int, int) passed type checking")
	}
}

func TestInlinePreservesEvaluationErrors(t *testing.T) {
	text, err := Inline(`let x = 1 / 0; 42`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Eval(context.Background(), text); !errors.Is(err, ErrDivisionByZero) {
		t.Fatalf("Inline produced %q with error %v, want ErrDivisionByZero", text, err)
	}
}

func TestDoubleBitwiseNotHandlesOverflowPromotion(t *testing.T) {
	eng := New(WithConstantFolding(false))
	got, err := eng.Eval(context.Background(), `~~(9223372036854775807 + 1)`)
	if err != nil || got != int64(math.MinInt64) {
		t.Fatalf("got (%#v, %T, %v), want (MinInt64, int64, nil)", got, got, err)
	}
}

func TestTypeCheckerModelsPipeArguments(t *testing.T) {
	eng := New(WithTypeCheck(struct{}{}))
	if _, err := eng.Compile(`[1,2] | take(1)`); err != nil {
		t.Fatalf("valid pipe rejected: %v", err)
	}
	if _, err := eng.Compile(`[1] | take`); err == nil {
		t.Fatal("pipe with insufficient total arguments accepted")
	}
}

func TestTypedEvalFloatRejectsNonFiniteConversion(t *testing.T) {
	eng := New(WithNonFinite(NonFiniteError))
	for _, method := range []string{"engine", "program"} {
		var err error
		if method == "engine" {
			_, err = eng.EvalFloat(context.Background(), `"NaN"`)
		} else {
			prog, compileErr := eng.Compile(`"NaN"`)
			if compileErr != nil {
				t.Fatal(compileErr)
			}
			_, err = prog.EvalFloat(context.Background(), nil)
		}
		if !errors.Is(err, ErrNonFinite) {
			t.Fatalf("%s EvalFloat got %v, want ErrNonFinite", method, err)
		}
	}
}
