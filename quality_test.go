package hyperex

import (
	"context"
	"math"
	"testing"
)

func TestQuality_IsGood_Map(t *testing.T) {
	eng := New()
	env := map[string]any{
		"temp": map[string]any{"v": 1, "q": 0},
	}
	got, err := eng.EvalBool(context.Background(), `isGood(temp)`, env)
	if err != nil {
		t.Fatalf("isGood: %v", err)
	}
	if !got {
		t.Fatal("isGood({v:1,q:0}) want true")
	}
}

func TestQuality_IsBad_ClassBit(t *testing.T) {
	eng := New()
	env := map[string]any{
		"temp": map[string]any{"v": 1, "q": QualityClassBad},
	}
	got, err := eng.EvalBool(context.Background(), `isBad(temp)`, env)
	if err != nil {
		t.Fatalf("isBad: %v", err)
	}
	if !got {
		t.Fatal("isBad with 0x80000000 want true")
	}
	if ok, _ := eng.EvalBool(context.Background(), `isGood(temp)`, env); ok {
		t.Fatal("isGood should be false for Bad")
	}
}

func TestQuality_IsUncertain_ClassBit(t *testing.T) {
	eng := New()
	env := map[string]any{
		"temp": map[string]any{"v": 1.5, "q": float64(QualityClassUncertain)},
	}
	got, err := eng.EvalBool(context.Background(), `isUncertain(temp)`, env)
	if err != nil {
		t.Fatalf("isUncertain: %v", err)
	}
	if !got {
		t.Fatal("isUncertain with 0x40000000 want true")
	}
}

func TestQuality_V_PlusOne_NumericFloat64(t *testing.T) {
	eng := New(WithNumericMode(NumericFloat64))
	env := map[string]any{
		"temp": map[string]any{"v": 1.5, "q": 0},
	}
	got, err := eng.Eval(context.Background(), `v(temp) + 1`, env)
	if err != nil {
		t.Fatalf("v(temp)+1: %v", err)
	}
	if NewValue(got).Float() != 2.5 {
		t.Fatalf("got %v want 2.5", got)
	}

	// No arithmetic auto-unwrap: temp + 1 must fail.
	_, err = eng.Eval(context.Background(), `temp + 1`, env)
	if err == nil {
		t.Fatal("temp + 1 on map must fail (no auto-unwrap)")
	}
}

func TestQuality_Secure_QualifiedMap(t *testing.T) {
	eng := New(WithSecureSandbox())
	env := map[string]any{
		"temp": map[string]any{"v": 12.5, "q": float64(0)},
	}
	got, err := eng.Eval(context.Background(), `isGood(temp) && v(temp) > 10`, env)
	if err != nil {
		t.Fatalf("Secure + qualified map: %v", err)
	}
	if got != true {
		t.Fatalf("got %v want true", got)
	}
}

func TestQuality_ExtractVariables_V(t *testing.T) {
	vars, err := ExtractVariables(`v(temp) + 1`)
	if err != nil {
		t.Fatalf("ExtractVariables: %v", err)
	}
	found := false
	for _, name := range vars {
		if name == "temp" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("ExtractVariables(v(temp)) missing temp: %v", vars)
	}
}

func TestQuality_BareQcode_IsGood(t *testing.T) {
	eng := New()
	got, err := eng.EvalBool(context.Background(), `isGood(0)`)
	if err != nil {
		t.Fatalf("isGood(0): %v", err)
	}
	if !got {
		t.Fatal("bare qcode isGood(0) want true")
	}
	bad, err := eng.EvalBool(context.Background(), `isBad(0x80000000)`)
	if err != nil {
		t.Fatalf("isBad bare: %v", err)
	}
	if !bad {
		t.Fatal("isBad(0x80000000) want true")
	}
}

func TestQuality_WithQ_AndHelpers(t *testing.T) {
	eng := New()
	got, err := eng.Eval(context.Background(), `let t = withQ(42, 0); v(t)`)
	if err != nil {
		t.Fatalf("withQ/v: %v", err)
	}
	if NewValue(got).Int() != 42 {
		t.Fatalf("v(withQ(...)): got %v", got)
	}
	ok, err := eng.EvalBool(context.Background(), `isGood(withQ(1, 0))`)
	if err != nil || !ok {
		t.Fatalf("isGood(withQ): %v %v", ok, err)
	}
}

