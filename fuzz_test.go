package hyperex

import (
	"context"
	"errors"
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
	f.Add("timeHour(timeUnix(0))")
	f.Add("durationSeconds(1)+durationMs(500)")
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

// FuzzCompileEval_Pipeline asserts parse->compile->eval never panics on arbitrary
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
		fuzzNoPanic(t, "parse->compile->eval", func() {
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
			engAlg := New(WithConstantFolding(false), WithMaxOps(50_000), WithTimeout(10*time.Millisecond))

			pOpt, err1 := engOpt.Compile(expr)
			pNo, err2 := engNo.Compile(expr)
			pAlg, err3 := engAlg.Compile(expr)
			if err1 != nil || err2 != nil || err3 != nil {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			results := []struct {
				name  string
				value any
				err   error
			}{
				{name: "optimized"},
				{name: "unoptimized"},
				{name: "algebraic-only"},
			}
			results[0].value, results[0].err = pOpt.Eval(ctx, env)
			results[1].value, results[1].err = pNo.Eval(ctx, env)
			results[2].value, results[2].err = pAlg.Eval(ctx, env)
			for i := 1; i < len(results); i++ {
				if fuzzErrorDivergence(results[0].err, results[i].err) {
					t.Fatalf("error mismatch: %s=%v %s=%v", results[0].name, results[0].err, results[i].name, results[i].err)
				}
			}
			if results[0].err != nil || results[1].err != nil || results[2].err != nil {
				return
			}
			for i := 1; i < len(results); i++ {
				assertFuzzEquivalent(t, results[0].value, results[i].value)
			}
		})
	})
}

func fuzzErrorDivergence(a, b error) bool {
	if a == nil || b == nil {
		if fuzzResourceError(a) || fuzzResourceError(b) {
			return false
		}
		return a != b
	}
	aResource := fuzzResourceError(a)
	bResource := fuzzResourceError(b)
	if aResource || bResource {
		return false
	}
	if errors.Is(a, b) || errors.Is(b, a) {
		return false
	}
	var aUndefined, bUndefined *UndefinedError
	if errors.As(a, &aUndefined) && errors.As(b, &bUndefined) {
		return aUndefined.Name != bUndefined.Name || aUndefined.Kind != bUndefined.Kind
	}
	aRoot, bRoot := fuzzRootError(a), fuzzRootError(b)
	if aRoot == bRoot {
		return false
	}
	var aRuntime, bRuntime *RuntimeError
	if errors.As(a, &aRuntime) && errors.As(b, &bRuntime) {
		return aRuntime.Message != bRuntime.Message ||
			aRuntime.Op != bRuntime.Op ||
			aRuntime.Left != bRuntime.Left ||
			aRuntime.Right != bRuntime.Right
	}
	return true
}

func fuzzRootError(err error) error {
	for {
		next := errors.Unwrap(err)
		if next == nil {
			return err
		}
		err = next
	}
}

func fuzzResourceError(err error) bool {
	return errors.Is(err, ErrTimeout) || errors.Is(err, ErrOpsLimitExceeded) || errors.Is(err, ErrOperationCancelled)
}

func assertFuzzEquivalent(t *testing.T, a, b any) {
	t.Helper()
	va, vb := NewValue(a), NewValue(b)
	if va.IsFunc() || vb.IsFunc() {
		if va.IsFunc() && vb.IsFunc() {
			return
		}
		t.Fatalf("func-ness mismatch: %v != %v", a, b)
	}
	if va.Type() != vb.Type() {
		t.Fatalf("type mismatch: %v (%s) != %v (%s)", a, va.Type(), b, vb.Type())
	}
	if !va.Equal(vb) {
		t.Fatalf("result mismatch: %v != %v", a, b)
	}
}

func TestFuzzErrorDivergenceSemanticTypes(t *testing.T) {
	for _, tc := range []struct {
		name string
		a    error
		b    error
		want bool
	}{
		{name: "different sentinels", a: ErrDivisionByZero, b: ErrIndexOutOfBounds, want: true},
		{name: "same sentinel", a: ErrDivisionByZero, b: ErrDivisionByZero, want: false},
		{name: "same wrapped sentinel", a: wrapRuntime(ErrDivisionByZero), b: wrapRuntime(ErrDivisionByZero), want: false},
		{name: "different wrapped sentinels", a: wrapRuntime(ErrDivisionByZero), b: wrapRuntime(ErrIndexOutOfBounds), want: true},
		{name: "same runtime semantics", a: &RuntimeError{Message: "same"}, b: &RuntimeError{Message: "same"}, want: false},
		{name: "different runtime semantics", a: &RuntimeError{Message: "one"}, b: &RuntimeError{Message: "two"}, want: true},
		{name: "semantic versus value", a: ErrDivisionByZero, b: nil, want: true},
		{name: "resource versus value", a: ErrTimeout, b: nil, want: false},
		{name: "resource versus semantic", a: ErrTimeout, b: ErrDivisionByZero, want: false},
		{name: "two resource errors", a: ErrTimeout, b: ErrOperationCancelled, want: false},
		{name: "same undefined", a: &UndefinedError{Name: "x", Kind: "variable"}, b: &UndefinedError{Name: "x", Kind: "variable"}, want: false},
		{name: "different undefined", a: &UndefinedError{Name: "x", Kind: "variable"}, b: &UndefinedError{Name: "y", Kind: "variable"}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := fuzzErrorDivergence(tc.a, tc.b); got != tc.want {
				t.Fatalf("got %t, want %t", got, tc.want)
			}
		})
	}
}
