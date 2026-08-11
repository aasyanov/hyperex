// Analyze and profile hyperex expressions (static + runtime).
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()
	expr := `let total = sum(items); total > threshold && len(items) > 0`

	// --- Static analysis ---
	analysis, err := hyperex.Analyze(expr)
	must(err)
	fmt.Println("=== Analyze ===")
	fmt.Printf("Variables:   %v\n", analysis.Variables)
	fmt.Printf("Functions:   %v\n", analysis.Functions)
	fmt.Printf("Operators:   %v\n", analysis.Operators)
	fmt.Printf("Complexity:  %s\n", analysis.Complexity.String())
	fmt.Printf("Depth:       %d\n", analysis.Depth)
	fmt.Printf("NodeCount:   %d\n", analysis.NodeCount)
	fmt.Printf("IsPure:      %v\n", analysis.IsPure)
	fmt.Printf("HasSideEffects: %v\n", analysis.HasSideEffects)
	fmt.Printf("HasAssignment:  %v\n", analysis.HasAssignment)
	fmt.Printf("UsesOptionalChaining: %v\n", analysis.UsesOptionalChaining)
	fmt.Printf("UsesNilCoalescing:    %v\n", analysis.UsesNilCoalescing)
	fmt.Printf("EstimatedCost:  %d\n", analysis.EstimatedCost)

	// Complexity constants conceptually via String()
	fmt.Println("\n=== Complexity classes ===")
	for _, c := range []hyperex.Complexity{
		hyperex.ComplexityConstant,
		hyperex.ComplexityLinear,
		hyperex.ComplexityLinearithmic,
		hyperex.ComplexityQuadratic,
		hyperex.ComplexityUnknown,
	} {
		fmt.Printf("  %d -> %s\n", c, c.String())
	}

	env := map[string]any{
		"items":     []any{1, 2, 3, 4},
		"threshold": 5,
	}

	// --- ProfileExpr (compile + run) ---
	profile, result, err := hyperex.ProfileExpr(ctx, expr, env)
	must(err)
	fmt.Println("\n=== ProfileExpr ===")
	fmt.Printf("Result:        %v\n", result)
	fmt.Printf("CompileTime:   %v\n", profile.CompileTime)
	fmt.Printf("ExecuteTime:   %v\n", profile.ExecuteTime)
	fmt.Printf("TotalTime:     %v\n", profile.TotalTime)
	fmt.Printf("OpsCount:      %d\n", profile.OpsCount)
	fmt.Printf("MaxStackDepth: %d\n", profile.MaxStackDepth)

	// --- Program.Analyze / Program.Profile after Compile ---
	prog, err := hyperex.Compile(expr)
	must(err)
	pa := prog.Analyze()
	fmt.Println("\n=== Program.Analyze ===")
	fmt.Printf("Variables:  %v\n", pa.Variables)
	fmt.Printf("Complexity: %s\n", pa.Complexity.String())
	fmt.Printf("IsPure:     %v\n", pa.IsPure)

	pp, presult, err := prog.Profile(ctx, env)
	must(err)
	fmt.Println("\n=== Program.Profile ===")
	fmt.Printf("Result:      %v\n", presult)
	fmt.Printf("ExecuteTime: %v\n", pp.ExecuteTime)
	fmt.Printf("OpsCount:    %d\n", pp.OpsCount)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
