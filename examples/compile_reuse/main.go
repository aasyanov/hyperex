// Compile once, evaluate many times — the primary hyperex pattern.
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()
	engine := hyperex.New()

	program, err := engine.Compile(`price * qty + fee`)
	must(err)
	fmt.Printf("Source:     %s\n", program.Source())
	fmt.Printf("ResultHint: %s (no As* at Compile)\n", program.ResultHint())

	const (
		priceKey = "price"
		qtyKey   = "qty"
		feeKey   = "fee"
	)
	for _, env := range []map[string]any{
		{priceKey: 10, qtyKey: 2, feeKey: 1},
		{priceKey: 5, qtyKey: 4, feeKey: 0},
		{priceKey: 7, qtyKey: 3, feeKey: 2},
	} {
		result, evalErr := program.Eval(ctx, env)
		must(evalErr)
		fmt.Printf("  Eval %v => %v\n", env, result)
	}

	fmt.Println("\n=== Typed Program.Eval* ===")
	sum := engine.MustCompile(`a + b`)
	n, err := sum.EvalInt(ctx, map[string]any{"a": 3, "b": 4})
	must(err)
	fmt.Printf("  EvalInt:  %d\n", n)

	okProg := engine.MustCompile(`n >= 10`)
	ok, err := okProg.EvalBool(ctx, map[string]any{"n": 12})
	must(err)
	fmt.Printf("  EvalBool: %v\n", ok)

	fmt.Println("\n=== Package Compile + ResultHint ===")
	// AsFloat records a compile-time expectation; ResultHint() exposes it
	asFloat, err := hyperex.Compile(`1.5 * 2`, hyperex.AsFloat())
	must(err)
	fmt.Printf("  Source:     %s\n", asFloat.Source())
	fmt.Printf("  ResultHint: %s\n", asFloat.ResultHint())
	f, err := asFloat.EvalFloat(ctx, nil)
	must(err)
	fmt.Printf("  EvalFloat:  %v\n", f)

	strProg := hyperex.MustCompile(`upper("ok")`)
	s, err := strProg.EvalString(ctx, nil)
	must(err)
	fmt.Printf("  MustCompile+EvalString: %q\n", s)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
