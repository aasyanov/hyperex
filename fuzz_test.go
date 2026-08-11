package hyperex

import (
	"context"
	"testing"
	"time"
)

func fuzzNoPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()
	fn()
}

// FuzzLexer_NoPanic ensures tokenization never panics on arbitrary input.
func FuzzLexer_NoPanic(f *testing.F) {
	f.Add("")
	f.Add("1 + 2 * 3")
	f.Add(`"hello"`)
	f.Add("x?.y ?? 42")
	f.Add("arr[1:10:2]")
	f.Add("/* comment */ 1")
	f.Add("let f = x => x + 1")
	f.Add("\x00\xff\xfe")
	f.Add("(((((((((((((((((((")
	f.Add(`"unterminated`)

	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > 1024 {
			t.Skip()
		}
		fuzzNoPanic(t, "lexer.Tokenize", func() {
			lexer := newLexer(src)
			_, _ = lexer.tokenize()
		})
	})
}

// FuzzParser_NoPanic ensures parsing never panics on arbitrary input.
func FuzzParser_NoPanic(f *testing.F) {
	f.Add("")
	f.Add("1 + 2 * 3")
	f.Add("x in [1,2,3]")
	f.Add("let f = x => x + 1; f(10)")
	f.Add("a?.b?.c ?? 0")
	f.Add("[1,2,3][0:2]")
	f.Add("if true then 1 else 2")
	f.Add("{a:1, b:2}")
	f.Add(")))((((")

	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > 1024 {
			t.Skip()
		}
		fuzzNoPanic(t, "parser.Parse", func() {
			p := newParser(src)
			_, _ = p.parse()
		})
	})
}

// FuzzEval_NoPanic_NoHang runs Eval under tight sandbox constraints to catch panics
// and unintended long-running paths.
func FuzzEval_NoPanic_NoHang(f *testing.F) {
	f.Add("1+2")
	f.Add("len([1,2,3])")
	f.Add(`"abc"[0:2]`)
	f.Add("2 in [1,2,3]")
	f.Add("1/0")
	f.Add("missing")
	f.Add("[1][99]")
	f.Add("range(1,100) | sum")
	f.Add("x?.y ?? 1")
	f.Add("")

	f.Fuzz(func(t *testing.T, expr string) {
		if len(expr) > 512 {
			t.Skip()
		}
		fuzzNoPanic(t, "Engine.Eval", func() {
			engine := New(
				WithSandbox(SandboxConfig{
					MaxOps:   20_000,
					Timeout:  2 * time.Millisecond,
					Secure:   true,
					ReadOnly: true,
				}),
			)
			_, _ = engine.Eval(context.Background(), expr, map[string]any{})
		})
	})
}

