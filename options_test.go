package hyperex

import (
	"context"
	"testing"
	"time"
)

// options_test.go - default vs override for each WithXxx (functional Option pattern).
// Uses white-box inspection of *engineConfig after New(...).

func TestOptions_Defaults(t *testing.T) {
	eng := New()
	cfg := eng.config
	if cfg.optimize != defaultOptimize {
		t.Fatalf("default optimize=%v want %v", cfg.optimize, defaultOptimize)
	}
	if cfg.constantFolding != defaultConstantFolding {
		t.Fatalf("default constantFolding=%v want %v", cfg.constantFolding, defaultConstantFolding)
	}
	if cfg.strengthReduce != defaultStrengthReduce {
		t.Fatalf("default strengthReduce=%v want %v", cfg.strengthReduce, defaultStrengthReduce)
	}
	if cfg.maxCallDepth != defaultMaxCallDepth {
		t.Fatalf("default maxCallDepth=%d want %d", cfg.maxCallDepth, defaultMaxCallDepth)
	}
	if cfg.strictMode {
		t.Fatal("default strictMode=false")
	}
	if cfg.numericMode != NumericDefault {
		t.Fatalf("default numericMode=%v want NumericDefault", cfg.numericMode)
	}
	if cfg.nonFiniteMode != NonFiniteAllow {
		t.Fatalf("default nonFiniteMode=%v want NonFiniteAllow", cfg.nonFiniteMode)
	}
	if cfg.typeCheck {
		t.Fatal("default typeCheck=false")
	}
	if cfg.sandbox != nil {
		t.Fatal("default sandbox=nil")
	}
	if len(cfg.functions) != 0 {
		t.Fatalf("default functions empty, got %d", len(cfg.functions))
	}
	if len(cfg.operators) != 0 {
		t.Fatalf("default operators empty, got %d", len(cfg.operators))
	}
}

func TestOptions_WithOptimize_Override(t *testing.T) {
	eng := New(WithOptimize(false))
	if eng.config.optimize {
		t.Fatal("WithOptimize(false): optimize still true")
	}
	if eng.config.constantFolding || eng.config.strengthReduce {
		t.Fatal("WithOptimize(false) should clear constantFolding and strengthReduce")
	}
}

func TestOptions_WithConstantFolding_Override(t *testing.T) {
	eng := New(WithConstantFolding(false))
	if eng.config.constantFolding {
		t.Fatal("WithConstantFolding(false): still true")
	}
	// optimize remains true; only folding toggled
	if !eng.config.optimize {
		t.Fatal("WithConstantFolding should not clear optimize")
	}
}

func TestOptions_WithStrengthReduction_Override(t *testing.T) {
	eng := New(WithStrengthReduction(false))
	if eng.config.strengthReduce {
		t.Fatal("WithStrengthReduction(false): still true")
	}
}

func TestOptions_WithMaxCallDepth_Override(t *testing.T) {
	eng := New(WithMaxCallDepth(42))
	if eng.config.maxCallDepth != 42 {
		t.Fatalf("maxCallDepth=%d want 42", eng.config.maxCallDepth)
	}
	// non-positive ignored
	eng2 := New(WithMaxCallDepth(0))
	if eng2.config.maxCallDepth != defaultMaxCallDepth {
		t.Fatalf("WithMaxCallDepth(0) should keep default %d, got %d", defaultMaxCallDepth, eng2.config.maxCallDepth)
	}
}

func TestOptions_WithStrictMode_Override(t *testing.T) {
	eng := New(WithStrictMode(true))
	if !eng.config.strictMode {
		t.Fatal("WithStrictMode(true): still false")
	}
}

func TestOptions_WithNumericMode_Override(t *testing.T) {
	eng := New(WithNumericMode(NumericFloat64))
	if eng.config.numericMode != NumericFloat64 {
		t.Fatal("WithNumericMode(NumericFloat64): not set")
	}
}

func TestOptions_WithNonFinite_Override(t *testing.T) {
	eng := New(WithNonFinite(NonFiniteError))
	if eng.config.nonFiniteMode != NonFiniteError {
		t.Fatal("WithNonFinite(NonFiniteError): not set")
	}
}

