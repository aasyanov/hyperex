// Package main demonstrates custom functions and operator overloads.
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()

	engine := hyperex.New(
		hyperex.WithFunction("double", func(x int) int { return x * 2 }),
		hyperex.WithFunctions(map[string]any{
			"add": func(a, b int) int { return a + b },
			"mul": func(a, b int) int { return a * b },
		}),
	)

	result, err := engine.Eval(ctx, `double(21)`)
	must(err)
	fmt.Printf("WithFunction:   %v\n", result)

	result, err = engine.Eval(ctx, `add(3, mul(4, 5))`)
	must(err)
	fmt.Printf("WithFunctions:  %v\n", result)

	engine.AddFunction("triple", func(x int) int { return x * 3 })
	result, err = engine.Eval(ctx, `triple(7)`)
	must(err)
	fmt.Printf("AddFunction:    %v\n", result)

	// Overload "+" with a custom array concat function.
	concatEngine := hyperex.New(
		hyperex.WithFunction("concat", func(a, b []any) []any {
			out := make([]any, 0, len(a)+len(b))
			return append(append(out, a...), b...)
		}),
		hyperex.WithOperator("+", "concat"),
	)
	result, err = concatEngine.Eval(ctx, `[1, 2] + [3, 4]`)
	must(err)
	fmt.Printf("WithOperator:   %v\n", result)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
