// Secure calc presets with time.Time / time.Duration in the env
// (SCADA / PDM-style DateTime formulas without kindAny).
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()
	now := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	env := map[string]any{
		"now":   now,
		"shift": 8 * time.Hour,
	}

	eng := hyperex.NewSecureCalc()

	hour, err := eng.Eval(ctx, `timeHour(now)`, env)
	must(err)
	fmt.Printf("timeHour(now):          %v\n", hour)

	shifted, err := eng.Eval(ctx, `timeHour(now + shift)`, env)
	must(err)
	fmt.Printf("timeHour(now + shift):  %v\n", shifted)

	delta, err := eng.Eval(ctx, `durSeconds((now + shift) - now)`, env)
	must(err)
	fmt.Printf("durSeconds(delta):      %v\n", delta)

	// AUTO preset denies print / fromJSON / toJSON.
	auto := hyperex.NewSecureAuto()
	ok, err := auto.EvalBool(ctx, `timeHour(now) >= 8 && timeHour(now) < 17`, env)
	must(err)
	fmt.Printf("shift window (Auto):    %v\n", ok)

	_, err = auto.Eval(ctx, `print("x")`)
	fmt.Printf("print denied (Auto):    %v\n", err != nil)

	// Catalog helpers for formula editors / allow-lists.
	fmt.Printf("BuiltinNames count:     %d\n", len(hyperex.BuiltinNames()))
	fmt.Printf("IsBuiltin(\"timeHour\"):  %v\n", hyperex.IsBuiltin("timeHour"))
	fmt.Printf("IsBuiltin(\"nope\"):      %v\n", hyperex.IsBuiltin("nope"))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
