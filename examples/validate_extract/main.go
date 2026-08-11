// Validate expressions and extract variables / functions.
package main

import (
	"fmt"

	"github.com/aasyanov/hyperex"
)

type Env struct {
	Age  int    `expr:"age"`
	Name string `expr:"name"`
}

func main() {
	fmt.Println("=== Validate ===")
	if err := hyperex.Validate(`x + y * 2`); err != nil {
		fmt.Printf("  valid expr error: %v\n", err)
	} else {
		fmt.Println("  valid:   x + y * 2  -> ok")
	}
	if err := hyperex.Validate(`((((`); err != nil {
		fmt.Printf("  invalid: ((((  -> %v\n", err)
	} else {
		fmt.Println("  invalid unexpectedly ok")
	}

	fmt.Println("\n=== ValidateWithEnv ===")
	if err := hyperex.ValidateWithEnv(`age >= 18 && name != ""`, Env{}); err != nil {
		fmt.Printf("  typed env error: %v\n", err)
	} else {
		fmt.Println("  age >= 18 && name != \"\" with Env{} -> ok")
	}
	if err := hyperex.ValidateWithEnv(`name * 2`, Env{}); err != nil {
		fmt.Printf("  type mismatch: %v\n", err)
	} else {
		fmt.Println("  name * 2 unexpectedly ok")
	}

	fmt.Println("\n=== ExtractVariables ===")
	vars, err := hyperex.ExtractVariables(`price * quantity + tax`)
	must(err)
	fmt.Printf("  vars: %v\n", vars)

	fmt.Println("\n=== ExtractFunctions ===")
	funcs, err := hyperex.ExtractFunctions(`sum(items) + len(items)`)
	must(err)
	fmt.Printf("  funcs: %v\n", funcs)

	fmt.Println("\n=== CheckExpressionType ===")
	if err := hyperex.CheckExpressionType(`1 + 2`, hyperex.TypeInt, nil); err != nil {
		fmt.Printf("  TypeInt error: %v\n", err)
	} else {
		fmt.Println("  1 + 2 as TypeInt -> ok")
	}
	env := Env{Age: 21, Name: "Ada"}
	if err := hyperex.CheckExpressionType(`age`, hyperex.TypeInt, env); err != nil {
		fmt.Printf("  age as TypeInt error: %v\n", err)
	} else {
		fmt.Println("  age as TypeInt with Env -> ok")
	}
	if err := hyperex.CheckExpressionType(`name`, hyperex.TypeInt, env); err != nil {
		fmt.Printf("  name as TypeInt: %v\n", err)
	} else {
		fmt.Println("  name as TypeInt unexpectedly ok")
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
