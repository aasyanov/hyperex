package hyperex

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Acceptance scenarios not already covered elsewhere.

func TestAcceptance_A3_SecureDeniesTimeMemberReflect(t *testing.T) {
	eng := New(WithSecureSandbox())
	ts := time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC)
	_, err := eng.Eval(context.Background(), `ts.Location`, map[string]any{"ts": ts})
	if err == nil {
		t.Fatal("expected Secure to deny reflect member on time (use builtins)")
	}
}

func TestAcceptance_A5_TimeAdd(t *testing.T) {
	eng := NewSecureCalc()
	ts := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	got, err := eng.Eval(context.Background(), `timeAdd(ts, durationSeconds(60))`, map[string]any{"ts": ts})
	if err != nil {
		t.Fatalf("timeAdd: %v", err)
	}
	want := time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC)
	if !NewValue(got).Equal(Time(want)) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestAcceptance_C1_BareQcode_IsGood(t *testing.T) {
	// Bare qcode: isGood accepts a numeric status code (host may bind temp_q).
	eng := New()
	ok, err := eng.EvalBool(context.Background(), `isGood(temp_q)`, map[string]any{"temp_q": float64(0)})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("isGood(0) want true")
	}
}

func TestAcceptance_D3_WithFunctionTrustedNoForcedTax(t *testing.T) {
	// WithFunction remains trusted: many calls under low MaxOps still succeed
	// (unlike SafeFunction which taxes each call).
	calls := 0
	eng := New(
		WithMaxOps(50),
		WithFunction("bump", func() int {
			calls++
			return calls
		}),
	)
	// 40 bytecode-light calls: trusted fn must not apply SafeFunction tax of 16.
	_, err := eng.Eval(context.Background(), `
		bump() + bump() + bump() + bump() + bump() +
		bump() + bump() + bump() + bump() + bump()
	`)
	if err != nil {
		t.Fatalf("WithFunction should not be ops-taxed like SafeFunction: %v", err)
	}
	if calls != 10 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestAcceptance_E1_AnalyzeWarnConstants(t *testing.T) {
	a, err := Analyze(`x | y`)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range a.Warnings {
		if w == WarnAmbiguousBitwiseOr {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("want WarnAmbiguousBitwiseOr in %v", a.Warnings)
	}
}

func TestAcceptance_B4_NonFinite_SqrtNeg(t *testing.T) {
	eng := New(WithNonFinite(NonFiniteError))
	_, err := eng.Eval(context.Background(), `sqrt(-1)`)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("want ErrNonFinite, got %v", err)
	}
}
