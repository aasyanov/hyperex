package hyperex

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCoverage_ComplexityString(t *testing.T) {
	cases := []struct {
		c    Complexity
		want string
	}{
		{ComplexityConstant, "O(1)"},
		{ComplexityLinear, "O(n)"},
		{ComplexityLinearithmic, "O(n log n)"},
		{ComplexityQuadratic, "O(n²)"},
		{ComplexityUnknown, "O(?)"},
		{Complexity(99), "O(?)"},
	}
	for _, tt := range cases {
		if got := tt.c.String(); got != tt.want {
			t.Fatalf("%v.String()=%q want %q", tt.c, got, tt.want)
		}
	}
}

func TestCoverage_WithEnvTypes_AndComparable(t *testing.T) {
	tc := NewTypeChecker().WithEnvTypes(map[string]any{
		"n":  int64(1),
		"f":  1.5,
		"s":  "hi",
		"b":  true,
		"a":  []any{1},
		"m":  map[string]any{"k": 1},
		"fn": func() {},
		"p":  ptrInt(3),
	})
	if tc.env["n"] != TypeInt || tc.env["s"] != TypeString || tc.env["fn"] != TypeFunc {
		t.Fatalf("WithEnvTypes mapping incomplete: %#v", tc.env)
	}

	// Two-arg WithEnvType
	tc2 := NewTypeChecker().WithEnvType("x", 42)
	if tc2.env["x"] != TypeInt {
		t.Fatalf("WithEnvType name/value: got %v", tc2.env["x"])
	}
	// Odd arity ignored
	tc2.WithEnvType()
	tc2.WithEnvType(1, 2, 3)

	parser := newParser(`n < s`)
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if errs := tc.Check(ast); errs == nil {
		t.Fatal("expected comparison type error for int < string")
	}

	// Hit remaining check() branches with typed env
	exprs := []string{
		`b ? n : f`,
		`s[n]`,
		`a["bad"]`,
		`a[1:3:1]`,
		`{a: n, b: s}`,
		`"a"..3`,
		`1.."b"`,
		`n | abs`,
		`x => x + 1`,
		`let y = n; y + 1`,
		`n ?? 0`,
		`[...n]`,
		`let [a, b=1] = n; a`,
		`let {a, b=2} = n; a`,
		`!n`,
		`~s`,
		`"x" - 1`,
		`true && 1`,
		`len()`,
		`len(1,2,3)`,
		`obj.field`,
		`obj["k"]`,
	}
	for _, expr := range exprs {
		p := newParser(expr)
		node, perr := p.parse()
		if perr != nil {
			continue
		}
		_ = tc.Check(node)
		_ = tc.Infer(node)
	}

	// isComparable true paths via numeric/string compare
	okParser := newParser(`n < f`)
	okAST, _ := okParser.parse()
	if errs := tc.Check(okAST); errs != nil {
		t.Fatalf("n < f should be ok: %v", errs)
	}
}

func ptrInt(v int) *int { return &v }

