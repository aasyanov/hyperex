package hyperex

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestExportedSentinels_HavePackagePrefix(t *testing.T) {
	sentinels := []error{
		ErrDivisionByZero,
		ErrIndexOutOfBounds,
		ErrInvalidOperation,
		ErrNilDereference,
		ErrNotCallable,
		ErrArgumentCount,
		ErrInvalidType,
		ErrNotIterable,
		ErrNotIndexable,
		ErrKeyNotFound,
		ErrInvalidSliceIndex,
		ErrIntegerOverflow,
		ErrInvalidRegex,
		ErrMaxCallDepth,
		ErrOperationCancelled,
		ErrInvalidConversion,
		ErrOpsLimitExceeded,
		ErrMemoryLimitExceeded,
		ErrTimeout,
		ErrFunctionNotAllowed,
		ErrAssignmentNotAllowed,
		ErrStackOverflow,
	}
	for _, s := range sentinels {
		if !strings.HasPrefix(s.Error(), "hyperex: ") {
			t.Errorf("sentinel %v missing hyperex: prefix", s)
		}
	}
}

func TestRuntimeSentinel_IsViaWrapAndSource(t *testing.T) {
	sentinels := []error{
		ErrDivisionByZero,
		ErrIndexOutOfBounds,
		ErrInvalidOperation,
		ErrNilDereference,
		ErrNotCallable,
		ErrArgumentCount,
		ErrInvalidType,
		ErrNotIterable,
		ErrNotIndexable,
		ErrKeyNotFound,
		ErrInvalidSliceIndex,
		ErrIntegerOverflow,
		ErrInvalidRegex,
		ErrMaxCallDepth,
		ErrOperationCancelled,
		ErrInvalidConversion,
	}

	for _, sentinel := range sentinels {
		t.Run(sentinel.Error(), func(t *testing.T) {
			wrapped := wrapRuntime(sentinel)
			if !errors.Is(wrapped, sentinel) {
				t.Fatalf("wrapRuntime: errors.Is(%v, %v) = false", wrapped, sentinel)
			}
			sourced := wrapped.WithSource("expr")
			if !errors.Is(sourced, sentinel) {
				t.Fatalf("WithSource: errors.Is(%v, %v) = false", sourced, sentinel)
			}
			if sourced.Unwrap() != sentinel {
				t.Fatalf("Unwrap() = %v, want %v", sourced.Unwrap(), sentinel)
			}
		})
	}
}

func TestRuntimeError_IsDoesNotMatchByMessageAlone(t *testing.T) {
	adHoc := &RuntimeError{Message: "division by zero"}
	if errors.Is(adHoc, ErrDivisionByZero) {
		t.Fatal("ad-hoc RuntimeError must not match ErrDivisionByZero by Message")
	}
}

func TestSandboxSentinel_IsViaTypedError(t *testing.T) {
	cases := []struct {
		name     string
		err      *SandboxError
		sentinel error
	}{
		{"ops_limit", sandboxErr(ErrOpsLimitExceeded, "ops_limit", ErrOpsLimitExceeded.Error(), 10, 11), ErrOpsLimitExceeded},
		{"memory_limit", sandboxErr(ErrMemoryLimitExceeded, "memory_limit", ErrMemoryLimitExceeded.Error(), 100, 101), ErrMemoryLimitExceeded},
		{"timeout", sandboxErr(ErrTimeout, "timeout", ErrTimeout.Error(), 0, 0), ErrTimeout},
		{"function_denied", sandboxErr(ErrFunctionNotAllowed, "function_denied", ErrFunctionNotAllowed.Error()+": print", 0, 0), ErrFunctionNotAllowed},
		{"assignment_denied", sandboxErr(ErrAssignmentNotAllowed, "assignment_denied", ErrAssignmentNotAllowed.Error(), 0, 0), ErrAssignmentNotAllowed},
		{"stack_overflow", sandboxErr(ErrStackOverflow, "stack_overflow", ErrStackOverflow.Error(), 8, 9), ErrStackOverflow},
		{"type_fallback", &SandboxError{Type: "ops_limit", Message: "ops"}, ErrOpsLimitExceeded},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.sentinel) {
				t.Fatalf("errors.Is(%v, %v) = false", tt.err, tt.sentinel)
			}
			if tt.err.cause != nil && tt.err.Unwrap() != tt.sentinel {
				t.Fatalf("Unwrap() = %v, want %v", tt.err.Unwrap(), tt.sentinel)
			}
		})
	}
}

func TestErrDivisionByZero_EvalPath(t *testing.T) {
	_, err := Eval(context.Background(), `1/0`)
	if !errors.Is(err, ErrDivisionByZero) {
		t.Fatalf("errors.Is(err, ErrDivisionByZero) = false; err=%v", err)
	}
	var re *RuntimeError
	if !errors.As(err, &re) || re.Source == "" {
		t.Fatalf("expected annotated *RuntimeError with Source, got %T %v", err, err)
	}
}

