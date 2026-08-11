// String-level transform helpers: Simplify, Expand, Inline.
package main

import (
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	// Note: ASTToString / Walk / Transform / AnalyzeAST take hyperex.Node.
	// Concrete AST node types are unexported, so string-level APIs
	// (Simplify, Expand, Inline, Analyze, …) are the public path.

	fmt.Println("=== Simplify ===")
	for _, expr := range []string{
		"2 + 3 * 4",
		"x * 2 + 0",
		"x * 1",
		"(a + b) - (a + b)",
	} {
		out, err := hyperex.Simplify(expr)
		must(err)
		fmt.Printf("  %q -> %q\n", expr, out)
	}

	fmt.Println("\n=== Expand (macros) ===")
	macros := map[string]string{
		"ADULT": "age >= 18",
		"USA":   `country == "US"`,
	}
	expanded, err := hyperex.Expand("ADULT && USA", macros)
	must(err)
	fmt.Printf("  macros: %v\n", macros)
	fmt.Printf("  %q -> %q\n", "ADULT && USA", expanded)

	fmt.Println("\n=== Inline ===")
	inlined, err := hyperex.Inline("let x = 5; x + x")
	must(err)
	fmt.Printf("  %q -> %q\n", "let x = 5; x + x", inlined)

	nested, err := hyperex.Inline("let a = 1; let b = a + 2; a + b")
	must(err)
	fmt.Printf("  %q -> %q\n", "let a = 1; let b = a + 2; a + b", nested)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
