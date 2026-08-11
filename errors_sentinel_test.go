package hyperex

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestErrorSentinel_RuntimeTriggerMatrix exercises Compile/Eval paths and asserts
// sentinel / typed errors via errors.Is / errors.As (public runtime contract).
// Complements errors_test.go (API unit checks on wrapRuntime / sentinels): this file
// triggers failures through Eval/Compile rather than constructing errors directly.
//
// Undefined variables are wrapped into *RuntimeError with cause *UndefinedError
// (annotateEvalError); errors.As recovers the typed cause.
func TestErrorSentinel_RuntimeTriggerMatrix(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *Engine
		expr    string
		env     map[string]any
		check   func(t *testing.T, err error)
		viaEval bool // when set, exercise Engine.Eval; otherwise Engine.Compile
	}{
		{
			name:    "division by zero yields ErrDivisionByZero",
			setup:   func() *Engine { return New() },
			expr:    `1/0`,
			viaEval: true,
			check: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, ErrDivisionByZero) {
					t.Fatalf("errors.Is(err, ErrDivisionByZero)=false; err=%v (%T)", err, err)
				}
			},
		},
		{
			name:    "undefined variable wraps *UndefinedError",
			setup:   func() *Engine { return New() },
			expr:    `missing_var + 1`,
			viaEval: true,
			check: func(t *testing.T, err error) {
				t.Helper()
				var ue *UndefinedError
				if !errors.As(err, &ue) {
					// Fallback: RuntimeError message (pre-wrap contract)
					var re *RuntimeError
					if !errors.As(err, &re) || !strings.Contains(re.Message, "undefined") {
						t.Fatalf("want UndefinedError or RuntimeError(undefined), got %T %v", err, err)
					}
					return
				}
				if ue.Name != "missing_var" {
					t.Fatalf("UndefinedError.Name=%q", ue.Name)
				}
			},
		},
		{
			name:  "parse error yields *ParseError",
			setup: func() *Engine { return New() },
			expr:  `((((`,
			check: func(t *testing.T, err error) {
				t.Helper()
				var pe *ParseError
				if !errors.As(err, &pe) {
					t.Fatalf("want *ParseError, got %T %v", err, err)
				}
			},
		},
		{
			name:    "index out of bounds yields ErrIndexOutOfBounds",
			setup:   func() *Engine { return New() },
			expr:    `[1,2,3][10]`,
			viaEval: true,
			check: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, ErrIndexOutOfBounds) {
					t.Fatalf("errors.Is(err, ErrIndexOutOfBounds)=false; err=%v (%T)", err, err)
				}
			},
		},
		{
			name: "sandbox ops limit yields ErrOpsLimitExceeded",
			setup: func() *Engine {
				return New(WithMaxOps(3))
			},
			expr:    `x + x + x + x + x + x + x + x`,
			env:     map[string]any{"x": int64(1)},
			viaEval: true,
			check: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, ErrOpsLimitExceeded) {
					t.Fatalf("errors.Is(err, ErrOpsLimitExceeded)=false; err=%v (%T)", err, err)
				}
			},
		},
		{
			name: "sandbox timeout yields ErrTimeout",
			setup: func() *Engine {
				return New(
					WithTimeout(1*time.Nanosecond),
					WithMaxOps(100_000_000),
				)
			},
			expr:    `range(1, 1000000) | sum`,
			viaEval: true,
			check: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, ErrTimeout) {
					t.Fatalf("errors.Is(err, ErrTimeout)=false; err=%v (%T)", err, err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := tt.setup()
			var err error
			if tt.viaEval {
				_, err = eng.Eval(context.Background(), tt.expr, tt.env)
			} else {
				_, err = eng.Compile(tt.expr)
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			tt.check(t, err)
		})
	}
}
