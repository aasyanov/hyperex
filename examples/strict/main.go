// Package main demonstrates WithStrictMode conversion behavior.
package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()
	strict := hyperex.New(hyperex.WithStrictMode(true))
	loose := hyperex.New(hyperex.WithStrictMode(false))

	compareInt := func(expr string) {
		_, serr := strict.EvalInt(ctx, expr)
		n, lerr := loose.EvalInt(ctx, expr)
		fmt.Printf("EvalInt(%s)\n", expr)
		fmt.Printf("  strict: err=%v  is ErrInvalidConversion=%v\n", serr, errors.Is(serr, hyperex.ErrInvalidConversion))
		fmt.Printf("  loose:  %d  err=%v\n", n, lerr)
	}

	compareBool := func(expr string) {
		_, serr := strict.EvalBool(ctx, expr)
		ok, lerr := loose.EvalBool(ctx, expr)
		fmt.Printf("EvalBool(%s)\n", expr)
		fmt.Printf("  strict: err=%v  is ErrInvalidConversion=%v\n", serr, errors.Is(serr, hyperex.ErrInvalidConversion))
		fmt.Printf("  loose:  %v  err=%v\n", ok, lerr)
	}

	compareString := func(expr string) {
		_, serr := strict.EvalString(ctx, expr)
		s, lerr := loose.EvalString(ctx, expr)
		fmt.Printf("EvalString(%s)\n", expr)
		fmt.Printf("  strict: err=%v  is ErrInvalidConversion=%v\n", serr, errors.Is(serr, hyperex.ErrInvalidConversion))
		fmt.Printf("  loose:  %q  err=%v\n", s, lerr)
	}

	compareFloat := func(expr string) {
		_, serr := strict.EvalFloat(ctx, expr)
		f, lerr := loose.EvalFloat(ctx, expr)
		fmt.Printf("EvalFloat(%s)\n", expr)
		fmt.Printf("  strict: err=%v  is ErrInvalidConversion=%v\n", serr, errors.Is(serr, hyperex.ErrInvalidConversion))
		fmt.Printf("  loose:  %v  err=%v\n", f, lerr)
	}

	compareInt(`"not-a-number"`)
	compareBool(`42`)
	compareString(`1 + 2`)
	compareFloat(`"abc"`)

	prog := strict.MustCompile(`"hello"`)
	_, err := prog.EvalInt(ctx, nil)
	fmt.Printf("Program.EvalInt(strict)\n")
	fmt.Printf("  err=%v  is ErrInvalidConversion=%v\n", err, errors.Is(err, hyperex.ErrInvalidConversion))
}
