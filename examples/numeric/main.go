// NumericMode and NonFinite policy — SCADA / float-first engines.
package main

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== NumericDefault (v1.0 int-preserving) ===")
	def := hyperex.New(hyperex.WithNumericMode(hyperex.NumericDefault))
	v, err := def.Eval(ctx, `1 / 2`)
	must(err)
	fmt.Printf("  1/2 => %v (%T)  // integer division\n", v, v)

	sum, err := def.Eval(ctx, `1 + 2`)
	must(err)
	fmt.Printf("  1+2 => %v (%T)\n", sum, sum)

	fmt.Println("\n=== NumericFloat64 (float-first arithmetic) ===")
	flt := hyperex.New(hyperex.WithNumericMode(hyperex.NumericFloat64))
	v, err = flt.Eval(ctx, `1 / 2`)
	must(err)
	fmt.Printf("  1/2 => %v (%T)  // float64 promotion\n", v, v)

	// String + and time/duration ops stay unchanged under NumericFloat64.
	s, err := flt.Eval(ctx, `"a" + "b"`)
	must(err)
	fmt.Printf("  \"a\"+\"b\" => %v  // concat unchanged\n", s)

	fmt.Println("\n=== NonFiniteAllow vs NonFiniteError ===")
	allow := hyperex.New(
		hyperex.WithNumericMode(hyperex.NumericFloat64),
		hyperex.WithNonFinite(hyperex.NonFiniteAllow),
	)
	nan, err := allow.Eval(ctx, `sqrt(-1)`)
	must(err)
	fmt.Printf("  Allow  sqrt(-1) => %v  IsNaN=%v\n", nan, math.IsNaN(nan.(float64)))

	strictNF := hyperex.New(
		hyperex.WithNumericMode(hyperex.NumericFloat64),
		hyperex.WithNonFinite(hyperex.NonFiniteError),
	)
	_, err = strictNF.Eval(ctx, `sqrt(-1)`)
	fmt.Printf("  Error  sqrt(-1) => errors.Is(ErrNonFinite)=%v  err=%v\n",
		errors.Is(err, hyperex.ErrNonFinite), err)

	// NewSecureCalc / NewSecureAuto already set NumericFloat64 + NonFiniteError.
	// Arithmetic 0/0 stays ErrDivisionByZero; NaN/Inf paths use ErrNonFinite.
	secure := hyperex.NewSecureCalc()
	_, err = secure.Eval(ctx, `sqrt(-1)`)
	fmt.Printf("  SecureCalc sqrt(-1) => errors.Is(ErrNonFinite)=%v\n",
		errors.Is(err, hyperex.ErrNonFinite))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