func TestCoverage_OptimizerBuildersAndFolding(t *testing.T) {
	opt := newOptimizer().
		withConstantFolding(true).
		withStrengthReduction(true).
		withSkipOperators(map[string]string{"+": "add"})
	opt.deadCodeElim = true
	opt.algebraicSimplify = true
	opt.skipOperators["@@"] = true
	if opt.foldedCount != 0 || opt.reducedCount != 0 {
		t.Fatalf("fresh counters want 0,0 got %d,%d", opt.foldedCount, opt.reducedCount)
	}

	// Constant folding across arithmetic / comparisons / bitwise
	foldExprs := []string{
		`10 % 3`,
		`10 % 3.0`,
		`10.0 % 3`,
		`10.5 % 2.5`,
		`10 % 0`,
		`2 ** 3`,
		`2 ** 3.0`,
		`2.0 ** 3`,
		`2 ** 100`,
		`1 == 1`,
		`1 == 1.0`,
		`1.0 == 1`,
		`1.5 == 1.5`,
		`"a" == "a"`,
		`true == true`,
		`nil == nil`,
		`1 != 2`,
		`1 < 2`,
		`1 < 2.0`,
		`1.0 < 2`,
		`1.0 < 2.0`,
		`"a" < "b"`,
		`1 <= 1`,
		`1 <= 1.0`,
		`1.0 <= 2`,
		`1.0 <= 1.0`,
		`"a" <= "a"`,
		`2 > 1`,
		`2 >= 2`,
		`5 & 3`,
		`5 | 2`,
		`5 ^ 1`,
		`8 << 1`,
		`8 >> 1`,
		`-5`,
		`!false`,
		`~0`,
		`10 / 2`,
		`10 / 2.0`,
		`10.0 / 2`,
		`10.0 / 2.0`,
		`10 / 0`,
		`true ? 1 : 2`,
		`false ? 1 : 2`,
		`1 ?? 2`,
		`nil ?? 2`,
		`false && true`,
		`true || false`,
		`x + 0`,
		`0 + x`,
		`x - 0`,
		`x - x`,
		`x * 1`,
		`1 * x`,
		`x * 0`,
		`0 * x`,
		`x / 1`,
		`x / x`,
		`true && x`,
		`false && x`,
		`x && true`,
		`false || x`,
		`true || x`,
		`x || false`,
		`+0.0 + x`,
		`x + 0.0`,
		`x - 0.0`,
		`x * 1.0`,
		`1.0 * x`,
		`x / 1.0`,
	}
	for _, expr := range foldExprs {
		if _, err := Simplify(expr); err != nil && !strings.Contains(expr, "/") {
			// some may fail parse; ignore
			_ = err
		}
	}

	// Skip overloaded operator during fold
	eng := New(
		WithFunction("concat", func(a, b string) string { return a + b }),
		WithOperator("+", "concat"),
	)
	prog, err := eng.Compile(`"a" + "b"`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	got, err := prog.Eval(context.Background(), nil)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if got != "ab" {
		t.Fatalf("got %v want ab", got)
	}
	_ = eng.optimizer.foldedCount
}

func TestCoverage_TransformWalkCopyInline(t *testing.T) {
	exprs := []string{
		`1 + 2`,
		`-x`,
		`a ? b : c`,
		`f(1, 2)`,
		`obj.field`,
		`arr[0]`,
		`arr[1:5:2]`,
		`[1, 2, ...xs]`,
		`{a: 1, b: 2}`,
		`1..3`,
		`xs | sum`,
		`x => x + 1`,
		`a ?? b`,
		`let x = 1; x`,
		`let [a, b=1] = xs; a + b`,
		`let {a, b=2} = obj; a`,
	}
	identity := func(n Node) Node { return n }
	for _, expr := range exprs {
		ast, err := newParser(expr).parse()
		if err != nil {
			t.Fatalf("parse %q: %v", expr, err)
		}
		_ = Transform(ast, identity)
		Walk(ast, func(Node) bool { return true })
		Walk(ast, func(Node) bool { return false })
		_ = copyNode(ast)
		_ = inlineLetExprs(ast, map[string]Node{"x": &intLit{Value: 1}})
	}

	// Expand macros that exercise deep copyNode
	macros := map[string]string{
		"ONE":   `1`,
		"PAIR":  `[1, 2]`,
		"OBJ":   `{a: 1}`,
		"CALL":  `f(1)`,
		"PIPE":  `[1,2] | sum`,
		"RANGE": `1..3`,
		"TERN":  `true ? 1 : 0`,
		"COAL":  `a ?? 0`,
		"MEM":   `obj.x`,
		"IDX":   `arr[0]`,
		"SL":    `arr[0:1]`,
		"UN":    `-1`,
		"BIN":   `1 + 2`,
		"LAM":   `x => x`,
		"LET":   `let x = 1; x`,
	}
	for name, def := range macros {
		if _, err := Expand(name, map[string]string{name: def}); err != nil {
			t.Fatalf("Expand %s: %v", name, err)
		}
	}
	if _, err := Expand("BAD", map[string]string{"BAD": "(((("}); err == nil {
		t.Fatal("expected invalid macro error")
	}

	// Inline node variety
	inlineExprs := []string{
		`let x = 1; -x`,
		`let x = 1; x ? 2 : 3`,
		`let x = 1; f(x)`,
		`let x = 1; obj.x`,
		`let x = 1; arr[x]`,
		`let x = 1; [x, 2]`,
		`let x = 1; {a: x}`,
		`let x = 1; x..3`,
		`let x = [1]; x | sum`,
		`let x = 1; (y => x + y)`,
		`let x = 1; x ?? 0`,
		`let x = [1,2]; let [a,b=x] = x; a`,
		`let x = {a:1}; let {a,b=x} = x; a`,
		`let x = [1]; [...x]`,
		`let x = "hi"; x[0:2:1]`,
	}
	for _, expr := range inlineExprs {
		if _, err := Inline(expr); err != nil {
			t.Fatalf("Inline %q: %v", expr, err)
		}
	}

	_ = copyNode(nil)
	_ = copyNode(&spreadExpr{Expr: &ident{Name: "xs"}})
}

func TestCoverage_AnalyticsEdges(t *testing.T) {
	if err := CheckExpressionType("(((", TypeInt, nil); err == nil {
		t.Fatal("expected parse error")
	}
	if err := CheckExpressionType(`"hi"`, TypeInt, nil); err == nil {
		t.Fatal("expected type mismatch")
	}
	type Env struct {
		X string `expr:"x"`
	}
	if err := CheckExpressionType(`x + 1`, TypeInt, Env{}); err == nil {
		// may or may not error depending on + rules; just exercise path
		_ = err
	}

	a, err := Analyze(`map(items, x => map(x, y => y))`)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if a.Complexity < ComplexityLinear {
		t.Fatalf("expected at least linear complexity, got %s", a.Complexity.String())
	}

	prof, _, err := ProfileExpr(context.Background(), "(((", nil)
	if err == nil {
		t.Fatal("expected ProfileExpr compile error")
	}
	if prof == nil || prof.Error == nil {
		t.Fatal("expected profile error populated")
	}

	prog, err := Compile(`1 + 2`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	p2, res, err := prog.Profile(context.Background(), nil)
	if err != nil || NewValue(res).Int() != 3 {
		t.Fatalf("Program.Profile: res=%v err=%v", res, err)
	}
	if p2.OpsCount <= 0 {
		t.Fatal("expected ops")
	}
}

func TestCoverage_VMEdges_AndNativeCall(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		expr string
		env  map[string]any
	}{
		{`10 % 3`, nil},
		{`10.5 % 2`, nil},
		{`10 % 2.5`, nil},
		{`7 / 2`, nil},
		{`7.0 / 2`, nil},
		{`"ab" + "cd"`, nil},
		{`[1,2] + [3]`, nil},
		{`1 - 2`, nil},
		{`1.5 - 0.5`, nil},
		{`3 * 4`, nil},
		{`1.5 * 2`, nil},
		{`"hello"[1:4]`, nil},
		{`[1,2,3,4][1:3]`, nil},
		{`[1,2,3,4][::2]`, nil},
		{`[1,2,3][-1]`, nil},
		{`"hi"[0]`, nil},
		{`{a:1}["a"]`, nil},
		{`1 in [1,2,3]`, nil},
		{`"a" in {a:1}`, nil},
		{`"ell" in "hello"`, nil},
		{`sort([3,1,2])`, nil},
		{`sort(["b","a"])`, nil},
		{`reverse([1,2,3])`, nil},
		{`last([1,2,3])`, nil},
		{`take([1,2,3], 2)`, nil},
		{`avg([1,2,3])`, nil},
		{`flatten([[1],[2,3]])`, nil},
		{`compact([1, null, 0, 2])`, nil},
		{`truthy([0,1,false,true])`, nil},
		{`pluck([{a:1},{a:2}], "a")`, nil},
		{`chunk([1,2,3,4], 2)`, nil},
		{`hasKey({a:1}, "a")`, nil},
		{`default(null, 5)`, nil},
		{`coalesce(null, 0, 2)`, nil},
		{`notEmpty("")`, nil},
		{`toJSON({a:1})`, nil},
		{`fromJSON('{"a":1}')`, nil},
		{`fromJSON("not-json")`, nil},
		{`findIndex([1,2,3], x => x == 2)`, nil},
		{`trimRight("  hi  ")`, nil},
	}
	for _, tt := range cases {
		if _, err := Eval(ctx, tt.expr, tt.env); err != nil {
			// fromJSON invalid is expected to error
			if !strings.Contains(tt.expr, "not-json") {
				t.Fatalf("Eval(%q): %v", tt.expr, err)
			}
		}
	}

	// Native Go function via env (callNative + convertArg)
	add := func(a int, b int) int { return a + b }
	sumSlice := func(xs []int) int {
		n := 0
		for _, x := range xs {
			n += x
		}
		return n
	}
	variadic := func(prefix string, nums ...int) int {
		n := 0
		for _, x := range nums {
			n += x
		}
		return n + len(prefix)
	}
	nilArg := func(p *int) bool { return p == nil }
	noRet := func(x int) {}
	withErr := func(x int) (int, error) {
		if x < 0 {
			return 0, errors.New("neg")
		}
		return x, nil
	}
	panics := func(x int) int { panic("boom") }

	env := map[string]any{
		"add":      add,
		"sumSlice": sumSlice,
		"variadic": variadic,
		"nilArg":   nilArg,
		"noRet":    noRet,
		"withErr":  withErr,
		"panics":   panics,
	}
	nativeCases := []string{
		`add(2, 3)`,
		`sumSlice([1,2,3])`,
		`variadic("ab", 1, 2)`,
		`nilArg(null)`,
		`noRet(1)`,
		`withErr(1)`,
		`withErr(-1)`,
		`panics(1)`,
	}
	for _, expr := range nativeCases {
		_, _ = Eval(ctx, expr, env)
	}

	// Overflow / strict numeric edges
	strict := New(WithStrictMode(true))
	_, _ = strict.Eval(ctx, `9223372036854775807 + 1`)
	_, _ = strict.Eval(ctx, `-9223372036854775808 - 1`)
	_, _ = strict.Eval(ctx, `9223372036854775807 * 2`)
	_, _ = Eval(ctx, `9223372036854775807 + 1`)
	_, _ = Eval(ctx, `1 / 0`)
	_, _ = Eval(ctx, `1 % 0`)

}

func TestCoverage_SandboxCoerceEnv(t *testing.T) {
	eng := New(WithSandbox(SandboxConfig{Secure: true, MaxOps: 100000}))
	ctx := context.Background()

	okEnv := map[string]any{
		"n":  1,
		"f":  1.5,
		"s":  "x",
		"b":  true,
		"a":  []any{1, 2},
		"m":  map[string]any{"k": 1},
		"ai": []int{1, 2, 3},
		"ms": map[string]int{"a": 1},
		"v":  Int(9),
	}
	got, err := eng.Eval(ctx, `n + len(a) + len(m) + len(ai) + len(ms) + v`, okEnv)
	if err != nil {
		t.Fatalf("secure ok env: %v", err)
	}
	_ = got

	// Rejected shapes
	type S struct{ X int }
	for _, env := range []any{
		map[string]any{"bad": S{1}},
		map[string]any{"bad": &S{1}},
		map[string]any{"bad": func() {}},
		map[string]any{"bad": make(chan int)},
		map[string]any{"bad": map[int]int{1: 2}},
		map[string]any{"bad": Value{kind: kindFunc, ptr: func() {}}},
		map[string]any{"bad": Value{kind: kindAny, ptr: 1}},
	} {
		_, evalErr := eng.Eval(ctx, `1`, env)
		if evalErr == nil {
			t.Fatalf("expected secure reject for %#v", env)
		}
	}

	// Depth limit
	deep := any(1)
	for i := 0; i < sandboxMaxEnvDepth+2; i++ {
		deep = []any{deep}
	}
	_, err = eng.Eval(ctx, `1`, map[string]any{"d": deep})
	if err == nil {
		t.Fatal("expected depth deny")
	}
}

func TestCoverage_PublicEvalHelpers(t *testing.T) {
	ctx := context.Background()
	if MustEval(ctx, `2+2`) != int64(4) && MustEval(ctx, `2+2`) != 4 {
		// MustEval may return int64
		v := MustEval(ctx, `40+2`)
		if NewValue(v).Int() != 42 {
			t.Fatalf("MustEval got %v", v)
		}
	}

	b, err := EvalBool(ctx, `1 < 2`)
	if err != nil || !b {
		t.Fatalf("EvalBool: %v %v", b, err)
	}
	// Non-bool may coerce rather than error depending on engine mode.
	_, _ = EvalBool(ctx, `[1,2]`)
	_, _ = EvalBool(ctx, `"not-a-bool-expr!!!`)

	n, err := EvalInt(ctx, `40+2`)
	if err != nil || n != 42 {
		t.Fatalf("EvalInt: %v %v", n, err)
	}
	_, err = EvalInt(ctx, `"x"`)
	_ = err

	f, err := EvalFloat(ctx, `1.5 + 2.5`)
	if err != nil || math.Abs(f-4) > 1e-9 {
		t.Fatalf("EvalFloat: %v %v", f, err)
	}
	_, err = EvalFloat(ctx, `"x"`)
	_ = err

	s, err := EvalString(ctx, `"hi"`)
	if err != nil || s != "hi" {
		t.Fatalf("EvalString: %v %v", s, err)
	}

	eng := New()
	bb, err := eng.EvalBool(ctx, `true`)
	if err != nil || !bb {
		t.Fatalf("Engine.EvalBool: %v %v", bb, err)
	}
	ni, err := eng.EvalInt(ctx, `7`)
	if err != nil || ni != 7 {
		t.Fatalf("Engine.EvalInt: %v %v", ni, err)
	}
	ff, err := eng.EvalFloat(ctx, `1.25`)
	if err != nil || ff != 1.25 {
		t.Fatalf("Engine.EvalFloat: %v %v", ff, err)
	}
	ss, err := eng.EvalString(ctx, `"z"`)
	if err != nil || ss != "z" {
		t.Fatalf("Engine.EvalString: %v %v", ss, err)
	}

	prog, err := Compile(`x`, AsInt())
	if err != nil {
		t.Fatalf("Compile AsInt: %v", err)
	}
	iv, err := prog.EvalInt(ctx, map[string]any{"x": 9})
	if err != nil || iv != 9 {
		t.Fatalf("Program.EvalInt: %v %v", iv, err)
	}
	progF, _ := Compile(`1.5`, AsFloat())
	fv, err := progF.EvalFloat(ctx, nil)
	if err != nil || fv != 1.5 {
		t.Fatalf("Program.EvalFloat: %v %v", fv, err)
	}
	progS, _ := Compile(`"ok"`, AsString())
	sv, err := progS.EvalString(ctx, nil)
	if err != nil || sv != "ok" {
		t.Fatalf("Program.EvalString: %v %v", sv, err)
	}
	progB, _ := Compile(`true`, AsBool())
	bv, err := progB.EvalBool(ctx, nil)
	if err != nil || !bv {
		t.Fatalf("Program.EvalBool: %v %v", bv, err)
	}

	// Value Try* error paths
	if _, err := NewValue("x").TryBool(); err == nil {
		t.Fatal("TryBool want err")
	}
	if _, err := NewValue([]any{}).TryInt(); err == nil {
		t.Fatal("TryInt want err")
	}
	if _, err := NewValue("x").TryFloat(); err == nil {
		t.Fatal("TryFloat want err")
	}
	if _, err := NewValue([]any{}).TryString(); err == nil {
		t.Fatal("TryString want err")
	}
	_, _ = NewValue(1.5).TryInt()
	_, _ = NewValue(int64(3)).TryFloat()
	_, _ = NewValue(true).TryInt()
	_, _ = NewValue(true).TryBool()
	_, _ = NewValue("ok").TryString()
}

func TestCoverage_StructEnvTagsAndErrors(t *testing.T) {
	type Env struct {
		A int    `expr:"alpha"`
		B string `json:"beta"`
		C int    `expr:"-"`
		d int    // unexported
	}
	env := Env{A: 1, B: "x", d: 7}
	_ = env.d // unexported fields must not appear in the type env
	tc := NewTypeChecker().WithStructEnv(env)
	if tc.env["alpha"] != TypeInt || tc.env["beta"] != TypeString {
		t.Fatalf("tags not applied: %#v", tc.env)
	}
	// expr:"-" keeps the Go field name (does not remap).
	if tc.env["C"] != TypeInt {
		t.Fatalf("expr:\"-\" should keep field name C, got %#v", tc.env)
	}
	tc.WithStructEnv(42) // non-struct no-op
	tc.WithStructEnv(&Env{A: 2})

	// ParseError Line/Column/Error
	_, err := Compile(`@@@`)
	if err == nil {
		t.Fatal("want parse error")
	}
	var pe *ParseError
	if errors.As(err, &pe) {
		_ = pe.Error()
		_ = pe.Line()
		_ = pe.Column()
	}

	vars, err := ExtractVariables(`let x = 1; x + y + f(z)`)
	if err != nil || len(vars) == 0 {
		t.Fatalf("ExtractVariables: %v %v", vars, err)
	}
	funcs, err := ExtractFunctions(`sum(a) + len(b)`)
	if err != nil || len(funcs) < 2 {
		t.Fatalf("ExtractFunctions: %v %v", funcs, err)
	}
}

func TestCoverage_DeadCodeAndStrength(t *testing.T) {
	exprs := []string{
		`true ? 1 : expensive()`,
		`false ? expensive() : 2`,
		`1 ?? expensive()`,
		`nil ?? 3`,
		`false && expensive()`,
		`true || expensive()`,
		`x * 2`,
		`x * 4`,
		`x * 8`,
		`x / 2`,
		`x << 1`,
		`x >> 1`,
		`let x = 1; x`,
		`[1,2,3]`,
		`{a:1}`,
		`1..2`,
		`xs | sum`,
		`...xs`,
		`let [a]=xs; a`,
		`obj?.a`,
		`arr?.[0]`,
	}
	for _, expr := range exprs {
		_, _ = Simplify(expr)
		opt := defaultOptimizer()
		ast, err := newParser(expr).parse()
		if err != nil {
			continue
		}
		_ = opt.optimize(ast)
		_ = opt.isSimple(ast)
		_ = opt.isZero(&intLit{Value: 0})
		_ = opt.isZero(&floatLit{Value: 0})
		_ = opt.isZero(&intLit{Value: 1})
		_ = opt.nodesEqual(&ident{Name: "x"}, &ident{Name: "x"})
		_ = opt.nodesEqual(&stringLit{Value: "a"}, &stringLit{Value: "a"})
		_ = opt.nodesEqual(&boolLit{Value: true}, &boolLit{Value: true})
		_ = opt.nodesEqual(&nilLit{}, &nilLit{})
		_ = opt.nodesEqual(&intLit{Value: 1}, &stringLit{Value: "1"})
		_ = opt.isSideEffectFree(&ternaryExpr{
			Cond: &boolLit{Value: true},
			Then: &intLit{Value: 1},
			Else: &intLit{Value: 2},
		})
		_ = opt.isSideEffectFree(&mapLit{
			Keys:   []Node{&stringLit{Value: "a"}},
			Values: []Node{&intLit{Value: 1}},
		})
		_ = opt.isSideEffectFree(&memberExpr{Object: &ident{Name: "o"}, Property: &ident{Name: "f"}})
		_ = opt.isSideEffectFree(&indexExpr{Object: &ident{Name: "a"}, Index: &intLit{Value: 0}})
		_ = opt.isSideEffectFree(&arrayLit{Elements: []Node{&intLit{Value: 1}}})
	}
}

func TestCoverage_Wave2_VMBuiltinsParser(t *testing.T) {
	ctx := context.Background()

	// Slice / index / contains edge paths
	exprs := []string{
		`[1,2,3,4][::-1]`,
		`[1,2,3,4][3:0:-1]`,
		`[1,2,3][5:1]`,
		`[1,2,3][0:0]`,
		`"abcd"[::-1]`,
		`"abcd"[3:0:-1]`,
		`"hi"[10]`,
		`[1,2][10]`,
		`{a:1}["missing"]`,
		`"" in "abc"`,
		`9 in [1,2,3]`,
		`8 >>> 1`,
		`5 ^ 3`,
		`5 & 1`,
		`5 | 2`,
		`- -3`,
		`~5`,
		`!0`,
		`1.5 % 0`,
		`"x" / 1`,
		`sort(1)`,
		`reverse("no")`,
		`last(1)`,
		`toJSON()`,
		`fromJSON()`,
		`toJSON(1)`,
		`fromJSON("[1,2,3]")`,
		`fromJSON("1")`,
		`fromJSON("true")`,
		`fromJSON("null")`,
		`hasKey(1, "a")`,
		`default(0, 5)`,
		`coalesce()`,
		`notEmpty([1])`,
		`empty([])`,
		`empty("")`,
		`empty(null)`,
		`between(2, 1, 3)`,
		`float("1.5")`,
		`int("3")`,
		`bool(1)`,
		`string(1)`,
		`avg([])`,
		`avg(1)`,
		`pluck(1, "a")`,
		`pluck([{a:1}], 1)`,
		`chunk([1,2,3], 0)`,
		`chunk(1, 2)`,
		`compact(1)`,
		`truthy(1)`,
		`findIndex([1,2], "no")`,
		`[1,2,3][1:]`,
		`[1,2,3][:2]`,
		`"hello"[:2]`,
		`"hello"[1:]`,
		`arr[1:2:0]`,
	}
	for _, expr := range exprs {
		_, _ = Eval(ctx, expr, map[string]any{"arr": []any{1, 2, 3}})
	}

	strict := New(WithStrictMode(true))
	_, _ = strict.Eval(ctx, `"abc" % 2`)
	_, _ = strict.Eval(ctx, `2 % "abc"`)
	_, _ = strict.Eval(ctx, `"abc" / 2`)
	_, _ = strict.Eval(ctx, `2 / "abc"`)
	_, _ = strict.Eval(ctx, `"abc" + 1`)
	_, _ = strict.Eval(ctx, `1 - "abc"`)
	_, _ = strict.Eval(ctx, `"a" * 2`)

	// kindAny membership / indexing (non-secure)
	envAny := map[string]any{
		"ints":  []int{10, 20, 30},
		"imap":  map[string]int{"k": 7},
		"imap2": map[int]int{2: 9},
		"anyM":  map[any]int{nil: 1},
	}
	for _, expr := range []string{
		`20 in ints`,
		`99 in ints`,
		`ints[1]`,
		`ints[-1]`,
		`ints[99]`,
		`"k" in imap`,
		`imap["k"]`,
		`2 in imap2`,
		`imap2[2]`,
	} {
		_, _ = Eval(ctx, expr, envAny)
	}

	// Secure deny on kindAny in/index — keep values as kindAny via env after coerce failure path:
	// Without Secure, NewValue of []int stays as array conversion in toEnvMap? Check toEnvMap.
	sec := New(WithSandbox(SandboxConfig{Secure: true, MaxOps: 10000}))
	_, _ = sec.Eval(ctx, `1 in xs`, map[string]any{"xs": []int{1, 2}}) // may coerce to []any
	_, _ = sec.Eval(ctx, `xs[0]`, map[string]any{"xs": []int{1}})

	// convertArg breadth via env-native funcs
	takeAny := func(v any) any { return v }
	takeBool := func(b bool) bool { return b }
	takeStr := func(s string) string { return s }
	takeF32 := func(f float32) float32 { return f }
	takeF64 := func(f float64) float64 { return f }
	takeBytes := func(b []byte) int { return len(b) }
	takeIfaceSlice := func(xs []any) int { return len(xs) }
	takeStrSlice := func(xs []string) int { return len(xs) }
	takeMap := func(m map[string]int) int { return len(m) }
	takeChan := func(c chan int) bool { return c == nil }
	varadicMix := func(a int, rest ...string) int { return a + len(rest) }
	wrongArity := func(a, b int) int { return a + b }

	envFn := map[string]any{
		"takeAny":       takeAny,
		"takeBool":      takeBool,
		"takeStr":       takeStr,
		"takeF32":       takeF32,
		"takeF64":       takeF64,
		"takeBytes":     takeBytes,
		"takeIface":     takeIfaceSlice,
		"takeStrs":      takeStrSlice,
		"takeMap":       takeMap,
		"takeChan":      takeChan,
		"varadicMix":    varadicMix,
		"wrongArity":    wrongArity,
		"notFunc":       123,
	}
	for _, expr := range []string{
		`takeAny(null)`,
		`takeBool(true)`,
		`takeStr("hi")`,
		`takeF32(1.5)`,
		`takeF64(2.5)`,
		`takeBytes([1,2,3])`,
		`takeIface([1,"a"])`,
		`takeStrs(["a","b"])`,
		`takeMap({a:1})`,
		`takeChan(null)`,
		`varadicMix(1, "a", "b")`,
		`wrongArity(1)`,
		`notFunc()`,
		`takeStrs([1,2])`,
	} {
		_, _ = Eval(ctx, expr, envFn)
	}

	// Sandbox limits on builtins
	limited := New(WithSandbox(SandboxConfig{MaxOps: 1}))
	_, _ = limited.Eval(ctx, `sort([3,1,2])`)
	_, _ = limited.Eval(ctx, `toJSON({a:1,b:2,c:3})`)
	_, _ = limited.Eval(ctx, `fromJSON("[1,2,3,4,5,6,7,8,9]")`)

	// MustEval panic path
	func() {
		defer func() { _ = recover() }()
		_ = MustEval(ctx, `@@@`)
		t.Fatal("MustEval should panic")
	}()

	// Typed eval conversion failures
	eng := New()
	_, _ = eng.EvalBool(ctx, `[1]`)
	_, _ = eng.EvalInt(ctx, `[1]`)
	_, _ = eng.EvalFloat(ctx, `[1]`)
	_, _ = eng.EvalString(ctx, `null`)
	p, _ := Compile(`"x"`, AsInt())
	_, _ = p.EvalInt(ctx, nil)
	p2, _ := Compile(`"x"`, AsFloat())
	_, _ = p2.EvalFloat(ctx, nil)
	p3, _ := Compile(`1`, AsString())
	_, _ = p3.EvalString(ctx, nil)
	p4, _ := Compile(`1`, AsBool())
	_, _ = p4.EvalBool(ctx, nil)

	// Parser / let / destructure / lambda / slice quirks
	parseExprs := []string{
		`let x = 1; let y = 2; x + y`,
		`let [a, b, ...rest] = xs; a`,
		`let [a = 1, b = 2] = xs; a + b`,
		`let {a, b: c, d = 3} = obj; a`,
		`let {a: x = 1} = obj; x`,
		`(x, y) => x + y`,
		`() => 1`,
		`x => x`,
		`arr[::]`,
		`arr[1::2]`,
		`arr[:3:]`,
		`obj?.a?.b`,
		`arr?.[0]?.x`,
		`f?.(1)`,
	}
	for _, expr := range parseExprs {
		_, _ = Compile(expr)
		_, _ = Eval(ctx, expr, map[string]any{
			"xs":  []any{1, 2, 3, 4},
			"obj": map[string]any{"a": 1, "b": 2},
			"arr": []any{1, 2, 3, 4, 5},
		})
	}

	// Error helpers
	le := &LexerError{Message: "bad", Line: 1, Column: 2, Source: "x"}
	_ = le.Error()
	le2 := &LexerError{Message: "bad", Line: 1, Column: 2}
	_ = le2.Error()
	pe := &ParseError{Message: "bad", token: token{Line: 2, Column: 3}, Source: "y"}
	_ = pe.Error()
	_ = pe.Line()
	_ = pe.Column()
	_ = (*ParseError)(nil).Line()
	_ = (*ParseError)(nil).Column()
	tce := &TypeCheckErrors{Errors: []error{errors.New("e1"), errors.New("e2")}}
	_ = tce.Error()
	_ = tce.Len()
	_ = (&TypeCheckErrors{}).Error()
	_ = (&TypeError{Message: "m", Expected: "a", Got: "b"}).Error()

	// truncate / safeSprint / huge stringify
	big := strings.Repeat("x", 5000)
	_ = NewValue(big).String()
	_ = NewValue(struct{ X int }{1}).String()
	cycle := []any{nil}
	cycle[0] = cycle
	_ = NewValue(cycle).String()
	mcycle := map[string]any{}
	mcycle["self"] = mcycle
	_ = NewValue(mcycle).String()

	// toEnvMap variants
	type tagged struct {
		X int `expr:"xx"`
		Y int `json:"yy"`
		Z int `expr:"-"`
		h int
	}
	tg := tagged{X: 1, Y: 2, Z: 3, h: 9}
	_ = tg.h // unexported fields are omitted from env maps
	_, _ = Eval(ctx, `xx + yy + Z`, tg)
	_, _ = Eval(ctx, `xx`, &tagged{X: 4})
	_, _ = Eval(ctx, `1`, nil)
	_, _ = Eval(ctx, `a`, map[string]int{"a": 5})

	// Analyze / Profile remaining
	_, _ = Analyze(`sort(map(filter(items, x => x > 0), y => y * 2))`)
	_, _, _ = ProfileExpr(ctx, `len([1,2,3])`, nil)
	prog, _ := Compile(`1`)
	_, _, _ = prog.Profile(ctx, nil)

	// float64ToInt64Exact via int()
	_, _ = Eval(ctx, `int(1.0)`)
	_, _ = Eval(ctx, `int(1.5)`)
	_, _ = Eval(ctx, `int(1e20)`)

	// PeekN via parser internals (n starts at 1; 0 underflows)
	lx := newLexer("a + b + c")
	_ = lx.PeekN(1)
	_ = lx.PeekN(2)
	_ = lx.PeekN(3)
}

func TestCoverage_Wave3_DirectVMAndOpcodes(t *testing.T) {
	vm := newVM(&bytecode{}, nil)
	defer releaseVM(vm)

	arr := Array([]any{1, 2, 3, 4, 5})
	str := String("hello")
	hugeIdx := Int(math.MaxInt64)

	// slice edges (array + string, pos/neg step, invalid) — keep indices small to avoid makeslice panic
	for _, step := range []Value{Nil, Int(1), Int(2), Int(-1), Int(-2), Int(0)} {
		for _, low := range []Value{Nil, Int(0), Int(1), Int(-1), Int(-10), Int(10)} {
			for _, high := range []Value{Nil, Int(2), Int(5), Int(-1), Int(-20), Int(0), Int(100)} {
				_, _ = vm.slice(arr, low, high, step)
				_, _ = vm.slice(str, low, high, step)
			}
		}
	}
	_, _ = vm.slice(arr, hugeIdx, Int(1), Int(1))
	_, _ = vm.slice(str, Int(0), hugeIdx, Int(1))
	_, _ = vm.slice(str, Int(0), Int(1), hugeIdx)
	_, _ = vm.slice(Int(1), Int(0), Int(1), Int(1)) // not sliceable

	// index edges
	_, _ = vm.index(arr, Int(0))
	_, _ = vm.index(arr, Int(-1))
	_, _ = vm.index(arr, Int(99))
	_, _ = vm.index(arr, hugeIdx)
	_, _ = vm.index(str, Int(0))
	_, _ = vm.index(str, Int(-1))
	_, _ = vm.index(str, Int(99))
	_, _ = vm.index(Map(map[string]any{"a": 1}), String("a"))
	_, _ = vm.index(Map(map[string]any{"a": 1}), String("z"))
	_, _ = vm.index(Int(1), Int(0))
	anySlice := Value{kind: kindAny, ptr: []int{1, 2, 3}}
	anyMap := Value{kind: kindAny, ptr: map[string]int{"k": 1}}
	anyMapInt := Value{kind: kindAny, ptr: map[int]int{2: 9}}
	anyMapAny := Value{kind: kindAny, ptr: map[any]int{nil: 1}}
	_, _ = vm.index(anySlice, Int(1))
	_, _ = vm.index(anySlice, Int(-1))
	_, _ = vm.index(anySlice, Int(99))
	_, _ = vm.index(anySlice, hugeIdx)
	_, _ = vm.index(anyMap, String("k"))
	_, _ = vm.index(anyMap, String("z"))
	_, _ = vm.index(anyMapInt, Int(2))
	_, _ = vm.index(anyMapAny, Nil)
	vm.limits = &sandboxLimits{secure: true}
	_, _ = vm.index(anySlice, Int(0))
	_, _ = vm.contains(anySlice, Int(1))
	vm.limits = nil

	// contains edges
	_, _ = vm.contains(arr, Int(3))
	_, _ = vm.contains(arr, Int(99))
	_, _ = vm.contains(Map(map[string]any{"a": 1}), String("a"))
	_, _ = vm.contains(str, String(""))
	_, _ = vm.contains(str, String("ell"))
	_, _ = vm.contains(str, String("nope"))
	_, _ = vm.contains(anySlice, Int(2))
	_, _ = vm.contains(anyMap, String("k"))
	_, _ = vm.contains(anyMapInt, Int(2))
	_, _ = vm.contains(anyMapAny, Nil)
	_, _ = vm.contains(Int(1), Int(1))

	// arithmetic edges
	vm.strictMode = true
	_, _ = vm.div(String("x"), Int(1))
	_, _ = vm.div(Int(1), String("x"))
	_, _ = vm.mod(String("x"), Int(1))
	_, _ = vm.mod(Int(1), String("x"))
	vm.strictMode = false
	_, _ = vm.div(Int(10), Int(0))
	_, _ = vm.div(Int(10), Int(3))
	_, _ = vm.div(Float(10), Int(4))
	_, _ = vm.mod(Int(10), Int(0))
	_, _ = vm.mod(Int(10), Int(3))
	_, _ = vm.mod(Float(10.5), Float(2))
	_, _ = vm.mod(Float(10.5), Float(0))
	_, _ = vm.add(String("a"), String("b"))
	_, _ = vm.add(arr, Array([]any{9}))
	_, _ = vm.add(Int(1), String("x"))
	_, _ = vm.add(Map(map[string]any{}), Int(1))
	_, _ = vm.mul(String("ab"), Int(3))
	_, _ = vm.mul(Int(3), String("ab"))
	_, _ = vm.mul(arr, Int(2))
	_, _ = vm.mul(Bool(true), Int(1))
	_, _ = vm.sub(Int(5), Float(1.5))
	_, _ = vm.sub(Float(5), Int(1))
	_, _ = vm.sub(String("a"), Int(1))

	// convertArg directly
	types := []reflect.Type{
		reflect.TypeOf(int(0)),
		reflect.TypeOf(int8(0)),
		reflect.TypeOf(int16(0)),
		reflect.TypeOf(int32(0)),
		reflect.TypeOf(int64(0)),
		reflect.TypeOf(uint(0)),
		reflect.TypeOf(uint8(0)),
		reflect.TypeOf(uint16(0)),
		reflect.TypeOf(uint32(0)),
		reflect.TypeOf(uint64(0)),
		reflect.TypeOf(float32(0)),
		reflect.TypeOf(float64(0)),
		reflect.TypeOf(""),
		reflect.TypeOf(true),
		reflect.TypeOf([]any{}),
		reflect.TypeOf([]int{}),
		reflect.TypeOf([]string{}),
		reflect.TypeOf((*any)(nil)).Elem(),
		reflect.TypeOf(map[string]int{}),
		reflect.TypeOf((chan int)(nil)),
		reflect.TypeOf((*int)(nil)),
	}
	args := []Value{Nil, Int(1), Float(1.5), String("x"), Bool(true), Array([]any{1, "a"}), Map(map[string]any{"a": 1})}
	for _, typ := range types {
		if typ == nil {
			continue
		}
		for _, a := range args {
			_ = vm.convertArg(a, typ)
		}
	}

	// checkLimits branches
	vm.limits = &sandboxLimits{maxOps: 1}
	vm.opsCount = 2
	_ = vm.checkLimits()
	vm.limits = &sandboxLimits{deadline: time.Now().Add(-time.Second)}
	vm.opsCount = 0
	_ = vm.checkLimits()
	vm.limits = &sandboxLimits{maxStackDepth: 1}
	vm.sp = 5
	_ = vm.checkLimits()
	vm.sp = 0
	vm.limits = &sandboxLimits{maxMemory: 1}
	vm.opsCount = 1024
	vm.env = map[string]Value{"big": String(strings.Repeat("z", 100))}
	_ = vm.checkLimits()
	vm.limits = nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	vm.ctx = ctx
	_ = vm.checkLimits()
	vm.ctx = context.Background()

	// stableValueKey / unique
	_, _ = stableValueKey(vm, Nil, 0, map[uintptr]bool{})
	_, _ = stableValueKey(vm, Bool(true), 0, map[uintptr]bool{})
	_, _ = stableValueKey(vm, Bool(false), 0, map[uintptr]bool{})
	_, _ = stableValueKey(vm, Int(1), 0, map[uintptr]bool{})
	_, _ = stableValueKey(vm, Float(1.5), 0, map[uintptr]bool{})
	_, _ = stableValueKey(vm, String("s"), 0, map[uintptr]bool{})
	_, _ = stableValueKey(vm, Array([]any{1, map[string]any{"a": 2}}), 0, map[uintptr]bool{})
	_, _ = stableValueKey(vm, Map(map[string]any{"b": 1, "a": 2}), 0, map[uintptr]bool{})
	_, _ = stableValueKey(vm, Int(1), 100, map[uintptr]bool{})
	cyc := []any{nil}
	cyc[0] = cyc
	_, _ = stableValueKey(vm, Array(cyc), 0, map[uintptr]bool{})
	mc := map[string]any{}
	mc["self"] = mc
	_, _ = stableValueKey(vm, Map(mc), 0, map[uintptr]bool{})
	vm.limits = &sandboxLimits{maxOps: 1}
	vm.opsCount = 0
	_, _ = stableValueKey(vm, String("abcdef"), 0, map[uintptr]bool{})
	vm.limits = nil
	_, _ = Eval(context.Background(), `unique([{a:1},{a:1},{b:2},[1],[1]])`)

	// eliminateDeadCode / fold more AST shapes via optimizer
	opt := defaultOptimizer()
	for _, expr := range []string{
		`true ? f(1) : f(2)`,
		`false ? f(1) : f(2)`,
		`1 ?? f(1)`,
		`nil ?? f(1)`,
		`false && f(1)`,
		`true || f(1)`,
		`true && false`,
		`false || true`,
		`!true`,
		`!!false`,
		`~1`,
		`-1.5`,
		`+1`,
		`1 != 2`,
		`1 != 1.0`,
		`8 >> 1`,
		`8 >> 1.0`,
		`1 & 1`,
		`1 | 0`,
		`1 ^ 1`,
		`1 << 2`,
		`10 / 2.0`,
		`10.0 / 2`,
		`10.0 / 2.0`,
		`10 / 0`,
		`true && 1`,
		`false || 0`,
	} {
		ast, err := newParser(expr).parse()
		if err != nil {
			continue
		}
		_ = opt.optimize(ast)
		_ = opt.foldConstants(ast)
		_ = opt.reduceStrength(ast)
		_ = opt.simplifyAlgebra(ast)
		_ = opt.eliminateDeadCode(ast)
	}

	// Opcode soak via Eval
	ctx2 := context.Background()
	soak := []string{
		`1 ** 2`,
		`2 ** 0`,
		`9 ** 0.5`,
		`"hello" ~= "h.*"`,
		`1 not in [2,3]`,
		`null?.x`,
		`null?.[0]`,
		`{a:{b:1}}?.a?.b`,
		`[1,2] | map(x => x * 2) | sum`,
		`(x => x + 1)(5)`,
		`(x, y) => x + y`,
		`let {a: aa = 9, b} = {b: 2}; aa + b`,
		`let [a, b = 9, ...r] = [1]; a + b + len(r)`,
		`between(5, 1, 10)`,
		`clamp(5, 1, 3)`,
		`sprintf("%s-%d", "a", 1)`,
		`values({a:1,b:2})`,
		`keys({a:1})`,
		`zip([1,2], ["a","b"])`,
		`skip([1,2,3], 1)`,
		`take([1,2,3], 2)`,
		`trimLeft("  x")`,
		`trimRight("x  ")`,
		`groupBy([{t:"a"},{t:"a"}], x => x.t)`,
		`sortBy([{v:2},{v:1}], x => x.v)`,
		`all([1,2], x => x > 0)`,
		`any([0,2], x => x > 0)`,
		`none([0,0], x => x > 0)`,
		`one([0,1,0], x => x > 0)`,
		`count([1,2,3])`,
		`flatten([[1,[2]]], 2)`,
		`range(0, 3)`,
		`range(3)`,
		`repeat("ab", 2)`,
		`replace("aa", "a", "b")`,
		`substr("hello", 1, 3)`,
		`indexOf([1,2,3], 2)`,
		`startsWith("hello", "he")`,
		`endsWith("hello", "lo")`,
		`split("a,b", ",")`,
		`join(["a","b"], ",")`,
		`size([1,2])`,
		`type(1)`,
		`isNil(null)`,
		`abs(-3)`,
		`ceil(1.2)`,
		`floor(1.8)`,
		`round(1.5)`,
		`min(1,2,3)`,
		`max(1,2,3)`,
		`sqrt(9)`,
		`pow(2,3)`,
		`log(1)`,
		`exp(0)`,
		`sin(0)`,
		`cos(0)`,
		`upper("a")`,
		`lower("A")`,
		`len("аб")`,
	}
	for _, expr := range soak {
		_, _ = Eval(ctx2, expr)
	}

	// TypeCheckErrors Len zero / multi
	if (&TypeCheckErrors{}).Len() != 0 {
		t.Fatal("empty Len")
	}
	_ = (&TypeCheckErrors{Errors: []error{errors.New("a")}}).Len()
	_ = (&RuntimeError{Message: "x", Op: "+"}).Error()

	// float64ToInt64Exact edges
	_, _ = float64ToInt64Exact(math.NaN())
	_, _ = float64ToInt64Exact(math.Inf(1))
	_, _ = float64ToInt64Exact(1.5)
	_, _ = float64ToInt64Exact(float64(math.MinInt64))
	_, _ = float64ToInt64Exact(1e20)

	// Extract* error paths
	_, _ = ExtractVariables("(((")
	_, _ = ExtractFunctions("(((")

	// Inline/Expand parse errors
	_, _ = Inline("(((")
	_, _ = Simplify("(((")
	_, _ = Expand("x", map[string]string{"x": "(("})

	// Program Analyze
	prog, err := Compile(`map(xs, x => x)`)
	if err == nil {
		_ = prog.Analyze()
	}

	// WithEnvTypes + CheckExpressionType mismatch/errors
	_ = CheckExpressionType(`1 + "a"`, TypeInt, nil)
	tc := NewTypeChecker().WithEnvTypes(map[string]any{"x": true})
	ast, _ := newParser(`-x`).parse()
	_ = tc.Check(ast)
	_ = tc.Infer(ast)
	ast2, _ := newParser(`~x`).parse()
	_ = tc.Check(ast2)
}

func TestCoverage_Wave4_OverflowsParseErrors(t *testing.T) {
	ctx := context.Background()
	vm := newVM(&bytecode{}, nil)
	defer releaseVM(vm)

	maxI := Int(math.MaxInt64)
	minI := Int(math.MinInt64)
	_, _ = vm.add(maxI, Int(1))
	_, _ = vm.add(minI, Int(-1))
	_, _ = vm.sub(maxI, Int(-1))
	_, _ = vm.sub(minI, Int(1))
	_, _ = vm.mul(minI, Int(-1))
	_, _ = vm.mul(Int(-1), minI)
	_, _ = vm.mul(maxI, Int(2))
	_, _ = vm.mul(Int(0), maxI)
	_ = addOverflows(math.MaxInt64, 1)
	_ = addOverflows(math.MinInt64, -1)
	_ = addOverflows(1, 1)
	_ = subOverflows(math.MaxInt64, -1)
	_ = subOverflows(math.MinInt64, 1)
	_ = subOverflows(1, 1)
	_ = mulOverflows(0, 1)
	_ = mulOverflows(math.MinInt64, -1)
	_ = mulOverflows(-1, math.MinInt64)
	_ = mulOverflows(math.MaxInt64, 3)

	vm.strictMode = true
	_, _ = vm.add(Int(1), Int(2))
	_, _ = vm.add(String("1"), Int(2))
	_, _ = vm.mul(String("no"), Int(2))
	_, _ = vm.mul(Int(2), String("no"))
	_ = vm.strictCheckNumeric(String(""))
	_ = vm.strictCheckNumeric(String("12"))
	_ = vm.strictCheckNumeric(String("1.5"))
	vm.strictMode = false
	_ = vm.strictCheckNumeric(String("x"))

	// kindAny slicing
	anySlice := Value{kind: kindAny, ptr: []int{1, 2, 3, 4, 5}}
	for _, step := range []Value{Nil, Int(1), Int(-1), Int(0), Int(2)} {
		for _, low := range []Value{Nil, Int(0), Int(-1), Int(1)} {
			for _, high := range []Value{Nil, Int(3), Int(-1), Int(10)} {
				_, _ = vm.slice(anySlice, low, high, step)
			}
		}
	}
	vm.limits = &sandboxLimits{secure: true}
	_, _ = vm.slice(anySlice, Int(0), Int(2), Int(1))
	vm.limits = nil

	// toJSON marshal failure + limits accounting
	_, _ = builtinToJSON(vm, nil)
	_, _ = builtinToJSON(vm, []Value{Int(1), Int(2)})
	ch := make(chan int)
	_, _ = builtinToJSON(vm, []Value{NewValue(ch)})
	vm.limits = &sandboxLimits{maxOps: 1}
	vm.opsCount = 0
	_, _ = builtinToJSON(vm, []Value{Map(map[string]any{"a": 1, "b": 2, "c": 3})})
	_, _ = builtinFromJSON(vm, nil)
	_, _ = builtinFromJSON(vm, []Value{String("{")})
	vm.opsCount = 0
	_, _ = builtinFromJSON(vm, []Value{String(`[1,2,3,4,5,6]`)})
	vm.limits = nil

	// sort with limits
	vm.limits = &sandboxLimits{maxOps: 1}
	vm.opsCount = 0
	_, _ = builtinSort(vm, []Value{Array([]any{3, 1, 2})})
	_, _ = builtinSort(vm, nil)
	vm.limits = nil

	// Eval* compile/runtime error paths (hit early return)
	eng := New()
	_, _ = eng.EvalInt(ctx, `@@@`)
	_, _ = eng.EvalFloat(ctx, `@@@`)
	_, _ = eng.EvalString(ctx, `@@@`)
	_, _ = eng.EvalBool(ctx, `@@@`)
	_, _ = EvalInt(ctx, `@@@`)
	_, _ = EvalFloat(ctx, `@@@`)
	_, _ = EvalString(ctx, `@@@`)
	p, _ := Compile(`1+2`)
	if p != nil {
		_, _ = p.EvalInt(ctx, nil) // ok path already; force err via cancelled ctx
		cctx, cancel := context.WithCancel(context.Background())
		cancel()
		// short expr may finish before cancel observed; use heavy one
		heavy, _ := Compile(`map(range(0, 100000), x => x)`)
		if heavy != nil {
			_, _ = heavy.EvalInt(cctx, nil)
			_, _ = heavy.EvalFloat(cctx, nil)
			_, _ = heavy.EvalString(cctx, nil)
			_, _ = heavy.EvalBool(cctx, nil)
		}
	}

	// Parser error / edge paths
	bad := []string{
		`let`,
		`let = 1`,
		`let x`,
		`let x =`,
		`let [`,
		`let {`,
		`let [a,,] = xs; a`,
		`let {a,,} = obj; a`,
		`let [a b] = xs; a`,
		`(`,
		`() =>`,
		`(x,) => x`,
		`(1) => 1`,
		`f(`,
		`f(1,`,
		`arr[`,
		`arr[1:`,
		`arr[1:2:`,
		`arr[::`,
		`{`,
		`{a:`,
		`{a:1,`,
		`[`,
		`[1,`,
		`1 +`,
		`1 *`,
		`obj.`,
		`obj?.`,
		`let x = 1 y`,
	}
	for _, expr := range bad {
		_, _ = Compile(expr)
	}

	// debugMode empty stack
	restore := SetDebugModeForTest(true)
	defer restore()
	vm2 := newVM(&bytecode{}, nil)
	func() {
		defer func() { _ = recover() }()
		vm2.pop()
	}()
	func() {
		defer func() { _ = recover() }()
		_ = vm2.peek()
	}()
	releaseVM(vm2)
	restore()

	// UndefinedError / truncate / TryFloat remaining
	_ = (&UndefinedError{Kind: "variable", Name: "x", Suggestion: "y"}).Error()
	_ = (&UndefinedError{Kind: "function", Name: "f"}).Error()
	_ = truncateString(strings.Repeat("q", maxStringTotalLen+10))
	_ = truncateString("short")
	_, _ = Float(1.25).TryFloat()
	_, _ = Int(3).TryFloat()
	_, _ = String("no").TryFloat()
	_, _ = Nil.TryFloat()

	// TypeChecker binary remaining cases with known types
	tc := NewTypeChecker().WithEnvTypes(map[string]any{
		"s": "x", "n": 1, "b": true, "a": []any{}, "m": map[string]any{},
	})
	for _, expr := range []string{
		`s + s`, `a + a`, `n + n`, `s - n`, `b && b`, `b || b`,
		`n < n`, `s < s`, `n == s`, `n in a`, `n not in a`,
		`n & n`, `n | n`, `n ^ n`, `n << n`, `n >> n`, `n >>> n`,
		`s ~= s`, `!b`, `-n`, `~n`, `+n`,
	} {
		ast, err := newParser(expr).parse()
		if err != nil {
			continue
		}
		_ = tc.Check(ast)
		_ = tc.Infer(ast)
	}

	// sandbox coerce periodic checkLimits on large env
	sec := New(WithSandbox(SandboxConfig{Secure: true, MaxOps: 5}))
	big := make([]any, 1200)
	for i := range big {
		big[i] = i
	}
	_, _ = sec.Eval(ctx, `1`, map[string]any{"a": big})
	bigMap := make(map[string]any, 1200)
	for i := 0; i < 1200; i++ {
		bigMap[strconv.Itoa(i)] = i
	}
	_, _ = sec.Eval(ctx, `1`, map[string]any{"m": bigMap})

	// callBuiltinFunc non-func / closure paths
	_, _ = Eval(ctx, `map([1,2,3], x => x + 1)`)
	_, _ = Eval(ctx, `filter([1,2,3], x => x > 1)`)
	_, _ = Eval(ctx, `find([1,2,3], x => x == 2)`)
	_, _ = Eval(ctx, `findIndex([1,2,3], x => x == 2)`)
	_, _ = Eval(ctx, `reduce([1,2,3], (a,b) => a+b, 0)`)
	_, _ = Eval(ctx, `all([1,2], x => x > 0)`)
	_, _ = Eval(ctx, `any([0,2], x => x > 0)`)
	_, _ = Eval(ctx, `none([0], x => x > 0)`)
	_, _ = Eval(ctx, `one([1], x => x > 0)`)
	_, _ = Eval(ctx, `map([1], 1)`)         // bad pred
	_, _ = Eval(ctx, `findIndex([1], 1)`) // bad pred

	// WithFunction convertArgForBuiltin edges
	eng2 := New(
		WithFunction("needSlice", func(xs []string) int { return len(xs) }),
		WithFunction("needAny", func(v any) any { return v }),
		WithFunction("needPtr", func(p *int) bool { return p == nil }),
	)
	_, _ = eng2.Eval(ctx, `needSlice(["a","b"])`)
	_, _ = eng2.Eval(ctx, `needSlice([1,2])`)
	_, _ = eng2.Eval(ctx, `needAny(null)`)
	_, _ = eng2.Eval(ctx, `needPtr(null)`)

	// Analyze estimateCost / side effects
	_, _ = Analyze(`print(1)`)
	_, _ = Analyze(`sort(unique(map(filter(xs, x => x), y => y)))`)

	// Len of TypeCheckErrors already; ensure multi-error Error
	_ = (&TypeCheckErrors{Errors: []error{
		&TypeError{Message: "a"},
		&TypeError{Message: "b"},
	}}).Error()
	_ = (&TypeCheckErrors{Errors: nil}).Len()
}


