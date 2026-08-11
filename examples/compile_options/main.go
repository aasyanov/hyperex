// Compile options (As*) and engine optimization toggles.
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
	fmt.Printf("  AsBool  x>10 (x=15): %v\n", b)

	asInt, err := eng.Compile(`price * qty`, hyperex.AsInt())
	must(err)
	n, err := asInt.EvalInt(ctx, map[string]any{"price": 3.9, "qty": 4})
	must(err)
	fmt.Printf("  AsInt   price*qty:   %d\n", n)

	asFloat, err := eng.Compile(`1.5 * 2`, hyperex.AsFloat())
	must(err)
	f, err := asFloat.EvalFloat(ctx, nil)
	must(err)
	fmt.Printf("  AsFloat 1.5*2:       %v\n", f)

	asStr, err := eng.Compile(`upper("hi")`, hyperex.AsString())
	must(err)
	s, err := asStr.EvalString(ctx, nil)
	must(err)
	fmt.Printf("  AsString upper(hi):  %q\n", s)

	fmt.Println("\n=== Unoptimized compile ===")
	raw, err := eng.Compile(`2 + 3 * 4`, hyperex.Unoptimized())
	must(err)
	v, err := raw.Eval(ctx, nil)
	must(err)
	fmt.Printf("  Unoptimized 2+3*4 => %v\n", v)

	fmt.Println("\n=== Engine optimize options ===")
	noOpt := hyperex.New(hyperex.WithOptimize(false))
	p1, err := noOpt.Compile(`2 + 2`)
	must(err)
	r1, err := p1.Eval(ctx, nil)
	must(err)
	fmt.Printf("  WithOptimize(false) 2+2 => %v\n", r1)

	folded := hyperex.New(
		hyperex.WithConstantFolding(true),
		hyperex.WithStrengthReduction(true),
	)
	p2, err := folded.Compile(`x * 2 + 0`)
	must(err)
	r2, err := p2.Eval(ctx, map[string]any{"x": 21})
	must(err)
	fmt.Printf("  WithConstantFolding+StrengthReduction x*2+0 (x=21) => %v\n", r2)

	combo, err := eng.Compile(`100 / 4`, hyperex.AsInt(), hyperex.Unoptimized())
	must(err)
	n2, err := combo.EvalInt(ctx, nil)
	must(err)
	fmt.Printf("  AsInt+Unoptimized 100/4 => %d\n", n2)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
