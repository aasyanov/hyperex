// Package main demonstrates sandbox limits and restriction sentinels.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()

	safe, err := hyperex.New(
		hyperex.WithSandbox(hyperex.SandboxConfig{
			MaxOps:  100_000,
			Timeout: time.Second,
		}),
	).Eval(ctx, `1 + 2`)
	must(err)
	fmt.Printf("WithSandbox:           %v\n", safe)

	secure, err := hyperex.New(hyperex.WithSecureSandbox()).Eval(ctx, `upper("hi")`)
	must(err)
	fmt.Printf("WithSecureSandbox:     %v\n", secure)

	_, err = hyperex.New(hyperex.WithMaxOps(50)).Eval(ctx, `map(range(0, 10000), # + 1)`)
	printSentinel("WithMaxOps", err)

	_, err = hyperex.New(
		hyperex.WithTimeout(time.Nanosecond),
		hyperex.WithMaxOps(1_000_000_000),
	).Eval(ctx, `map(range(0, 100000), # + 1)`)
	printSentinel("WithTimeout", err)

	_, err = hyperex.New(hyperex.WithAllowedFuncs("len", "sum")).Eval(ctx, `upper("x")`)
	printSentinel("WithAllowedFuncs", err)

	_, err = hyperex.New(hyperex.WithDeniedFuncs("len")).Eval(ctx, `len([1, 2, 3])`)
	printSentinel("WithDeniedFuncs", err)

	_, err = hyperex.New(hyperex.WithReadOnly(true)).Eval(ctx, `let x = 1; x`)
	printSentinel("WithReadOnly", err)

	// Engine call-depth limit (pairs with SandboxConfig.MaxCallDepth).
	_, err = hyperex.New(hyperex.WithMaxCallDepth(5)).Eval(ctx, `
		let f1 = x => x + 1;
		let f2 = x => f1(x) + 1;
		let f3 = x => f2(x) + 1;
		let f4 = x => f3(x) + 1;
		let f5 = x => f4(x) + 1;
		let f6 = x => f5(x) + 1;
		f6(0)
	`)
	printSentinel("WithMaxCallDepth", err)

	// Rich SandboxConfig fields (MaxMemory / DisableBuiltins / stack).
	cfgEng := hyperex.New(hyperex.WithSandbox(hyperex.SandboxConfig{
		MaxOps:          10_000,
		MaxMemory:       1 << 20,
		MaxStackDepth:   256,
		MaxCallDepth:    64,
		DisableBuiltins: false,
		DeniedFuncs:     []string{"print"},
	}))
	v, err := cfgEng.Eval(ctx, `sum([1, 2, 3])`)
	must(err)
	fmt.Printf("SandboxConfig fields:  %v\n", v)
}

func printSentinel(label string, err error) {
	if err == nil {
		fmt.Printf("%-22s unexpected success\n", label+":")
		return
	}
	matched := "none"
	switch {
	case errors.Is(err, hyperex.ErrOpsLimitExceeded):
		matched = "ErrOpsLimitExceeded"
	case errors.Is(err, hyperex.ErrTimeout):
		matched = "ErrTimeout"
	case errors.Is(err, hyperex.ErrFunctionNotAllowed):
		matched = "ErrFunctionNotAllowed"
	case errors.Is(err, hyperex.ErrAssignmentNotAllowed):
		matched = "ErrAssignmentNotAllowed"
	case errors.Is(err, hyperex.ErrOperationCancelled):
		matched = "ErrOperationCancelled"
	case errors.Is(err, hyperex.ErrMaxCallDepth):
		matched = "ErrMaxCallDepth"
	case errors.Is(err, hyperex.ErrStackOverflow):
		matched = "ErrStackOverflow"
	}
	fmt.Printf("%-22s matched=%s  err=%v\n", label+":", matched, err)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
