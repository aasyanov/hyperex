// Package main demonstrates the SCADA / PDM hot-path pattern:
// compile once, reuse Program, reuse a pre-sized map[string]any env,
// prefer EvalValue+TryFloat (or EvalFloat), with NumericFloat64 via Secure preset.
//
// No BatchEval / EnvPool public API — reuse is host-side only.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()

	// SecureCalc includes NumericFloat64 + NonFiniteError + Secure sandbox.
	eng := hyperex.NewSecureCalc()

	// 1) Compile once at Apply / config load.
	prog := eng.MustCompile(`(pv - sp) * gain + bias`)

	// 2) Pre-sized env; clear/reuse keys across samples (no new map each tick).
	env := make(map[string]any, 4)

	samples := []struct {
		pv, sp, gain, bias float64
	}{
		{100.0, 95.0, 1.5, 0.0},
		{98.2, 95.0, 1.5, 0.1},
		{101.0, 95.0, 1.2, -0.2},
	}

	for i, s := range samples {
		env["pv"] = s.pv
		env["sp"] = s.sp
		env["gain"] = s.gain
		env["bias"] = s.bias

		// 3) Prefer EvalValue + TryFloat to avoid Interface() boxing,
		//    or EvalFloat when you only need float64.
		v, err := prog.EvalValue(ctx, env)
		must(err)
		out, err := v.TryFloat()
		must(err)
		fmt.Printf("sample %d EvalValue+TryFloat: %.4f\n", i, out)

		f, err := prog.EvalFloat(ctx, env)
		must(err)
		fmt.Printf("sample %d EvalFloat:          %.4f\n", i, f)
	}

	// Optional host-local "pool": clear keys between unrelated formulas.
	// (Not a library API — pattern only.)
	clearEnv(env)
	env["now"] = time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	hourProg := eng.MustCompile(`timeHour(now)`)
	hour, err := hourProg.EvalFloat(ctx, env)
	must(err)
	fmt.Printf("timeHour(now):              %.0f\n", hour)

	fmt.Println("tips: map[string]any hot path; avoid struct env / lambdas per sample")
}

func clearEnv(m map[string]any) {
	for k := range m {
		delete(m, k)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
