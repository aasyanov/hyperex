// Package main demonstrates package-level hyperex Eval helpers.
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()

	result, err := hyperex.Eval(ctx, `2 + 2 * 3`)
	must(err)
	fmt.Printf("Eval:           %v\n", result)

	n, err := hyperex.EvalInt(ctx, `10 * quantity`, map[string]any{"quantity": 3})
	must(err)
	fmt.Printf("EvalInt:        %d\n", n)

	ok, err := hyperex.EvalBool(ctx, `score >= 50`, map[string]any{"score": 72})
	must(err)
	fmt.Printf("EvalBool:       %v\n", ok)

	f, err := hyperex.EvalFloat(ctx, `3.5 * 2`)
	must(err)
	fmt.Printf("EvalFloat:      %v\n", f)

	s, err := hyperex.EvalString(ctx, `upper(name)`, map[string]any{"name": "hyperex"})
	must(err)
	fmt.Printf("EvalString:     %q\n", s)

	fmt.Printf("MustEval:       %v\n", hyperex.MustEval(ctx, `len([1, 2, 3])`))

	engine := hyperex.New()
	er, err := engine.Eval(ctx, `1 + 1`)
	must(err)
	fmt.Printf("Engine.Eval:    %v\n", er)
	ei, err := engine.EvalInt(ctx, `2 * 3`)
	must(err)
	fmt.Printf("Engine.EvalInt: %d\n", ei)
	fmt.Printf("Engine.MustEval:%v\n", engine.MustEval(ctx, `true && false`))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
