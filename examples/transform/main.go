// String transforms plus opaque Node tooling (Parse / Walk / ASTToString).
package main

import (
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
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

	fmt.Println("\n=== Tokenize (Studio highlight stream) ===")
	toks, err := hyperex.Tokenize(`let x = price >= 10 // comment`)
	must(err)
	for _, t := range toks {
		if t.Type == hyperex.TokenEOF {
			continue
		}
		fmt.Printf("  %-8s lit=%q line=%d col=%d\n", t.Type, t.Lit, t.Line, t.Column)
	}

	fmt.Println("\n=== Parse / Walk / ASTToString ===")
	// Concrete AST types are unexported. Parse returns an opaque Node for tooling.
	src := `price * qty + fee`
	node, err := hyperex.Parse(src)
	must(err)
	fmt.Printf("  ASTToString: %s\n", hyperex.ASTToString(node))

	var nodes int
	hyperex.Walk(node, func(_ hyperex.Node) bool {
		nodes++
		return true
	})
	fmt.Printf("  Walk node count: %d\n", nodes)

	// Transform replaces subtrees; identity pass shown here.
	same := hyperex.Transform(node, func(n hyperex.Node) hyperex.Node { return n })
	fmt.Printf("  Transform(identity): %s\n", hyperex.ASTToString(same))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
