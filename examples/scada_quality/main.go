// Package main demonstrates map-shaped quality bindings {v,q}
// with Secure sandbox and isGood / v helpers (SCADA / PDM-style).
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()
	env := map[string]any{
		"temp": map[string]any{
			"v": 72.5,
			"q": float64(0), // Good (QualityClassMask bits == 0)
		},
		"backup": map[string]any{
			"v": 70.0,
			"q": float64(hyperex.QualityClassUncertain),
		},
	}

	eng := hyperex.NewSecureCalc()

	ok, err := eng.EvalBool(ctx, `isGood(temp)`, env)
	must(err)
	fmt.Printf("isGood(temp):           %v\n", ok)

	val, err := eng.Eval(ctx, `v(temp) + 1`, env)
	must(err)
	fmt.Printf("v(temp) + 1:            %v\n", val)

	// Prefer primary when Good; otherwise use backup value.
	chosen, err := eng.Eval(ctx, `isGood(temp) ? v(temp) : v(backup)`, env)
	must(err)
	fmt.Printf("chosen (Good path):     %v\n", chosen)

	// Bare qcode channel beside a plain numeric value.
	envDual := map[string]any{
		"press":   1.2,
		"press_q": float64(0),
	}
	ok, err = eng.EvalBool(ctx, `isGood(press_q) && press > 1`, envDual)
	must(err)
	fmt.Printf("bare qcode isGood:      %v\n", ok)

	// Domain helpers under Secure: WithSafeFunction taxes MaxOps (default tax).
	// WithSafeFunctionMeta sets an explicit MaxOwnOps budget for the callback.
	auto := hyperex.New(append(hyperex.PresetSecureAuto(),
		hyperex.WithSafeFunction("tagGood", func(q uint32) bool {
			return hyperex.QualityClass(q) == hyperex.QualityClassGood
		}),
		hyperex.WithSafeFunctionMeta("clampQ", func(q uint32) uint32 {
			if q > 3 {
				return 3
			}
			return q
		}, hyperex.SafeFuncMeta{MaxOwnOps: 8}),
	)...)
	ok, err = auto.EvalBool(ctx, `tagGood(0) && isGood(temp)`, env)
	must(err)
	fmt.Printf("SafeFunction tagGood:   %v\n", ok)
	cq, err := auto.Eval(ctx, `clampQ(9)`, env)
	must(err)
	fmt.Printf("SafeFunctionMeta clampQ: %v\n", cq)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
