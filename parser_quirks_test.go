package hyperex

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// A. Pipe vs bitwise OR
// ---------------------------------------------------------------------------

func TestPipeVsBitwise_ParseHeuristic(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		wantPipe bool
	}{
		{"arr_pipe_sum", "arr | sum", true},
		{"xs_pipe_filter_call", "xs | filter(# > 0)", true},
		{"x_bitwise_y", "x | y", false},
		{"flags_bitwise_MASK", "flags | MASK", false},
		{"data_pipe_unique_builtin", "data | unique", true},
		{"array_lit_pipe_sum", "[1, 2, 3] | sum", true},
		{"int_lit_never_pipe", "1 | 2", false},
		{"float_lit_never_pipe", "1.5 | 2.5", false},
		{"ident_pipe_call", "data | filter(f)", true},
		{"member_pipe_ident", "a.b | trim", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := newParser(tt.expr).parse()
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			_, isPipe := ast.(*pipeExpr)
			be, isBin := ast.(*binaryExpr)
			if tt.wantPipe {
				if !isPipe {
					t.Fatalf("want pipeExpr, got %T (%s)", ast, ASTToString(ast))
				}
				return
			}
			if !isBin || be.Op != "|" {
				t.Fatalf("want binaryExpr (|), got %T (%s)", ast, ASTToString(ast))
			}
		})
	}
}

