package hyperex

import (
	"context"
	"testing"
)

func TestNotInHonorsPrecedence(t *testing.T) {
	for _, expr := range []string{`1 + 2 not in [3]`, `2 * 3 not in [6]`} {
		got, err := Eval(context.Background(), expr)
		if err != nil || got != false {
			t.Fatalf("%s: got (%#v, %v), want (false, nil)", expr, got, err)
		}
	}
}

func TestLetWithoutBodyEvaluatesValueOnce(t *testing.T) {
	calls := 0
	eng := New(WithFunction("bump", func() int { calls++; return calls }))
	got, err := eng.Eval(context.Background(), `let x = bump()`)
	if err != nil || got != int64(1) || calls != 1 {
		t.Fatalf("got=(%#v, %v), calls=%d; want value=1, calls=1", got, err, calls)
	}
}

func TestDestructureWithoutBodyEvaluatesValueOnce(t *testing.T) {
	calls := 0
	eng := New(WithFunction("bump", func() map[string]any {
		calls++
		return map[string]any{"a": calls}
	}))
	got, err := eng.Eval(context.Background(), `let {a} = bump()`)
	if err != nil || calls != 1 {
		t.Fatalf("got=(%#v, %v), calls=%d; want calls=1", got, err, calls)
	}
}
