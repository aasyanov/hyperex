// Type checking via TypeChecker builders and string-level APIs.
package main

import (
	"fmt"

	"github.com/aasyanov/hyperex"
)

type Order struct {
	Qty   int     `expr:"qty"`
	Price float64 `expr:"price"`
	SKU   string  `expr:"sku"`
}

func main() {
	// Prefer string-level CheckExpressionType and engine WithTypeCheck / Compile.
	// Node-level TypeChecker.Check / Infer require a hyperex.Node; concrete AST
	// node types are unexported, so ValidateWithEnv / WithTypeCheck Compile are
	// the practical public path.

	fmt.Println("=== NewTypeChecker builders ===")
	tc := hyperex.NewTypeChecker().
		WithVar("x", hyperex.TypeInt).
		WithFunc("double", hyperex.FuncSig{
			Params:  []hyperex.Type{hyperex.TypeInt},
			Returns: hyperex.TypeInt,
		}).
		WithStructEnv(Order{}).
		WithEnvType("bonus", 0).
		WithEnvTypes(map[string]any{
			"flag": true,
			"tag":  "x",
		})
	_ = tc // configured for illustration; Check/Infer need Node
	fmt.Println("  WithVar / WithFunc / WithStructEnv / WithEnvType / WithEnvTypes configured")

	fmt.Println("\n=== CheckExpressionType ===")
	if err := hyperex.CheckExpressionType(`qty * 2`, hyperex.TypeInt, Order{}); err != nil {
		fmt.Printf("  qty * 2 as TypeInt: %v\n", err)
	} else {
		fmt.Println("  qty * 2 as TypeInt -> ok")
	}
	if err := hyperex.CheckExpressionType(`sku`, hyperex.TypeInt, Order{}); err != nil {
		fmt.Printf("  sku as TypeInt: %v\n", err)
	} else {
		fmt.Println("  sku as TypeInt unexpectedly ok")
	}

	fmt.Println("\n=== ValidateWithEnv ===")
	if err := hyperex.ValidateWithEnv(`qty + price`, Order{}); err != nil {
		fmt.Printf("  qty + price: %v\n", err)
	} else {
		fmt.Println("  qty + price -> ok (numeric promotion)")
	}

	fmt.Println("\n=== Engine WithTypeCheck / Compile ===")
	eng := hyperex.New(hyperex.WithTypeCheck(Order{}))
	if _, err := eng.Compile(`qty > 0 && sku != ""`); err != nil {
		fmt.Printf("  compile ok-path error: %v\n", err)
	} else {
		fmt.Println("  Compile(qty > 0 && sku != \"\") -> ok")
	}
	if _, err := eng.Compile(`sku * qty`); err != nil {
		fmt.Printf("  Compile type error: %v\n", err)
	} else {
		fmt.Println("  Compile(sku * qty) unexpectedly ok")
	}
}