// FuzzCompileEval_Pipeline asserts parse→compile→eval never panics on arbitrary
// strings under a MaxOps sandbox (errors are fine; panics are not).
func FuzzCompileEval_Pipeline(f *testing.F) {
	f.Add("1+2")
	f.Add("x + y")
	f.Add("len(arr)")
	f.Add("true && false")
	f.Add("((((")
	f.Add("1/0")
	f.Add("[1,2,3][1]")
	f.Add(`"a" + "b"`)
	f.Add("unique([1,1,2])")
	f.Add("2 in arr")

	f.Fuzz(func(t *testing.T, expr string) {
		if len(expr) > 512 {
			t.Skip()
		}
		fuzzNoPanic(t, "parse→compile→eval", func() {
			engine := New(
				WithSandbox(SandboxConfig{
					MaxOps:        5_000,
					Timeout:       5 * time.Millisecond,
					MaxMemory:     1 << 20,
					MaxStackDepth: 64,
				}),
				WithOptimize(true),
			)
			prog, err := engine.Compile(expr)
			if err != nil {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			_, _ = prog.Eval(ctx, map[string]any{
				"x":   int64(1),
				"y":   int64(2),
				"arr": []any{int64(1), int64(2), int64(3)},
			})
		})
	})
}

func FuzzSimplify_NoPanic(f *testing.F) {
	f.Add("1+2+3")
	f.Add("x*2 + 0")
	f.Add(`"a" + "b"`)
	f.Add("")
	f.Add("((((")
	f.Add("x + x + x")

	f.Fuzz(func(t *testing.T, expr string) {
		if len(expr) > 512 {
			t.Skip()
		}
		fuzzNoPanic(t, "Simplify", func() {
			_, _ = Simplify(expr)
		})
	})
}

func FuzzAnalyze_NoPanic(f *testing.F) {
	f.Add("1+2")
	f.Add("x in [1,2,3]")
	f.Add("let x = 1; x+2")
	f.Add("")
	f.Add("a?.b")
	f.Add("f(1,2,3)")

	f.Fuzz(func(t *testing.T, expr string) {
		if len(expr) > 512 {
			t.Skip()
		}
		fuzzNoPanic(t, "Analyze", func() {
			_, _ = Analyze(expr)
		})
	})
}

func FuzzCompile_NoPanic(f *testing.F) {
	f.Add("1+2")
	f.Add("x?.y ?? 1")
	f.Add("arr[1:10:2]")
	f.Add("")
	f.Add("let a = 1; a")
	f.Add("(((((")

	f.Fuzz(func(t *testing.T, expr string) {
		if len(expr) > 512 {
			t.Skip()
		}
		fuzzNoPanic(t, "Compile", func() {
			engine := New(WithOptimize(true))
			_, _ = engine.Compile(expr)
		})
	})
}

func FuzzOptimize_Equivalent(f *testing.F) {
	f.Add("1+2*3")
	f.Add("x + y*2")
	f.Add("arr[1:3]")
	f.Add("2 in arr")
	f.Add("unique([1,2,2,3]) | len")
	f.Add("1+2+3+4")
	f.Add("x*1 + 0")
	// Regression seeds (optimizer vs runtime semantics)
	f.Add(`0+"hello"`)
	f.Add("1 && true")
	f.Add("~~!0")
	f.Add("~~true")
	f.Add("~~false")
	f.Add("s**1")
	f.Add("s**0")
	f.Add("~~.1")
	f.Add("~~1.5")

	f.Fuzz(func(t *testing.T, expr string) {
		if len(expr) > 512 {
			t.Skip()
		}
		fuzzNoPanic(t, "Optimize_Equivalent", func() {
			env := map[string]any{
				"x":   int64(10),
				"y":   int64(3),
				"arr": []any{int64(1), int64(2), int64(3), int64(4)},
				"m":   map[string]any{"a": int64(1), "b": int64(2)},
				"s":   "hello",
			}

			engOpt := New(WithOptimize(true), WithMaxOps(50_000), WithTimeout(10*time.Millisecond))
			engNo := New(WithOptimize(false), WithMaxOps(50_000), WithTimeout(10*time.Millisecond))

			pOpt, err1 := engOpt.Compile(expr)
			pNo, err2 := engNo.Compile(expr)
			// Compile may diverge when the optimizer folds away ill-formed subtrees;
			// only require result equivalence when both compile and both eval succeed.
			if err1 != nil || err2 != nil {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			r1, e1 := pOpt.Eval(ctx, env)
			r2, e2 := pNo.Eval(ctx, env)
			if e1 != nil || e2 != nil {
				// Algebraic folds (e.g. A*0 → 0) can mask UndefinedError; errors need not match.
				return
			}
			v1, v2 := NewValue(r1), NewValue(r2)
			// Closures/funcs are rebuilt per Compile — pointer identity will differ.
			if v1.IsFunc() || v2.IsFunc() {
				if v1.IsFunc() && v2.IsFunc() {
					return
				}
				t.Fatalf("func-ness mismatch: opt=%v no=%v", r1, r2)
			}
			if !v1.Equal(v2) {
				t.Fatalf("result mismatch: opt=%v no=%v", r1, r2)
			}
		})
	})
}
