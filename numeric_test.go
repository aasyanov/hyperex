package hyperex

import (
	"context"
	"errors"
	"math"
	"testing"
)

func TestNumericFloat64_IntAddPromotes(t *testing.T) {
	ctx := context.Background()
	eng := New(WithNumericMode(NumericFloat64), WithOptimize(false))
	got, err := eng.Eval(ctx, `a + b`, map[string]any{"a": int64(1), "b": int64(2)})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	f, ok := got.(float64)
	if !ok {
		t.Fatalf("want float64, got %T (%v)", got, got)
	}
	if f != 3.0 {
		t.Fatalf("got %v want 3.0", f)
	}
}

func TestNumericFloat64_DivHalf(t *testing.T) {
	ctx := context.Background()
	floatEng := New(WithNumericMode(NumericFloat64), WithOptimize(false))
	got, err := floatEng.Eval(ctx, `1 / 2`)
	if err != nil {
		t.Fatalf("Float64 Eval: %v", err)
	}
	f, ok := got.(float64)
	if !ok || f != 0.5 {
		t.Fatalf("Float64 1/2: got %T %v want float64 0.5", got, got)
	}

	defEng := New(WithOptimize(false))
	got, err = defEng.Eval(ctx, `1 / 2`)
	if err != nil {
		t.Fatalf("Default Eval: %v", err)
	}
	n, ok := got.(int64)
	if !ok || n != 0 {
		t.Fatalf("Default 1/2: got %T %v want int64 0", got, got)
	}
}

func TestNumericFloat64_StringConcatUnchanged(t *testing.T) {
	ctx := context.Background()
	eng := New(WithNumericMode(NumericFloat64), WithOptimize(false))
	got, err := eng.Eval(ctx, `"a" + "b"`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	s, ok := got.(string)
	if !ok || s != "ab" {
		t.Fatalf("got %T %v want string \"ab\"", got, got)
	}
}

func TestNumericFloat64_OptimizerFoldsToFloat(t *testing.T) {
	ctx := context.Background()
	eng := New(WithNumericMode(NumericFloat64)) // optimize on
	got, err := eng.Eval(ctx, `1 + 2`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	f, ok := got.(float64)
	if !ok || f != 3.0 {
		t.Fatalf("optimized 1+2: got %T %v want float64 3.0", got, got)
	}
	got, err = eng.Eval(ctx, `1 / 2`)
	if err != nil {
		t.Fatalf("Eval 1/2: %v", err)
	}
	f, ok = got.(float64)
	if !ok || f != 0.5 {
		t.Fatalf("optimized 1/2: got %T %v want float64 0.5", got, got)
	}
}

func TestNonFiniteError_SqrtNeg(t *testing.T) {
	ctx := context.Background()
	eng := New(WithNonFinite(NonFiniteError))
	_, err := eng.Eval(ctx, `sqrt(-1)`)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("errors.Is(err, ErrNonFinite)=false; err=%v", err)
	}
}

func TestNonFiniteError_InfMul(t *testing.T) {
	ctx := context.Background()
	eng := New(WithNonFinite(NonFiniteError), WithOptimize(false))
	_, err := eng.Eval(ctx, `1e308 * 1e308`)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("errors.Is(err, ErrNonFinite)=false; err=%v", err)
	}
}

