package hyperex

import (
	"context"
	"strings"
	"testing"
)

// =============================================================================
// ERROR TESTS
// =============================================================================

func TestError_Strings(t *testing.T) {
	errors := []error{
		&LexerError{Message: "test", Line: 1, Column: 5},
		&ParseError{Message: "test", token: token{Type: tokenInt, Line: 1, Column: 1}},
		&TypeError{Message: "test"},
		&RuntimeError{Message: "test", Op: "+"},
		&RuntimeError{Message: "test"},
		&UndefinedError{Name: "x", Kind: "variable"},
		&CompileError{Message: "test"},
	}

	for _, err := range errors {
		s := err.Error()
		if s == "" {
			t.Errorf("empty error string for %T", err)
		}
	}
}

func TestRuntime_Errors(t *testing.T) {
	tests := []struct {
		name string
		expr string
		env  map[string]any
	}{
		{"div_zero", "1 / 0", nil},
		{"mod_zero", "1 % 0", map[string]any{}},
		{"index_bounds", "arr[10]", map[string]any{"arr": []any{1, 2}}},
		{"undefined_var", "undefined_variable", nil},
		{"not_callable", "x()", map[string]any{"x": 42}},
		{"invalid_regex", `"test" ~= "["`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Eval(context.Background(), tt.expr, tt.env)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

// =============================================================================
// ERROR CONTEXT TESTS
// =============================================================================

func TestError_RuntimeErrorWithSource(t *testing.T) {
	// Test that runtime errors include source expression
	_, err := Eval(context.Background(), "x + y", map[string]any{"x": 10}) // y is undefined
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "x + y") {
		t.Errorf("error should contain source expression, got: %s", errStr)
	}
}

func TestError_LexerErrorWithSource(t *testing.T) {
	// Test that lexer errors include source
	lexer := newLexer(`"unterminated string`)
	_, err := lexer.tokenize()
	if err == nil {
		t.Fatal("expected error")
	}
	lexErr, ok := err.(*LexerError)
	if !ok {
		t.Fatalf("expected LexerError, got %T", err)
	}
	if lexErr.Source == "" {
		t.Error("LexerError.Source should be set")
	}
}

func TestError_ParseErrorWithSource(t *testing.T) {
	// Test that parse errors include source
	_, err := Compile("x ++ y") // invalid syntax
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	// Parse error should have context
	if !strings.Contains(errStr, "x ++ y") {
		t.Errorf("error should contain source expression, got: %s", errStr)
	}
}

func TestError_DivisionByZero(t *testing.T) {
	_, err := Eval(context.Background(), "10 / 0")
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "division by zero") {
		t.Errorf("error should mention division by zero, got: %s", errStr)
	}
	if !strings.Contains(errStr, "10 / 0") {
		t.Errorf("error should contain source expression, got: %s", errStr)
	}
}

func TestProgram_EvalBool(t *testing.T) {
	program := MustCompile("x > 5")

	b, err := program.EvalBool(context.Background(), map[string]any{"x": int64(10)})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !b {
		t.Error("expected true")
	}

	b, err = program.EvalBool(context.Background(), map[string]any{"x": int64(3)})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if b {
		t.Error("expected false")
	}
}