func TestErrIndexOutOfBounds_ValueIndex(t *testing.T) {
	_, err := NewValue([]any{1, 2}).Index(5)
	if !errors.Is(err, ErrIndexOutOfBounds) {
		t.Fatalf("got %v", err)
	}
}

func TestErrNotIndexable_ValueIndex(t *testing.T) {
	_, err := NewValue(true).Index(0)
	if !errors.Is(err, ErrNotIndexable) {
		t.Fatalf("got %v", err)
	}
}

func TestErrMaxCallDepth_Trigger(t *testing.T) {
	engine := New(WithMaxCallDepth(5))
	expr := `
		let f1 = x => x + 1;
		let f2 = x => f1(x) + 1;
		let f3 = x => f2(x) + 1;
		let f4 = x => f3(x) + 1;
		let f5 = x => f4(x) + 1;
		let f6 = x => f5(x) + 1;
		f6(0)
	`
	_, err := engine.Eval(context.Background(), expr)
	if !errors.Is(err, ErrMaxCallDepth) {
		t.Fatalf("errors.Is(err, ErrMaxCallDepth) = false; err=%v", err)
	}
}

func TestErrOpsLimitExceeded_Trigger(t *testing.T) {
	_, err := New(WithMaxOps(1)).Eval(context.Background(), `1+2+3+4+5+6+7+8+9+10`)
	if !errors.Is(err, ErrOpsLimitExceeded) {
		t.Fatalf("got %v", err)
	}
}

func TestErrTimeout_Trigger(t *testing.T) {
	_, err := New(WithTimeout(time.Nanosecond), WithMaxOps(1_000_000_000)).Eval(context.Background(), `map(range(0, 100000), # + 1)`)
	if err == nil {
		t.Fatal("expected timeout or ops limit")
	}
	if !errors.Is(err, ErrTimeout) && !errors.Is(err, ErrOpsLimitExceeded) {
		t.Fatalf("expected ErrTimeout or ErrOpsLimitExceeded, got %v", err)
	}
	if !errors.Is(sandboxErr(ErrTimeout, "timeout", ErrTimeout.Error(), 0, 0), ErrTimeout) {
		t.Fatal("ErrTimeout Is plumbing broken")
	}
}

func TestErrFunctionNotAllowed_Trigger(t *testing.T) {
	_, err := New(WithDeniedFuncs("len")).Eval(context.Background(), `len([1,2,3])`)
	if !errors.Is(err, ErrFunctionNotAllowed) {
		t.Fatalf("got %v", err)
	}
}

func TestErrAssignmentNotAllowed_Trigger(t *testing.T) {
	_, err := New(WithReadOnly(true)).Eval(context.Background(), `let x = 1; x`)
	if !errors.Is(err, ErrAssignmentNotAllowed) {
		t.Fatalf("got %v", err)
	}
}

func TestErrArgumentCount_Builtin(t *testing.T) {
	_, err := Eval(context.Background(), `abs()`)
	if !errors.Is(err, ErrArgumentCount) {
		t.Fatalf("errors.Is(err, ErrArgumentCount) = false; err=%v", err)
	}
}

func TestErrNotCallable_Trigger(t *testing.T) {
	_, err := Eval(context.Background(), `1()`)
	if !errors.Is(err, ErrNotCallable) {
		t.Fatalf("got %v", err)
	}
}

func TestErrInvalidRegex_Trigger(t *testing.T) {
	_, err := Eval(context.Background(), `"abc" ~= "["`)
	if !errors.Is(err, ErrInvalidRegex) {
		t.Fatalf("got %v", err)
	}
}

func TestErrInvalidConversion_StrictMode(t *testing.T) {
	_, err := New(WithStrictMode(true)).Eval(context.Background(), `"abc" - 1`)
	if !errors.Is(err, ErrInvalidConversion) {
		t.Fatalf("errors.Is(err, ErrInvalidConversion) = false; err=%v", err)
	}
}

func TestErrInvalidSliceIndex_Trigger(t *testing.T) {
	_, err := Eval(context.Background(), `[1,2,3][::0]`)
	if !errors.Is(err, ErrInvalidSliceIndex) {
		t.Fatalf("got %v", err)
	}
}

func TestAnnotateEvalError_PreservesSandbox(t *testing.T) {
	orig := sandboxErr(ErrOpsLimitExceeded, "ops_limit", ErrOpsLimitExceeded.Error(), 1, 2)
	if annotateEvalError(orig, "1+1") != orig {
		t.Fatal("sandbox errors must be returned as-is")
	}
}

func TestOptionNilIgnored(t *testing.T) {
	engine := New(nil, WithStrictMode(true))
	if !engine.config.strictMode {
		t.Fatal("expected WithStrictMode to apply when mixed with nil options")
	}
	engine2 := New(WithStrictMode(true), nil)
	if !engine2.config.strictMode {
		t.Fatal("expected trailing nil Option to be ignored")
	}
}
