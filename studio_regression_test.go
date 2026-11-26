package hyperex

import (
	"context"
	"math"
	"strings"
	"testing"
)

func TestInlineRespectsLambdaAndDestructureShadowing(t *testing.T) {
	for _, tc := range []struct {
		expr string
		want any
	}{
		{`let x = 5; map([1], x => x)`, []any{int64(1)}},
		{`let a = 1; let [a] = [2]; a`, int64(2)},
	} {
		inlined, err := Inline(tc.expr)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Eval(context.Background(), inlined)
		if err != nil {
			t.Fatalf("%s -> %s: %v", tc.expr, inlined, err)
		}
		if !NewValue(got).Equal(NewValue(tc.want)) {
			t.Fatalf("%s -> %s: got %#v, want %#v", tc.expr, inlined, got, tc.want)
		}
	}
}

func TestInlineDoesNotDuplicateEffectfulBindings(t *testing.T) {
	calls := 0
	eng := New(WithFunction("bump", func() int { calls++; return 2 }))
	text, err := Inline(`let x = bump(); x + x`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := eng.Eval(context.Background(), text)
	if err != nil || got != int64(4) || calls != 1 {
		t.Fatalf("Inline produced %q -> (%#v, %v), calls=%d", text, got, err, calls)
	}
}

func TestASTToStringPreservesComputedKeysAndPostfixOperands(t *testing.T) {
	node, err := Parse(`{[k]: 1}`)
	if err != nil {
		t.Fatal(err)
	}
	text := ASTToString(node)
	got, err := Eval(context.Background(), text, map[string]any{"k": "actual"})
	if err != nil || NewValue(got).Map()["actual"] != int64(1) {
		t.Fatalf("%q -> (%#v, %v)", text, got, err)
	}

	member, err := Parse(`(a + b).c`)
	if err != nil {
		t.Fatal(err)
	}
	if text := ASTToString(member); !strings.Contains(text, "(a + b).c") {
		t.Fatalf("member parentheses lost: %q", text)
	}
}

func TestASTToStringPreservesLambdaCallsAndUnaryExpressions(t *testing.T) {
	for _, tc := range []struct {
		expr string
		env  map[string]any
		want any
	}{
		{`(x => x)(1)`, nil, int64(1)},
		{`- -x`, map[string]any{"x": int64(2)}, int64(2)},
		{`(1 + 2)..4`, nil, []any{int64(3), int64(4)}},
	} {
		node, err := Parse(tc.expr)
		if err != nil {
			t.Fatal(err)
		}
		text := ASTToString(node)
		got, err := Eval(context.Background(), text, tc.env)
		if err != nil || !NewValue(got).Equal(NewValue(tc.want)) {
			t.Fatalf("%q -> %q -> (%#v, %v), want %#v", tc.expr, text, got, err, tc.want)
		}
	}
}

func TestASTToStringPreservesNumericLiteralTypesAndExtremes(t *testing.T) {
	node, err := Parse(`1.0`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Eval(context.Background(), ASTToString(node))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.(float64); !ok {
		t.Fatalf("float literal became %#v (%T)", got, got)
	}

	text, err := Simplify(`-9223372036854775807 - 1`)
	if err != nil {
		t.Fatal(err)
	}
	got, err = Eval(context.Background(), text)
	if err != nil || got != int64(math.MinInt64) {
		t.Fatalf("%q -> (%#v, %v), want MinInt64", text, got, err)
	}
}

func TestASTToStringKeepsImplicitLetSingleEvaluation(t *testing.T) {
	node, err := Parse(`let x = bump()`)
	if err != nil {
		t.Fatal(err)
	}
	text := ASTToString(node)
	calls := 0
	eng := New(WithFunction("bump", func() int { calls++; return calls }))
	got, err := eng.Eval(context.Background(), text)
	if err != nil || got != int64(1) || calls != 1 {
		t.Fatalf("%q -> (%#v, %v), calls=%d", text, got, err, calls)
	}
}

func TestExpandPreservesAllNodeShapes(t *testing.T) {
	expr := `let x = A; [x > 0 ? x : 0, x ?? 0, abs(x), {a: x}.a, {a: x}["a"], [x, x + 1][0], [x, x + 1][0:2:1], [x, x + 1], {a: x}, (x..(x + 1)), (x => x)(x), [...[x]], let [y] = [x]; y, let {z} = {z: x}; z, [x] | sum]`
	text, err := Expand(expr, map[string]string{"A": "10"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(text); err != nil {
		t.Fatalf("expanded expression is not parseable: %v\n%s", err, text)
	}
}

func TestExpandRespectsLexicalBindingsAndProperties(t *testing.T) {
	text, err := Expand(`map([1], A => A)`, map[string]string{"A": "5"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Eval(context.Background(), text)
	if err != nil || !NewValue(got).Equal(NewValue([]any{int64(1)})) {
		t.Fatalf("%q -> (%#v, %v), want [1]", text, got, err)
	}

	text, err = Expand(`A.c`, map[string]string{"A": "a + b", "c": "replacement"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "(a + b).c") || strings.Contains(text, "replacement") {
		t.Fatalf("unexpected expansion: %q", text)
	}

	text, err = Expand(`let A = 1; A`, map[string]string{"A": "5"})
	if err != nil {
		t.Fatal(err)
	}
	got, err = Eval(context.Background(), text)
	if err != nil || got != int64(1) {
		t.Fatalf("%q -> (%#v, %v), want 1", text, got, err)
	}
}
