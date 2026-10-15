// Compile As* hints and optimizer toggles — Eval results match; costs differ.
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()
	eng := hyperex.New()

	fmt.Println("=== Compile As* options ===")
	asBool, err := eng.Compile(`x > 10`, hyperex.AsBool())
	must(err)
	b, err := asBool.EvalBool(ctx, map[string]any{"x": 15})
	must(err)
	fmt.Printf("  AsBool   ResultHint=%-6s EvalBool => %v\n", asBool.ResultHint(), b)

	asInt, err := eng.Compile(`price * qty`, hyperex.AsInt())
	must(err)
	n, err := asInt.EvalInt(ctx, map[string]any{"price": 3.9, "qty": 4})
	must(err)
	fmt.Printf("  AsInt    ResultHint=%-6s EvalInt  => %d\n", asInt.ResultHint(), n)

	asFloat, err := eng.Compile(`1.5 * 2`, hyperex.AsFloat())
	must(err)
	f, err := asFloat.EvalFloat(ctx, nil)
	must(err)
	fmt.Printf("  AsFloat  ResultHint=%-6s EvalFloat=> %v\n", asFloat.ResultHint(), f)

	asStr, err := eng.Compile(`upper("hi")`, hyperex.AsString())
	must(err)
	s, err := asStr.EvalString(ctx, nil)
	must(err)
	fmt.Printf("  AsString ResultHint=%-6s EvalString=> %q\n", asStr.ResultHint(), s)

	fmt.Println("\n=== Optimizer effect (same result, fewer ops) ===")
	// Constant folding collapses literal trees at compile time.
	// Profile.OpsCount makes the difference visible; Eval values stay equal.
	expr := `2 + 3 * 4`
	folded, err := eng.Compile(expr)
	must(err)
	raw, err := eng.Compile(expr, hyperex.Unoptimized())
	must(err)

	pf, rf, err := folded.Profile(ctx, nil)
	must(err)
	pr, rr, err := raw.Profile(ctx, nil)
	must(err)
	fmt.Printf("  default    => %v  OpsCount=%d\n", rf, pf.OpsCount)
	fmt.Printf("  Unoptimized=> %v  OpsCount=%d\n", rr, pr.OpsCount)

	simp, err := hyperex.Simplify(expr)
	must(err)
	fmt.Printf("  Simplify(%q) => %q  (string-level fold)\n", expr, simp)

	fmt.Println("\n=== Engine optimize toggles ===")
	noOpt := hyperex.New(hyperex.WithOptimize(false))
	p1, err := noOpt.Compile(`2 + 2`)
	must(err)
	prof1, r1, err := p1.Profile(ctx, nil)
	must(err)
	fmt.Printf("  WithOptimize(false) 2+2 => %v  OpsCount=%d\n", r1, prof1.OpsCount)

	// Strength reduction / folding are on by default when WithOptimize(true).
	// Disable individually only for A/B debugging.
	foldedEng := hyperex.New(
		hyperex.WithConstantFolding(true),
		hyperex.WithStrengthReduction(true),
	)
	p2, err := foldedEng.Compile(`x * 2 + 0`)
	must(err)
	r2, err := p2.Eval(ctx, map[string]any{"x": 21})
	must(err)
	fmt.Printf("  Folding+StrengthReduction x*2+0 (x=21) => %v\n", r2)

	combo, err := eng.Compile(`100 / 4`, hyperex.AsInt(), hyperex.Unoptimized())
	must(err)
	n2, err := combo.EvalInt(ctx, nil)
	must(err)
	fmt.Printf("  AsInt+Unoptimized 100/4 => %d  hint=%s\n", n2, combo.ResultHint())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
