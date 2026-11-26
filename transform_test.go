package hyperex

import (
	"context"
	"strings"
	"testing"
)

// =============================================================================
// TRANSFORM TESTS
// =============================================================================

func TestSimplify_Constants(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2 + 3", "5"},
		{"2 * 3 + 4", "10"},
		{"10 / 2", "5"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Simplify(tt.input)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Simplify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSimplify_Algebraic(t *testing.T) {
	tests := []struct {
		input    string
		contains string // Result should contain this
	}{
		{"x + 0", "x"}, // may stay "x + 0" (ident not proven numeric) or fold
		{"x * 1", "x"},
		{"(1+2) + 0", "3"}, // numeric - foldable via const fold / algebra
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Simplify(tt.input)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Simplify(%q) = %q, should contain %q", tt.input, result, tt.contains)
			}
		})
	}
}

func TestOptimizer_PlusZero_PreservesStringConcat(t *testing.T) {
	// Algebraic 0+x / x+0 must not apply when + is string concat.
	env := map[string]any{"s": "hello"}
	cases := []struct {
		expr string
		want string
	}{
		{`0+"hello"`, "0hello"},
		{`"hello"+0`, "hello0"},
		{`0+s`, "0hello"},
		{`s+0`, "hello0"},
	}
	for _, tt := range cases {
		t.Run(tt.expr, func(t *testing.T) {
			for _, opt := range []bool{true, false} {
				eng := New(WithOptimize(opt))
				got, err := eng.Eval(context.Background(), tt.expr, env)
				if err != nil {
					t.Fatalf("optimize=%v: %v", opt, err)
				}
				if got != tt.want {
					t.Fatalf("optimize=%v: got %v, want %q", opt, got, tt.want)
				}
			}
		})
	}
}

func TestOptimizer_Logical_PreserveOperandSemantics(t *testing.T) {
	// && / || return an operand (LANGUAGE.md), not a coerced bool.
	cases := []struct {
		expr string
		want any
	}{
		{"1 && true", true},
		{"true && 1", int64(1)},
		{"0 || false", false},
		{"false || 0", int64(0)},
		{"1 || true", int64(1)},
		{"true || 1", true},
		{"0 && true", int64(0)},
		{`"hi" && true`, true},
		{`true && "hi"`, "hi"},
	}
	for _, tt := range cases {
		t.Run(tt.expr, func(t *testing.T) {
			for _, opt := range []bool{true, false} {
				eng := New(WithOptimize(opt))
				got, err := eng.Eval(context.Background(), tt.expr, nil)
				if err != nil {
					t.Fatalf("optimize=%v: %v", opt, err)
				}
				if !NewValue(got).Equal(NewValue(tt.want)) {
					t.Fatalf("optimize=%v: got %v (%T), want %v (%T)", opt, got, got, tt.want, tt.want)
				}
			}
		})
	}
}

func TestOptimizer_DoubleBitNot_PreservesBoolCoercion(t *testing.T) {
	// ~~x -> x is only valid for integers; ~ coerces bool via Int().
	cases := []struct {
		expr string
		want any
	}{
		{"~~!0", int64(1)},
		{"~~!00", int64(1)},
		{"~~true", int64(1)},
		{"~~false", int64(0)},
		{"~~0", int64(0)},
		{"~~1", int64(1)},
		{"~~-1", int64(-1)},
		{"~~.1", int64(0)},
		{"~~1.9", int64(1)},
	}
	for _, tt := range cases {
		t.Run(tt.expr, func(t *testing.T) {
			for _, opt := range []bool{true, false} {
				eng := New(WithOptimize(opt))
				got, err := eng.Eval(context.Background(), tt.expr, nil)
				if err != nil {
					t.Fatalf("optimize=%v: %v", opt, err)
				}
				if !NewValue(got).Equal(NewValue(tt.want)) {
					t.Fatalf("optimize=%v: got %v (%T), want %v (%T)", opt, got, got, tt.want, tt.want)
				}
			}
		})
	}
}

func TestExpand_Macros(t *testing.T) {
	macros := map[string]string{
		"ADULT": "age >= 18",
		"USA":   `country == "US"`,
	}

	result, err := Expand("ADULT", macros)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(result, "age") || !strings.Contains(result, "18") {
		t.Errorf("Expand did not expand ADULT macro: %s", result)
	}
}

func TestExpand_Combined(t *testing.T) {
	macros := map[string]string{
		"A": "x > 0",
		"B": "y < 10",
	}

	result, err := Expand("A && B", macros)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Should contain both expanded macros
	if !strings.Contains(result, "x") || !strings.Contains(result, "y") {
		t.Errorf("Expand did not expand both macros: %s", result)
	}
}

func TestInline_Let(t *testing.T) {
	result, err := Inline("let x = 5; x + x")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Should not contain "let"
	if strings.Contains(result, "let") {
		t.Errorf("Inline should remove let: %s", result)
	}
	// Should contain "5"
	if !strings.Contains(result, "5") {
		t.Errorf("Inline should substitute value: %s", result)
	}
}

func TestInline_NestedLet(t *testing.T) {
	result, err := Inline("let x = 2; let y = x + 1; y * 2")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	got, err := Eval(context.Background(), result)
	if err != nil || got != int64(6) {
		t.Fatalf("Inline changed nested-let semantics: %q -> (%#v, %v)", result, got, err)
	}
}

func TestASTToString_Roundtrip(t *testing.T) {
	exprs := []string{
		"1 + 2",
		"x * y",
		"a && b || c",
		"x > 0 ? 1 : 0",
		"x ?? 0",
		"[1, 2, 3]",
		`{"a": 1}`,
		"sum(arr)",
		"obj.field",
		"arr[0]",
	}

	for _, expr := range exprs {
		t.Run(expr, func(t *testing.T) {
			parser := newParser(expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			result := ASTToString(ast)
			// The result should be parseable
			parser2 := newParser(result)
			_, err = parser2.parse()
			if err != nil {
				t.Errorf("ASTToString produced unparseable result: %s (from %s)", result, expr)
			}
		})
	}
}

func TestExtractVariables(t *testing.T) {
	vars, err := ExtractVariables("x + y * z")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(vars) != 3 {
		t.Errorf("expected 3 variables, got %d: %v", len(vars), vars)
	}
}

func TestExtractFunctions(t *testing.T) {
	funcs, err := ExtractFunctions("sum(arr) + len(s)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(funcs) != 2 {
		t.Errorf("expected 2 functions, got %d: %v", len(funcs), funcs)
	}
}

func TestValidate(t *testing.T) {
	// Valid expressions
	if err := Validate("1 + 2"); err != nil {
		t.Errorf("valid expression failed: %v", err)
	}

	// Invalid expressions
	if err := Validate("1 +"); err == nil {
		t.Error("invalid expression should fail")
	}
}

func TestValidateWithEnv(t *testing.T) {
	type Env struct {
		X int `expr:"x"`
	}

	// Valid
	if err := ValidateWithEnv("x + 1", Env{}); err != nil {
		t.Errorf("valid expression failed: %v", err)
	}
}

func TestPrint_Builtin(t *testing.T) {
	// print just outputs to stdout and returns nil
	result, err := Eval(context.Background(), "print(42)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != nil {
		t.Errorf("got %v, want nil", result)
	}
}
