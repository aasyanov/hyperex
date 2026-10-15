// One-shot Eval helpers: package-level vs Engine-owned.
// Prefer Compile + Program.Eval when the same expression runs many times
// (see compile_reuse / scada_hotpath).
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()

	env := map[string]any{
		"price":    12.5,
		"qty":      4,
		"discount": 0.1,
		"score":    72,
		"name":     "hyperex",
	}

	fmt.Println("=== Package-level (default engine) ===")
	// Convenient for ad-hoc checks; each call compiles internally.
	total, err := hyperex.Eval(ctx, `price * qty * (1 - discount)`, env)
	must(err)
	fmt.Printf("  Eval:       %v\n", total)

	ok, err := hyperex.EvalBool(ctx, `score >= 50`, env)
	must(err)
	fmt.Printf("  EvalBool:   %v\n", ok)

	n, err := hyperex.EvalInt(ctx, `qty * 10`, env)
	must(err)
	fmt.Printf("  EvalInt:    %d\n", n)

	f, err := hyperex.EvalFloat(ctx, `price * qty`, env)
	must(err)
	fmt.Printf("  EvalFloat:  %v\n", f)

	s, err := hyperex.EvalString(ctx, `upper(name)`, env)
	must(err)
	fmt.Printf("  EvalString: %q\n", s)

	// Must* panics on error by contract — use only when failure is a bug.
	fmt.Printf("  MustEval:   %v\n", hyperex.MustEval(ctx, `len([1, 2, 3])`))

	fmt.Println("\n=== Engine-owned one-shot ===")
	// Same Eval* surface, but options (sandbox, functions, numeric mode, …)
	// live on the Engine for the lifetime of the process.
	eng := hyperex.New()
	gate, err := eng.EvalBool(ctx, `score >= 50 && qty > 0`, env)
	must(err)
	fmt.Printf("  Engine.EvalBool: %v\n", gate)
	fmt.Printf("  Engine.MustEval: %v\n", eng.MustEval(ctx, `price + qty`, env))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