func TestOptions_WithFunction_AndFunctions(t *testing.T) {
	double := func(x int) int { return x * 2 }
	eng := New(WithFunction("double", double))
	if _, ok := eng.config.functions["double"]; !ok {
		t.Fatal("WithFunction did not register double")
	}
	got, err := eng.Eval(context.Background(), `double(21)`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if NewValue(got).Int() != 42 {
		t.Fatalf("got %v want 42", got)
	}

	eng2 := New(WithFunctions(map[string]any{
		"add": func(a, b int) int { return a + b },
	}))
	if _, ok := eng2.config.functions["add"]; !ok {
		t.Fatal("WithFunctions did not register add")
	}
}

func TestOptions_WithOperator_Override(t *testing.T) {
	eng := New(
		WithFunction("concat", func(a, b string) string { return a + b }),
		WithOperator("+", "concat"),
	)
	if eng.config.operators["+"] != "concat" {
		t.Fatalf("operators[+]=%q want concat", eng.config.operators["+"])
	}
}

func TestOptions_WithTypeCheck_Override(t *testing.T) {
	type Env struct {
		Count int `expr:"count"`
	}
	eng := New(WithTypeCheck(Env{}))
	if !eng.config.typeCheck {
		t.Fatal("WithTypeCheck should enable typeCheck")
	}
}

func TestOptions_WithSandbox_AndHelpers(t *testing.T) {
	eng := New(WithSandbox(SandboxConfig{
		MaxOps:   1000,
		Timeout:  50 * time.Millisecond,
		ReadOnly: true,
		Secure:   true,
	}))
	if eng.config.sandbox == nil {
		t.Fatal("WithSandbox: sandbox nil")
	}
	if eng.config.sandbox.MaxOps != 1000 {
		t.Fatalf("MaxOps=%d", eng.config.sandbox.MaxOps)
	}
	if eng.config.sandbox.Timeout != 50*time.Millisecond {
		t.Fatalf("Timeout=%v", eng.config.sandbox.Timeout)
	}
	if !eng.config.sandbox.ReadOnly || !eng.config.sandbox.Secure {
		t.Fatal("ReadOnly/Secure not set")
	}

	eng2 := New(WithMaxOps(77), WithTimeout(5*time.Millisecond), WithReadOnly(true))
	if eng2.config.sandbox == nil || eng2.config.sandbox.MaxOps != 77 {
		t.Fatalf("WithMaxOps: %+v", eng2.config.sandbox)
	}
	if eng2.config.sandbox.Timeout != 5*time.Millisecond {
		t.Fatalf("WithTimeout: %v", eng2.config.sandbox.Timeout)
	}
	if !eng2.config.sandbox.ReadOnly {
		t.Fatal("WithReadOnly")
	}

	eng3 := New(WithSecureSandbox())
	if eng3.config.sandbox == nil || !eng3.config.sandbox.Secure {
		t.Fatal("WithSecureSandbox")
	}

	eng4 := New(WithAllowedFuncs("len", "sum"), WithDeniedFuncs("print"))
	if len(eng4.config.sandbox.AllowedFuncs) != 2 {
		t.Fatalf("AllowedFuncs=%v", eng4.config.sandbox.AllowedFuncs)
	}
	if len(eng4.config.sandbox.DeniedFuncs) != 1 || eng4.config.sandbox.DeniedFuncs[0] != "print" {
		t.Fatalf("DeniedFuncs=%v", eng4.config.sandbox.DeniedFuncs)
	}
}

func TestOptions_CompileOptions_AsTypes(t *testing.T) {
	eng := New()
	p, err := eng.Compile(`1 + 2`, AsInt())
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !p.expectInt {
		t.Fatal("AsInt: expectInt false")
	}
	got, err := p.Eval(context.Background(), nil)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if _, ok := got.(int64); !ok {
		t.Fatalf("AsInt result type %T", got)
	}

	p2, err := eng.Compile(`1 < 2`, AsBool())
	if err != nil {
		t.Fatalf("Compile AsBool: %v", err)
	}
	if !p2.expectBool {
		t.Fatal("AsBool: expectBool false")
	}

	p3, err := eng.Compile(`1.5`, AsFloat())
	if err != nil {
		t.Fatalf("Compile AsFloat: %v", err)
	}
	if !p3.expectFloat {
		t.Fatal("AsFloat")
	}

	p4, err := eng.Compile(`"hi"`, AsString())
	if err != nil {
		t.Fatalf("Compile AsString: %v", err)
	}
	if !p4.expectString {
		t.Fatal("AsString")
	}

	p5, err := eng.Compile(`2 + 2`, Unoptimized())
	if err != nil {
		t.Fatalf("Unoptimized: %v", err)
	}
	_ = p5
}