func TestPipeVsBitwise_Eval(t *testing.T) {
	tests := []struct {
		name string
		expr string
		env  map[string]any
		want any
	}{
		{"arr_sum", "arr | sum", map[string]any{"arr": []any{1, 2, 3}}, int64(6)},
		{"data_unique", "data | unique", map[string]any{"data": []any{1, 2, 2, 3}}, []any{1, 2, 3}},
		{"xs_filter", "xs | filter(x => x > 0)", map[string]any{"xs": []any{-1, 2, 0, 3}}, []any{2, 3}},
		{"x_or_y", "x | y", map[string]any{"x": 1, "y": 2}, int64(3)},
		{"flags_or_mask", "flags | MASK", map[string]any{"flags": 1, "MASK": 4}, int64(5)},
		{"lit_or", "1 | 2", nil, int64(3)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Eval(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("eval: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v (%T), want %#v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestPipeVsBitwise_AnalysisWarnings(t *testing.T) {
	t.Run("ambiguous_ident_or", func(t *testing.T) {
		a, err := Analyze("x | y")
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if !containsWarning(a.Warnings, warnAmbiguousBitwiseOr) {
			t.Fatalf("want ambiguous warning, got %v", a.Warnings)
		}
		if containsWarning(a.Warnings, warnBuiltinNamePipe) {
			t.Fatalf("unexpected builtin-pipe warning: %v", a.Warnings)
		}
	})

	t.Run("builtin_name_pipe_info", func(t *testing.T) {
		a, err := Analyze("data | unique")
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if !containsWarning(a.Warnings, warnBuiltinNamePipe) {
			t.Fatalf("want builtin-name pipe warning, got %v", a.Warnings)
		}
		if containsWarning(a.Warnings, warnAmbiguousBitwiseOr) {
			t.Fatalf("unexpected ambiguous warning: %v", a.Warnings)
		}
	})

	t.Run("array_lit_pipe_no_builtin_ident_warning", func(t *testing.T) {
		a, err := Analyze("[1, 2, 3] | sum")
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if containsWarning(a.Warnings, warnBuiltinNamePipe) {
			t.Fatalf("arrayLit|ident should not emit builtin-name warning: %v", a.Warnings)
		}
	})

	t.Run("flags_MASK_ambiguous", func(t *testing.T) {
		a, err := Analyze("flags | MASK")
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if !containsWarning(a.Warnings, warnAmbiguousBitwiseOr) {
			t.Fatalf("want ambiguous warning, got %v", a.Warnings)
		}
	})
}

func containsWarning(warnings []string, msg string) bool {
	for _, w := range warnings {
		if w == msg {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// B. Range .. precedence
// ---------------------------------------------------------------------------

func TestRangePrecedence_Parse(t *testing.T) {
	t.Run("add_then_range", func(t *testing.T) {
		ast, err := newParser("1+2..4").parse()
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		bin, ok := ast.(*binaryExpr)
		if !ok || bin.Op != "+" {
			t.Fatalf("want (1 + (2..4)), got %T %s", ast, ASTToString(ast))
		}
		if _, isInt := bin.Left.(*intLit); !isInt {
			t.Fatalf("left want intLit, got %T", bin.Left)
		}
		rng, isRange := bin.Right.(*rangeExpr)
		if !isRange {
			t.Fatalf("right want rangeExpr, got %T", bin.Right)
		}
		if rng.Start.(*intLit).Value != 2 || rng.End.(*intLit).Value != 4 {
			t.Fatalf("range = %s", ASTToString(rng))
		}
	})

	t.Run("range_then_add", func(t *testing.T) {
		ast, err := newParser("1..2+3").parse()
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		bin, ok := ast.(*binaryExpr)
		if !ok || bin.Op != "+" {
			t.Fatalf("want ((1..2) + 3), got %T %s", ast, ASTToString(ast))
		}
		rng, ok := bin.Left.(*rangeExpr)
		if !ok {
			t.Fatalf("left want rangeExpr, got %T", bin.Left)
		}
		if rng.Start.(*intLit).Value != 1 || rng.End.(*intLit).Value != 2 {
			t.Fatalf("range = %s", ASTToString(rng))
		}
		if bin.Right.(*intLit).Value != 3 {
			t.Fatalf("right = %s", ASTToString(bin.Right))
		}
	})

	t.Run("eq_then_range", func(t *testing.T) {
		ast, err := newParser("a==1..3").parse()
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		bin, ok := ast.(*binaryExpr)
		if !ok || bin.Op != "==" {
			t.Fatalf("want (a == (1..3)), got %T %s", ast, ASTToString(ast))
		}
		if _, ok := bin.Right.(*rangeExpr); !ok {
			t.Fatalf("right want rangeExpr, got %T", bin.Right)
		}
	})

	t.Run("chained_range_left_assoc", func(t *testing.T) {
		ast, err := newParser("1..2..3").parse()
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		outer, ok := ast.(*rangeExpr)
		if !ok {
			t.Fatalf("want rangeExpr, got %T", ast)
		}
		inner, ok := outer.Start.(*rangeExpr)
		if !ok {
			t.Fatalf("want left-assoc (1..2)..3, start=%T", outer.Start)
		}
		if inner.Start.(*intLit).Value != 1 || inner.End.(*intLit).Value != 2 {
			t.Fatalf("inner = %s", ASTToString(inner))
		}
		if outer.End.(*intLit).Value != 3 {
			t.Fatalf("outer end = %s", ASTToString(outer.End))
		}
	})

	t.Run("bang_range_grouped", func(t *testing.T) {
		ast, err := newParser("!(1..2)").parse()
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		u, ok := ast.(*unaryExpr)
		if !ok || u.Op != "!" {
			t.Fatalf("want !(1..2), got %T %s", ast, ASTToString(ast))
		}
		if _, ok := u.Operand.(*rangeExpr); !ok {
			t.Fatalf("operand want rangeExpr, got %T", u.Operand)
		}
	})

	t.Run("bang_then_range", func(t *testing.T) {
		ast, err := newParser("!1..2").parse()
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		rng, ok := ast.(*rangeExpr)
		if !ok {
			t.Fatalf("want (!1)..2, got %T %s", ast, ASTToString(ast))
		}
		u, ok := rng.Start.(*unaryExpr)
		if !ok || u.Op != "!" {
			t.Fatalf("start want unary !, got %T", rng.Start)
		}
		if rng.End.(*intLit).Value != 2 {
			t.Fatalf("end = %s", ASTToString(rng.End))
		}
	})
}

func TestRangePrecedence_Eval(t *testing.T) {
	t.Run("add_binds_looser_than_range", func(t *testing.T) {
		// 1+2..4 → 1+(2..4); int + array is an error.
		_, err := Eval(context.Background(), "1+2..4", nil)
		if err == nil {
			t.Fatal("expected error for 1+(2..4)")
		}
		got, err := Eval(context.Background(), "(1+2)..4", nil)
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		want := []any{int64(3), int64(4)}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("range_then_add", func(t *testing.T) {
		got, err := EvalInt(context.Background(), "len(1..2)+3")
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		if got != 5 {
			t.Fatalf("got %d, want 5", got)
		}
	})

	t.Run("eq_then_range", func(t *testing.T) {
		got, err := Eval(context.Background(), "a==1..3", map[string]any{
			"a": []any{int64(1), int64(2), int64(3)},
		})
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		if got != true {
			t.Fatalf("got %#v, want true", got)
		}
	})

	t.Run("chained_range", func(t *testing.T) {
		// (1..2)..3 → non-int start coerces via Int() to 0 → 0..3
		got, err := Eval(context.Background(), "1..2..3", nil)
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		want := []any{int64(0), int64(1), int64(2), int64(3)}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("bang_grouped_vs_bang_then_range", func(t *testing.T) {
		g1, err := Eval(context.Background(), "!(1..2)", nil)
		if err != nil {
			t.Fatalf("!(1..2): %v", err)
		}
		if g1 != false {
			t.Fatalf("!(1..2) = %#v, want false", g1)
		}
		g2, err := Eval(context.Background(), "!1..2", nil)
		if err != nil {
			t.Fatalf("!1..2: %v", err)
		}
		want := []any{int64(0), int64(1), int64(2)}
		if !reflect.DeepEqual(g2, want) {
			t.Fatalf("!1..2 = %#v, want %#v", g2, want)
		}
	})
}

// ---------------------------------------------------------------------------
// C. Destructure semicolon consistency with let
// ---------------------------------------------------------------------------

func TestDestructure_OptionalSemicolon(t *testing.T) {
	t.Run("object_no_semi_returns_value", func(t *testing.T) {
		got, err := Eval(context.Background(), `let {a} = m`, map[string]any{
			"m": map[string]any{"a": 42},
		})
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		m, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("got %T, want map", got)
		}
		switch v := m["a"].(type) {
		case int:
			if v != 42 {
				t.Fatalf("got %#v", m)
			}
		case int64:
			if v != 42 {
				t.Fatalf("got %#v", m)
			}
		default:
			t.Fatalf("got %#v", m)
		}
	})

	t.Run("object_with_semi_body", func(t *testing.T) {
		got, err := EvalInt(context.Background(), `let {a} = m; a+1`, map[string]any{
			"m": map[string]any{"a": 10},
		})
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		if got != 11 {
			t.Fatalf("got %d, want 11", got)
		}
	})

	t.Run("array_no_semi", func(t *testing.T) {
		got, err := Eval(context.Background(), `let [a, b] = xs`, map[string]any{
			"xs": []any{1, 2},
		})
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		arr, ok := got.([]any)
		if !ok || len(arr) != 2 {
			t.Fatalf("got %#v", got)
		}
	})

	t.Run("array_with_semi", func(t *testing.T) {
		got, err := EvalInt(context.Background(), `let [a, b] = xs; a+b`, map[string]any{
			"xs": []any{3, 4},
		})
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		if got != 7 {
			t.Fatalf("got %d, want 7", got)
		}
	})

	t.Run("parse_ast_body_equals_value", func(t *testing.T) {
		ast, err := newParser(`let {a} = m`).parse()
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		d, ok := ast.(*destructureExpr)
		if !ok {
			t.Fatalf("got %T", ast)
		}
		if d.Body != d.Value {
			t.Fatal("Body should be the same node as Value when ';' omitted")
		}
	})
}

func TestDestructure_StillRequiresEquals(t *testing.T) {
	_, err := newParser(`let {a} m`).parse()
	if err == nil {
		t.Fatal("expected parse error")
	}
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("got %T", err)
	}
	if !strings.Contains(pe.Message, "expected '='") {
		t.Fatalf("message = %q", pe.Message)
	}
}
