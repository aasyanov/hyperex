// Package main demonstrates Engine.Stats counters.
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()
	engine := hyperex.New()

	fmt.Printf("fresh:     %+v\n", engine.Stats())

	_, _ = engine.Compile(`1 + 1`)
	_, _ = engine.Compile(`x * 2`)
	_, _ = engine.Compile(`{{{`) // compile failure still increments Compiles
	fmt.Printf("compiles:  %+v\n", engine.Stats())

	_, err := engine.Eval(ctx, `2 + 2`)
	must(err)
	_, err = engine.Eval(ctx, `10 * 3`)
	must(err)
	_, err = engine.Eval(ctx, `missing_var`) // expected failure
	if err == nil {
		panic("expected eval error")
	}

	s := engine.Stats()
	fmt.Printf("final:     Compiles=%d Evals=%d EvalErrors=%d\n", s.Compiles, s.Evals, s.EvalErrors)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
