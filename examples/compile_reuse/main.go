// Package main demonstrates compile-once, eval-many reuse.
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

const (
	feeField = "fee"
	exPrice  = "price"
	exQty    = "qty"
)

func main() {
	ctx := context.Background()
	engine := hyperex.New()

	program, err := engine.Compile(`price * qty + fee`)
	must(err)
	fmt.Printf("Source:         %s\n", program.Source())

	for _, env := range []map[string]any{
		{exPrice: 10, exQty: 2, feeField: 1},
		{exPrice: 5, exQty: 4, feeField: 0},
		{exPrice: 7, exQty: 3, feeField: 2},
	} {
		result, evalErr := program.Eval(ctx, env)
		must(evalErr)
		fmt.Printf("Eval %v: %v\n", env, result)
	}

	sum := engine.MustCompile(`a + b`)
	fmt.Printf("MustCompile:    %s\n", sum.Source())
	n, err := sum.EvalInt(ctx, map[string]any{"a": 3, "b": 4})
	must(err)
	fmt.Printf("EvalInt:        %d\n", n)

	okProg := engine.MustCompile(`n >= 10`)
	ok, err := okProg.EvalBool(ctx, map[string]any{"n": 12})
	must(err)
	fmt.Printf("EvalBool:       %v\n", ok)

	pkgProg, err := hyperex.Compile(`1.5 * 2`)
	must(err)
	f, err := pkgProg.EvalFloat(ctx, nil)
	must(err)
	fmt.Printf("pkg Compile+EvalFloat: %v\n", f)

	strProg := hyperex.MustCompile(`upper("ok")`)
	s, err := strProg.EvalString(ctx, nil)
	must(err)
	fmt.Printf("pkg MustCompile+EvalString: %q\n", s)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
