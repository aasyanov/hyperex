package hyperex

import (
	"context"
	"testing"
)

func TestEngine_Stats_Increments(t *testing.T) {
	eng := New()
	if got := eng.Stats(); got != (Stats{}) {
		t.Fatalf("fresh Stats = %+v", got)
	}

	if _, err := eng.Compile(`1 + 1`); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if s := eng.Stats(); s.Compiles != 1 {
		t.Fatalf("Compiles=%d want 1", s.Compiles)
	}

	if _, err := eng.Compile(`{{{`); err == nil {
		t.Fatal("expected compile error")
	}
	if s := eng.Stats(); s.Compiles != 2 {
		t.Fatalf("Compiles=%d want 2 after failed compile", s.Compiles)
	}

	got, err := eng.Eval(context.Background(), `2 + 2`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if got != int64(4) {
		t.Fatalf("got %v", got)
	}
	s := eng.Stats()
	if s.Evals < 1 {
		t.Fatalf("Evals=%d want >=1", s.Evals)
	}
	if s.EvalErrors != 0 {
		t.Fatalf("EvalErrors=%d want 0", s.EvalErrors)
	}

	_, err = eng.Eval(context.Background(), `missing_var`)
	if err == nil {
		t.Fatal("expected eval error")
	}
	s = eng.Stats()
	if s.EvalErrors < 1 {
		t.Fatalf("EvalErrors=%d want >=1", s.EvalErrors)
	}
}

func TestEngine_Stats_ConcurrentSafe(t *testing.T) {
	eng := New()
	prog := eng.MustCompile(`x + 1`)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = prog.Eval(context.Background(), map[string]any{"x": i})
		}
		close(done)
	}()
	for i := 0; i < 50; i++ {
		_ = eng.Stats()
	}
	<-done
	if eng.Stats().Evals < 100 {
		t.Fatalf("Evals=%d", eng.Stats().Evals)
	}
}
