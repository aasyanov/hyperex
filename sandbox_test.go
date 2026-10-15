package hyperex

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// SANDBOX TESTS
// =============================================================================

func TestSandbox_MaxOps(t *testing.T) {
	env := map[string]any{"x": int64(1)}

	// Simple expression should work with enough ops
	engine := New(WithMaxOps(100))
	result, err := engine.Eval(context.Background(), "x + x + x", env)
	if err != nil {
		t.Fatalf("simple expression failed: %v", err)
	}
	if NewValue(result).Int() != 3 {
		t.Errorf("expected 3, got %v", result)
	}

	// With very low limit, even simple expressions should fail
	engine2 := New(WithMaxOps(3))
	_, err = engine2.Eval(context.Background(), "x + x + x + x + x + x + x + x", env)
	if err == nil {
		t.Error("expected ops limit error")
	}
	if !errors.Is(err, ErrOpsLimitExceeded) {
		t.Errorf("expected ErrOpsLimitExceeded, got %v", err)
	}
}

func TestSandbox_Timeout(t *testing.T) {
	engine := New(WithTimeout(10 * time.Millisecond))

	// Quick expression should work
	result, err := engine.Eval(context.Background(), "1 + 1")
	if err != nil {
		t.Fatalf("quick expression failed: %v", err)
	}
	if NewValue(result).Int() != 2 {
		t.Errorf("expected 2, got %v", result)
	}
}

func TestSandbox_Timeout_Exceeded(t *testing.T) {
	engine := New(
		WithTimeout(1*time.Nanosecond),
		WithMaxOps(100_000_000),
	)
	_, err := engine.Eval(context.Background(), `range(1, 1000000) | sum`)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("errors.Is(err, ErrTimeout)=false; err=%v (%T)", err, err)
	}
}

func TestSandbox_MaxMemory_Exceeded(t *testing.T) {
	// MaxMemory estimates env Values + stack slots (not range heap until walked via env).
	engine := New(WithSandbox(SandboxConfig{MaxMemory: 100}))
	big := strings.Repeat("x", 10_000)
	_, err := engine.Eval(context.Background(), "x", map[string]any{"x": big})
	if err == nil {
		t.Fatal("expected memory limit error")
	}
	if !errors.Is(err, ErrMemoryLimitExceeded) {
		t.Fatalf("errors.Is(err, ErrMemoryLimitExceeded)=false; err=%v (%T)", err, err)
	}
}

func TestSandbox_StackOverflow_Exceeded(t *testing.T) {
	engine := New(WithSandbox(SandboxConfig{MaxStackDepth: 1}))
	_, err := engine.Eval(context.Background(), "1+1")
	if err == nil {
		t.Fatal("expected stack overflow error")
	}
	if !errors.Is(err, ErrStackOverflow) {
		t.Fatalf("errors.Is(err, ErrStackOverflow)=false; err=%v (%T)", err, err)
	}
}

func TestSandbox_DeniedFuncs(t *testing.T) {
	engine := New(WithDeniedFuncs("print", "sprintf"))

	// Allowed functions should work
	result, err := engine.Eval(context.Background(), "len([1, 2, 3])")
	if err != nil {
		t.Fatalf("len should work: %v", err)
	}
	if NewValue(result).Int() != 3 {
		t.Errorf("expected 3, got %v", result)
	}

	// Denied function should fail
	_, err = engine.Eval(context.Background(), `print("hello")`)
	if err == nil {
		t.Error("expected function denied error")
	}
	if !errors.Is(err, ErrFunctionNotAllowed) {
		t.Errorf("expected ErrFunctionNotAllowed, got %v", err)
	}
}

func TestSandbox_AllowedFuncs(t *testing.T) {
	engine := New(WithAllowedFuncs("len", "sum"))

	// Allowed functions should work
	result, err := engine.Eval(context.Background(), "len([1, 2, 3])")
	if err != nil {
		t.Fatalf("len should work: %v", err)
	}
	if NewValue(result).Int() != 3 {
		t.Errorf("expected 3, got %v", result)
	}

	// Non-allowed function should fail
	_, err = engine.Eval(context.Background(), `upper("hello")`)
	if err == nil {
		t.Error("expected function denied error")
	}
	if !errors.Is(err, ErrFunctionNotAllowed) {
		t.Errorf("expected ErrFunctionNotAllowed, got %v", err)
	}
}

func TestSandbox_EnvFunction_BypassBlocked(t *testing.T) {
	engine := New(WithAllowedFuncs("len"))

	env := map[string]any{
		"evil": func() int { return 123 },
	}

	_, err := engine.Eval(context.Background(), `evil()`, env)
	if err == nil {
		t.Error("expected function denied error for env function under allowlist")
	}
	if !errors.Is(err, ErrFunctionNotAllowed) {
		t.Errorf("expected ErrFunctionNotAllowed, got %v", err)
	}
}