func TestNonFiniteAllow_SqrtNeg(t *testing.T) {
	ctx := context.Background()
	eng := New() // NonFiniteAllow default
	got, err := eng.Eval(ctx, `sqrt(-1)`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	f, ok := got.(float64)
	if !ok || !math.IsNaN(f) {
		t.Fatalf("want NaN float64, got %T %v", got, got)
	}
}

func TestProgram_EvalValue_Kind(t *testing.T) {
	ctx := context.Background()
	prog := New(WithOptimize(false)).MustCompile(`1 + 2`)
	v, err := prog.EvalValue(ctx, nil)
	if err != nil {
		t.Fatalf("EvalValue: %v", err)
	}
	if !v.IsInt() || v.Int() != 3 {
		t.Fatalf("want int 3, got kind=%s val=%v", v.Type(), v.Interface())
	}

	floatProg := New(WithNumericMode(NumericFloat64), WithOptimize(false)).MustCompile(`a + b`)
	v, err = floatProg.EvalValue(ctx, map[string]any{"a": 1, "b": 2})
	if err != nil {
		t.Fatalf("EvalValue float: %v", err)
	}
	if !v.IsFloat() || v.Float() != 3.0 {
		t.Fatalf("want float 3, got kind=%s val=%v", v.Type(), v.Interface())
	}
}

func TestProgram_ResultHint_AsFloat(t *testing.T) {
	prog, err := Compile(`x`, AsFloat())
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if prog.ResultHint() != ResultHintFloat {
		t.Fatalf("ResultHint=%v want ResultHintFloat", prog.ResultHint())
	}
	if prog.ResultHint().String() != "float" {
		t.Fatalf("String=%q", prog.ResultHint().String())
	}
	anyProg := MustCompile(`1`)
	if anyProg.ResultHint() != ResultHintAny {
		t.Fatalf("default hint=%v", anyProg.ResultHint())
	}
}

func TestPresetSecureCalc_NumericFloat64(t *testing.T) {
	ctx := context.Background()
	eng := NewSecureCalc()
	got, err := eng.Eval(ctx, `a + b`, map[string]any{"a": int64(10), "b": int64(20)})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	f, ok := got.(float64)
	if !ok || f != 30.0 {
		t.Fatalf("PresetSecureCalc a+b: got %T %v want float64 30", got, got)
	}
	if eng.config.numericMode != NumericFloat64 {
		t.Fatalf("numericMode=%v", eng.config.numericMode)
	}
	if eng.config.nonFiniteMode != NonFiniteError {
		t.Fatalf("nonFiniteMode=%v", eng.config.nonFiniteMode)
	}
}

func TestPresetSecureAuto_NumericFloat64(t *testing.T) {
	eng := NewSecureAuto()
	if eng.config.numericMode != NumericFloat64 || eng.config.nonFiniteMode != NonFiniteError {
		t.Fatalf("mode numeric=%v nonFinite=%v", eng.config.numericMode, eng.config.nonFiniteMode)
	}
}

func TestErrorsIs_NonFinite(t *testing.T) {
	ctx := context.Background()
	eng := New(WithNonFinite(NonFiniteError))
	_, err := eng.Eval(ctx, `sqrt(-1)`)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("errors.Is failed: %v", err)
	}
	var re *RuntimeError
	if !errors.As(err, &re) {
		t.Fatalf("want *RuntimeError, got %T", err)
	}
}

func TestNumericDefault_PreservesIntDiv(t *testing.T) {
	// Sanity: default engine path unchanged for int arithmetic.
	ctx := context.Background()
	got, err := New(WithOptimize(false)).Eval(ctx, `3 + 4`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if n, ok := got.(int64); !ok || n != 7 {
		t.Fatalf("got %T %v want int64 7", got, got)
	}
}

func TestConvertStrict_ExpectedGotFields(t *testing.T) {
	strict := New(WithStrictMode(true))
	_, err := strict.EvalBool(context.Background(), `42`)
	if !errors.Is(err, ErrInvalidConversion) {
		t.Fatalf("errors.Is: %v", err)
	}
	var re *RuntimeError
	if !errors.As(err, &re) {
		t.Fatalf("want *RuntimeError, got %T", err)
	}
	if re.Expected != "bool" || re.Got != "int" {
		t.Fatalf("Expected=%q Got=%q", re.Expected, re.Got)
	}
}

// ---------------------------------------------------------------------------
// Domain edge proofs -> numeric_test.go
// ---------------------------------------------------------------------------

func TestModeB_NonFinite_EvalValueAndEdges(t *testing.T) {
	ctx := context.Background()
	eng := New(WithNonFinite(NonFiniteError), WithOptimize(false))

	prog, err := eng.Compile(`sqrt(-1)`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	_, err = prog.EvalValue(ctx, nil)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("EvalValue sqrt(-1): %v", err)
	}

	_, err = eng.Eval(ctx, `log(0)`)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("log(0): %v", err)
	}
	_, err = eng.Eval(ctx, `1 / 0.0`)
	if !errors.Is(err, ErrNonFinite) && !errors.Is(err, ErrDivisionByZero) {
		// Either Inf rejected or div-by-zero - both fail-closed.
		t.Fatalf("1/0.0 under NonFiniteError: %v", err)
	}

	// Allow mode still returns NaN via EvalValue
	allow := New(WithNonFinite(NonFiniteAllow), WithOptimize(false))
	p2 := allow.MustCompile(`sqrt(-1)`)
	v, err := p2.EvalValue(ctx, nil)
	if err != nil {
		t.Fatalf("Allow EvalValue: %v", err)
	}
	if !v.IsFloat() || !math.IsNaN(v.Float()) {
		t.Fatalf("want NaN, got %v", v)
	}

	// ResultHint String coverage for all As*
	hints := []struct {
		opt  CompileOption
		want ResultHint
		str  string
	}{
		{AsBool(), ResultHintBool, "bool"},
		{AsInt(), ResultHintInt, "int"},
		{AsFloat(), ResultHintFloat, "float"},
		{AsString(), ResultHintString, "string"},
	}
	for _, h := range hints {
		p, err := Compile(`1`, h.opt)
		if err != nil {
			t.Fatalf("Compile: %v", err)
		}
		if p.ResultHint() != h.want || p.ResultHint().String() != h.str {
			t.Fatalf("hint=%v %q want %v %q", p.ResultHint(), p.ResultHint().String(), h.want, h.str)
		}
	}
	if ResultHint(99).String() == "" {
		t.Fatal("unknown ResultHint.String empty")
	}
}
