package hyperex

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// TestConcurrency_SharedProgramEval hammers Program.Eval from many goroutines
// against one shared Engine/Program (must be -race clean).
func TestConcurrency_SharedProgramEval(t *testing.T) {
	eng := New()
	prog, err := eng.Compile(`x + y * 2`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	const goroutines = 32
	const iters = 200
	var fails atomic.Int64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			env := map[string]any{
				"x": int64(id),
				"y": int64(3),
			}
			want := int64(id) + int64(3)*2
			for i := 0; i < iters; i++ {
				got, err := prog.Eval(context.Background(), env)
				if err != nil {
					fails.Add(1)
					t.Errorf("Eval: %v", err)
					return
				}
				if NewValue(got).Int() != want {
					fails.Add(1)
					t.Errorf("got %v want %d", got, want)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	if fails.Load() != 0 {
		t.Fatalf("%d goroutine failures", fails.Load())
	}
}

// TestConcurrency_SharedEngineCompile runs concurrent Compile on one Engine
// (config immutable after New; must be -race clean).
func TestConcurrency_SharedEngineCompile(t *testing.T) {
	eng := New(WithOptimize(true))

	exprs := []string{
		`1 + 2`,
		`x * y`,
		`len([1,2,3])`,
		`"a" + "b"`,
		`x > 0 && y < 10`,
		`arr[0]`,
		`unique([1,2,2,3])`,
		`2 in [1,2,3]`,
	}

	const goroutines = 16
	const iters = 100
	var fails atomic.Int64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				expr := exprs[(id+i)%len(exprs)]
				p, err := eng.Compile(expr)
				if err != nil {
					fails.Add(1)
					t.Errorf("Compile(%q): %v", expr, err)
					return
				}
				if p == nil || p.Source() != expr {
					fails.Add(1)
					t.Errorf("Compile(%q): bad program source %q", expr, p.Source())
					return
				}
			}
		}(g)
	}
	wg.Wait()
	if fails.Load() != 0 {
		t.Fatalf("%d goroutine failures", fails.Load())
	}
}

// TestConcurrency_MixedCompileAndEval mixes Compile and Eval on a shared Engine.
func TestConcurrency_MixedCompileAndEval(t *testing.T) {
	type Env struct {
		N int64 `expr:"n"`
	}
	eng := New(WithTypeCheck(Env{}))
	prog, err := eng.Compile(`n + 1`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	const goroutines = 24
	var fails atomic.Int64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 80; i++ {
				if id%2 == 0 {
					expr := fmt.Sprintf("%d + %d", id, i)
					p, err := eng.Compile(expr)
					if err != nil {
						fails.Add(1)
						t.Errorf("Compile: %v", err)
						return
					}
					got, err := p.Eval(context.Background(), nil)
					if err != nil {
						fails.Add(1)
						t.Errorf("Eval compiled: %v", err)
						return
					}
					if NewValue(got).Int() != int64(id+i) {
						fails.Add(1)
						t.Errorf("got %v want %d", got, id+i)
						return
					}
				} else {
					got, err := prog.Eval(context.Background(), map[string]any{"n": int64(id)})
					if err != nil {
						fails.Add(1)
						t.Errorf("Eval shared: %v", err)
						return
					}
					if NewValue(got).Int() != int64(id+1) {
						fails.Add(1)
						t.Errorf("got %v want %d", got, id+1)
						return
					}
				}
			}
		}(g)
	}
	wg.Wait()
	if fails.Load() != 0 {
		t.Fatalf("%d goroutine failures", fails.Load())
	}
}