func TestQuality_BuiltinNames(t *testing.T) {
	for _, name := range []string{nameV, nameQ, nameIsGood, nameIsBad, nameIsUncertain, nameWithQ} {
		if !IsBuiltin(name) {
			t.Fatalf("IsBuiltin(%q) = false", name)
		}
	}
	if n := len(BuiltinNames()); n != 100 {
		t.Fatalf("BuiltinNames len=%d, want 100", n)
	}
}

func TestQuality_NilAndPassthrough(t *testing.T) {
	eng := New()
	got, err := eng.Eval(context.Background(), `v(nil)`)
	if err != nil {
		t.Fatalf("v(nil): %v", err)
	}
	if got != nil {
		t.Fatalf("v(nil) want nil, got %v", got)
	}
	got, err = eng.Eval(context.Background(), `v(3)`)
	if err != nil {
		t.Fatalf("v(3): %v", err)
	}
	if NewValue(got).Int() != 3 {
		t.Fatalf("v(3): got %v", got)
	}
	got, err = eng.Eval(context.Background(), `q(7)`)
	if err != nil {
		t.Fatalf("q(7): %v", err)
	}
	if NewValue(got).Int() != 7 {
		t.Fatalf("q(7): got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Domain edge proofs -> quality_test.go
// ---------------------------------------------------------------------------

func TestModeB_Quality_ErrorPaths(t *testing.T) {
	eng := New()
	ctx := context.Background()
	env := map[string]any{
		"good": map[string]any{"v": 1, "q": 0},
		"noV":  map[string]any{"q": 0},
		"noQ":  map[string]any{"v": 1},
		"badQ": map[string]any{"v": 1, "q": "x"},
		"fn":   func() {},
	}

	// Success edges not covered elsewhere
	got, err := eng.Eval(ctx, `v(true)`, env)
	if err != nil || got != true {
		t.Fatalf("v(true): %v %v", got, err)
	}
	got, err = eng.Eval(ctx, `v("hi")`, env)
	if err != nil || got != "hi" {
		t.Fatalf("v(string): %v %v", got, err)
	}
	if _, err = eng.Eval(ctx, `v([1,2])`, env); err != nil {
		t.Fatalf("v(array): %v", err)
	}
	got, err = eng.Eval(ctx, `withQ(1, 0.0)`, env)
	if err != nil {
		t.Fatalf("withQ float q: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("withQ want map, got %T", got)
	}
	if _, isF := m["q"].(float64); !isF {
		t.Fatalf("withQ float qcode should store float64, got %T", m["q"])
	}

	errExprs := []string{
		`v()`,
		`v(noV)`,
		`v(fn)`,
		`q()`,
		`q(noQ)`,
		`q(badQ)`,
		`q("x")`,
		`q(fn)`,
		`isGood()`,
		`isGood("x")`,
		`isGood(fn)`,
		`isGood(noQ)`,
		`isBad()`,
		`isUncertain()`,
		`isGood(-1)`,
		`isGood(4294967296)`,
		`isGood(1e40)`,
		`withQ(1)`,
		`withQ(1, "x")`,
		`withQ()`,
	}
	for _, expr := range errExprs {
		if _, e := eng.Eval(ctx, expr, env); e == nil {
			t.Fatalf("%s: expected error", expr)
		}
	}

	// NaN / Inf qcode via float env
	nanEnv := map[string]any{
		"nanQ": map[string]any{"v": 1, "q": math.NaN()},
		"infQ": map[string]any{"v": 1, "q": math.Inf(1)},
	}
	for _, expr := range []string{`isGood(nanQ)`, `isBad(infQ)`, `isUncertain(nanQ)`} {
		if _, e := eng.Eval(ctx, expr, nanEnv); e == nil {
			t.Fatalf("%s: expected qcode range error", expr)
		}
	}
	// q() returns the numeric field as-is (including NaN); class helpers validate range.
	got, err = eng.Eval(ctx, `q(nanQ)`, nanEnv)
	if err != nil {
		t.Fatalf("q(nanQ) passthrough: %v", err)
	}
	if f, ok := got.(float64); !ok || !math.IsNaN(f) {
		t.Fatalf("q(nanQ) want NaN, got %v", got)
	}
}
