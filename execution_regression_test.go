package hyperex

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestChunk_ExtremeSizeDoesNotOverflow(t *testing.T) {
	got, err := Eval(context.Background(), `chunk([1,2], 9223372036854775807)`)
	if err != nil {
		t.Fatal(err)
	}
	want := []any{[]any{int64(1), int64(2)}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestHigherOrderLambdaOpsAreCumulative(t *testing.T) {
	body := "x" + strings.Repeat(" + 0", 600)
	eng := New(WithMaxOps(2000))
	_, err := eng.Eval(context.Background(), `map([1,2], x => `+body+`)`)
	if !errors.Is(err, ErrOpsLimitExceeded) {
		t.Fatalf("got %v, want ErrOpsLimitExceeded", err)
	}
}

func TestHigherOrderLambdaHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	marks := 0
	eng := New(
		WithFunction("cancelNow", func() bool { cancel(); return true }),
		WithFunction("mark", func(x int) int { marks++; return x }),
	)
	_, err := eng.Eval(ctx, `map([1], x => let c = cancelNow(); mark(x))`)
	if !errors.Is(err, ErrOperationCancelled) {
		t.Fatalf("got %v, want ErrOperationCancelled", err)
	}
	if marks != 0 {
		t.Fatalf("callback ran %d times after cancellation", marks)
	}
}

func TestProgramProfilePreservesSecureSandbox(t *testing.T) {
	type env struct{ Name string }
	eng := New(WithSecureSandbox())
	prog, err := eng.Compile(`x.Name`)
	if err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"x": env{Name: "secret"}}
	if _, evalErr := prog.Eval(context.Background(), input); evalErr == nil {
		t.Fatal("Eval unexpectedly allowed an external struct")
	}
	_, _, err = prog.Profile(context.Background(), input)
	if err == nil {
		t.Fatal("Profile bypassed the Secure environment policy")
	}
}
