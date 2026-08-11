package hyperex

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestProgram_Eval_ContextCancel(t *testing.T) {
	prog, err := Compile(`map(range(0, 1000000), # + 1)`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = prog.Eval(ctx, nil)
	if !errors.Is(err, ErrOperationCancelled) {
		t.Fatalf("errors.Is(err, ErrOperationCancelled) = false; err=%v", err)
	}
}

func TestProgram_Eval_ContextDeadline(t *testing.T) {
	eng := New(WithMaxOps(1_000_000_000))
	prog, err := eng.Compile(`map(range(0, 100000), # + 1)`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // ensure deadline already expired

	_, err = prog.Eval(ctx, nil)
	if !errors.Is(err, ErrOperationCancelled) {
		t.Fatalf("errors.Is(err, ErrOperationCancelled) = false; err=%v", err)
	}
}

func TestEngine_Eval_ConcurrentCancel(t *testing.T) {
	// Use an explicit lambda (not `#`) and a fresh Program per goroutine so
	// cancel races cannot surface as spurious undefined-variable errors.
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			eng := New(WithMaxOps(1_000_000_000))
			prog, err := eng.Compile(`map(range(0, 200000), x => x + 1)`)
			if err != nil {
				t.Errorf("Compile: %v", err)
				return
			}
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(time.Millisecond)
				cancel()
			}()
			_, err = prog.Eval(ctx, nil)
			if err != nil && !errors.Is(err, ErrOperationCancelled) {
				t.Errorf("unexpected err: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestProgram_Eval_StructEnv(t *testing.T) {
	type Env struct {
		X int `expr:"x"`
		Y int `expr:"y"`
	}
	prog, err := Compile(`x + y`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	got, err := prog.Eval(context.Background(), Env{X: 2, Y: 3})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if got != int64(5) {
		t.Fatalf("got %v want 5", got)
	}
}