func TestSandbox_CustomFunction_AllowedByName(t *testing.T) {
	engine := New(
		WithAllowedFuncs("double"),
		WithFunction("double", func(x int) int { return x * 2 }),
	)

	result, err := engine.Eval(context.Background(), `double(21)`, nil)
	if err != nil {
		t.Fatalf("double should be allowed: %v", err)
	}
	if NewValue(result).Int() != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestSandbox_SecureEnv_RejectsStructPointer(t *testing.T) {
	type Env struct {
		Secret string
	}
	engine := New(WithSandbox(SandboxConfig{Secure: true}))
	_, err := engine.Eval(context.Background(), `x`, map[string]any{"x": &Env{Secret: "nope"}})
	if err == nil {
		t.Error("expected secure sandbox to reject struct pointer in environment")
	}
	var sandboxErr *SandboxError
	if !errors.As(err, &sandboxErr) || sandboxErr.Type != "access_denied" {
		t.Errorf("expected access_denied SandboxError, got %v", err)
	}
}

func TestSandbox_ReadOnly(t *testing.T) {
	engine := New(WithReadOnly(true))

	// Reading should work
	result, err := engine.Eval(context.Background(), "x + 1", map[string]any{"x": 5})
	if err != nil {
		t.Fatalf("reading should work: %v", err)
	}
	if NewValue(result).Int() != 6 {
		t.Errorf("expected 6, got %v", result)
	}

	// Let expressions (assignment) should fail
	_, err = engine.Eval(context.Background(), "let y = 5; y + 1")
	if err == nil {
		t.Error("expected assignment denied error")
	}
	if !errors.Is(err, ErrAssignmentNotAllowed) {
		t.Errorf("expected ErrAssignmentNotAllowed, got %v", err)
	}
}

func TestSandbox_Combined(t *testing.T) {
	engine := New(
		WithSandbox(SandboxConfig{
			MaxOps:      10000,
			Timeout:     time.Second,
			DeniedFuncs: []string{"print"},
			ReadOnly:    true,
		}),
	)

	// Normal expression should work
	result, err := engine.Eval(context.Background(), "1 + 2 * 3")
	if err != nil {
		t.Fatalf("normal expression failed: %v", err)
	}
	if NewValue(result).Int() != 7 {
		t.Errorf("expected 7, got %v", result)
	}

	// Denied function should fail
	_, err = engine.Eval(context.Background(), `print("test")`)
	if err == nil {
		t.Error("expected function denied error")
	}

	// Assignment should fail
	_, err = engine.Eval(context.Background(), "let x = 5; x")
	if err == nil {
		t.Error("expected assignment denied error")
	}
}

func TestSandboxError_Is(t *testing.T) {
	tests := []struct {
		errType string
		target  error
		want    bool
	}{
		{"ops_limit", ErrOpsLimitExceeded, true},
		{"memory_limit", ErrMemoryLimitExceeded, true},
		{"timeout", ErrTimeout, true},
		{"function_denied", ErrFunctionNotAllowed, true},
		{"assignment_denied", ErrAssignmentNotAllowed, true},
		{"stack_overflow", ErrStackOverflow, true},
		{"ops_limit", ErrTimeout, false},
		{"unknown", ErrOpsLimitExceeded, false},
	}

	for _, tt := range tests {
		t.Run(tt.errType, func(t *testing.T) {
			err := &SandboxError{Type: tt.errType, Message: "test"}
			if got := errors.Is(err, tt.target); got != tt.want {
				t.Errorf("SandboxError{Type: %q}.Is(%v) = %v, want %v", tt.errType, tt.target, got, tt.want)
			}
		})
	}
}

func TestSandboxError_Error(t *testing.T) {
	err := &SandboxError{
		Type:    "ops_limit",
		Message: "hyperex: operations limit exceeded",
		Limit:   1000,
		Actual:  1001,
	}
	if err.Error() != "hyperex: operations limit exceeded" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Domain edge proofs -> sandbox_test.go
// ---------------------------------------------------------------------------

func TestModeB_NewSecureHelpers(t *testing.T) {
	c := NewSecureCalc()
	a := NewSecureAuto()
	if c == nil || a == nil {
		t.Fatal("nil engine")
	}
	got, err := c.Eval(context.Background(), `1 + 2`)
	if err != nil {
		t.Fatal(err)
	}
	if NewValue(got).Float() != 3 {
		t.Fatalf("SecureCalc NumericFloat64: %v", got)
	}
}
