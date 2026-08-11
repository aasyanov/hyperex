// Package main demonstrates struct environments and expr tags.
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

type Order struct {
	Price    float64 `expr:"price"`
	Quantity int     `expr:"qty"`
	Secret   string  `expr:"-"` // skipped — not visible to expressions
	Note     string  // no tag → field name "Note"
}

func main() {
	ctx := context.Background()
	engine := hyperex.New()

	order := Order{
		Price:    12.5,
		Quantity: 4,
		Secret:   "do-not-leak",
		Note:     "rush",
	}

	result, err := engine.Eval(ctx, `price * qty`, order)
	must(err)
	fmt.Printf("Engine.Eval:    %v\n", result)

	program, err := engine.Compile(`price * qty`)
	must(err)
	result, err = program.Eval(ctx, order)
	must(err)
	fmt.Printf("Program.Eval:   %v\n", result)

	// Secret is tagged "-" so it is not in the env.
	_, err = engine.Eval(ctx, `Secret`, order)
	fmt.Printf("Secret access:  err=%v\n", err)

	// Untagged exported fields use the Go field name.
	note, err := engine.Eval(ctx, `Note`, order)
	must(err)
	fmt.Printf("Note field:     %v\n", note)

	// WithEnv wires a sample struct for typed compile checks, then Eval still takes a value.
	typed := hyperex.New(hyperex.WithEnv(Order{}))
	result, err = typed.Eval(ctx, `price * qty`, order)
	must(err)
	fmt.Printf("WithEnv Eval:   %v\n", result)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
