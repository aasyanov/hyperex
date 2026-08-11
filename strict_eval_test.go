package hyperex

import (
	"context"
	"errors"
	"testing"
)

func TestStrictEval_TypedHelpers(t *testing.T) {
	strict := New(WithStrictMode(true))
	loose := New(WithStrictMode(false))
	ctx := context.Background()

	_, err := strict.EvalInt(ctx, `"not-a-number"`)
	if !errors.Is(err, ErrInvalidConversion) {
		t.Fatalf("strict EvalInt: err=%v", err)
	}
	n, err := loose.EvalInt(ctx, `"not-a-number"`)
	if err != nil {
		t.Fatalf("loose EvalInt err: %v", err)
	}
	if n != 0 {
		t.Fatalf("loose EvalInt got %d", n)
	}

	_, err = strict.EvalBool(ctx, `42`)
	if !errors.Is(err, ErrInvalidConversion) {
		t.Fatalf("strict EvalBool: err=%v", err)
	}
	ok, err := loose.EvalBool(ctx, `42`)
	if err != nil {
		t.Fatalf("loose EvalBool: %v", err)
	}
	if !ok {
		t.Fatal("loose EvalBool want true")
	}

	_, err = strict.EvalString(ctx, `1 + 2`)
	if !errors.Is(err, ErrInvalidConversion) {
		t.Fatalf("strict EvalString: err=%v", err)
	}
	s, err := loose.EvalString(ctx, `1 + 2`)
	if err != nil {
		t.Fatalf("loose EvalString: %v", err)
	}
	if s == "" {
		t.Fatal("loose EvalString empty")
	}

	_, err = strict.EvalFloat(ctx, `"abc"`)
	if !errors.Is(err, ErrInvalidConversion) {
		t.Fatalf("strict EvalFloat: err=%v", err)
	}
}

func TestStrictEval_ProgramTypedHelpers(t *testing.T) {
	strict := New(WithStrictMode(true))
	prog := strict.MustCompile(`"hello"`)
	_, err := prog.EvalInt(context.Background(), nil)
	if !errors.Is(err, ErrInvalidConversion) {
		t.Fatalf("Program.EvalInt strict: %v", err)
	}
}
