package hyperex

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestSafeFunction_MaxOpsExceeded(t *testing.T) {
	// High tax + tight MaxOps: a few calls must trip the limit.
	eng := New(
		WithMaxOps(50),
		WithSafeFunctionMeta("tick", func() int { return 1 }, SafeFuncMeta{MaxOwnOps: 20}), // tax 21
	)
	_, err := eng.Eval(context.Background(), `tick() + tick() + tick()`)
	if err == nil {
		t.Fatal("expected MaxOps exceeded")
	}
	if !errors.Is(err, ErrOpsLimitExceeded) {
		t.Fatalf("expected ErrOpsLimitExceeded, got %v", err)
	}
}

func TestSafeFunction_TightLoop(t *testing.T) {
	eng := New(
		WithMaxOps(200),
		WithSafeFunction("bump", func(x int) int { return x + 1 }),
	)
	// map over a range calls SafeFunction many times -> ops tax accumulates.
	_, err := eng.Eval(context.Background(), `range(0, 40) | map(x => bump(x)) | sum`)
	if err == nil {
		t.Fatal("expected MaxOps exceeded in tight SafeFunction loop")
	}
	if !errors.Is(err, ErrOpsLimitExceeded) {
		t.Fatalf("expected ErrOpsLimitExceeded, got %v", err)
	}
}

func TestWithFunction_StillTrustedNoForcedTax(t *testing.T) {
	// Same MaxOps that fails SafeFunction should still allow trusted WithFunction
	// for a few calls (no forced tax beyond bytecode).
	eng := New(
		WithMaxOps(50),
		WithFunction("tick", func() int { return 1 }),
	)
	got, err := eng.Eval(context.Background(), `tick() + tick() + tick()`)
	if err != nil {
		t.Fatalf("WithFunction should work without forced tax: %v", err)
	}
	if NewValue(got).Int() != 3 {
		t.Fatalf("got %v want 3", got)
	}
}

func TestSafeFunction_PresetSecureAuto(t *testing.T) {
	eng := New(append(PresetSecureAuto(),
		WithSafeFunction("tagGood", func(q uint32) bool {
			return QualityClass(q) == QualityClassGood
		}),
	)...)
	got, err := eng.EvalBool(context.Background(), `tagGood(0)`)
	if err != nil {
		t.Fatalf("PresetSecureAuto + WithSafeFunction: %v", err)
	}
	if !got {
		t.Fatal("tagGood(0) want true")
	}
	// print still denied by preset
	_, err = eng.Eval(context.Background(), `print("x")`)
	if err == nil || !errors.Is(err, ErrFunctionNotAllowed) {
		t.Fatalf("print should remain denied, got %v", err)
	}
}

func TestSafeFunction_DeniedFuncs(t *testing.T) {
	eng := New(
		WithDeniedFuncs("tagGood"),
		WithSafeFunction("tagGood", func() bool { return true }),
	)
	_, err := eng.Eval(context.Background(), `tagGood()`)
	if err == nil || !errors.Is(err, ErrFunctionNotAllowed) {
		t.Fatalf("DeniedFuncs should apply to SafeFunction, got %v", err)
	}
}

func TestConcurrency_SafeFunction_CompileEval(t *testing.T) {
	eng := New(
		WithSafeFunction("double", func(x int64) int64 { return x * 2 }),
	)
	prog, err := eng.Compile(`double(n)`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	const goroutines = 24
	const iters = 100
	var fails atomic.Int64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				if id%2 == 0 {
					p, err := eng.Compile(`double(n) + 1`)
					if err != nil {
						fails.Add(1)
						t.Errorf("Compile: %v", err)
						return
					}
					got, err := p.Eval(context.Background(), map[string]any{"n": int64(id)})
					if err != nil {
						fails.Add(1)
						t.Errorf("Eval compiled: %v", err)
						return
					}
					if NewValue(got).Int() != int64(id)*2+1 {
						fails.Add(1)
						t.Errorf("got %v", got)
						return
					}
				} else {
					got, err := prog.Eval(context.Background(), map[string]any{"n": int64(id)})
					if err != nil {
						fails.Add(1)
						t.Errorf("Eval shared: %v", err)
						return
					}
					if NewValue(got).Int() != int64(id)*2 {
						fails.Add(1)
						t.Errorf("got %v", got)
						return
					}
				}
			}
		}(g)
	}
	wg.Wait()
	if fails.Load() != 0 {
		t.Fatalf("%d goroutine failures", fails.Load())
	}
}

func TestSafeFunction_PanicBecomesRuntimeError(t *testing.T) {
	eng := New(WithSafeFunction("boom", func() int {
		panic("explode")
	}))
	_, err := eng.Eval(context.Background(), `boom()`)
	if err == nil {
		t.Fatal("expected RuntimeError from panic")
	}
	var re *RuntimeError
	if !errors.As(err, &re) {
		t.Fatalf("expected RuntimeError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// Domain edge proofs -> safe_function_test.go
// ---------------------------------------------------------------------------

func TestModeB_SafeFunction_AllowedAndDenied(t *testing.T) {
	ctx := context.Background()

	// Allow-list: SafeFunction on list works; off-list denied.
	eng := New(
		WithAllowedFuncs("ok", "len"),
		WithSafeFunction("ok", func() int { return 7 }),
		WithSafeFunction("nope", func() int { return 9 }),
	)
	got, err := eng.Eval(ctx, `ok()`)
	if err != nil {
		t.Fatalf("allowed SafeFunction: %v", err)
	}
	if NewValue(got).Int() != 7 {
		t.Fatalf("got %v want 7", got)
	}
	_, err = eng.Eval(ctx, `nope()`)
	if err == nil || !errors.Is(err, ErrFunctionNotAllowed) {
		t.Fatalf("nope should be denied by AllowedFuncs, got %v", err)
	}

	// DeniedFuncs takes precedence over registration.
	eng2 := New(
		WithDeniedFuncs("blocked"),
		WithSafeFunction("blocked", func() int { return 1 }),
		WithSafeFunction("free", func() int { return 2 }),
	)
	_, err = eng2.Eval(ctx, `blocked()`)
	if err == nil || !errors.Is(err, ErrFunctionNotAllowed) {
		t.Fatalf("DeniedFuncs SafeFunction: %v", err)
	}
	got, err = eng2.Eval(ctx, `free()`)
	if err != nil || NewValue(got).Int() != 2 {
		t.Fatalf("free SafeFunction: %v %v", got, err)
	}

	// PresetSecureAuto + AllowedFuncs narrows further.
	eng3 := New(append(PresetSecureAuto(),
		WithAllowedFuncs("tag", "isGood", "v", "q"),
		WithSafeFunction("tag", func() bool { return true }),
	)...)
	ok, err := eng3.EvalBool(ctx, `tag()`)
	if err != nil || !ok {
		t.Fatalf("preset+allow SafeFunction: %v %v", ok, err)
	}
	_, err = eng3.Eval(ctx, `len([1])`)
	if err == nil || !errors.Is(err, ErrFunctionNotAllowed) {
		t.Fatalf("len should be outside allow list: %v", err)
	}
}
