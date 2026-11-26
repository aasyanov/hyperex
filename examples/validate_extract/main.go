// Static tooling: Validate / Extract* / TypeChecker / CheckExpressionType.
// Concrete AST types are unexported — obtain Node via Parse, then Check / Infer.
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
		fmt.Println("  name * 2 -> ok (string repetition)")
	}

	fmt.Println("\n=== ExtractVariables / ExtractFunctions ===")
	vars, err := hyperex.ExtractVariables(`price * quantity + tax`)
	must(err)
	fmt.Printf("  vars:  %v\n", vars)
	funcs, err := hyperex.ExtractFunctions(`sum(items) + len(items)`)
	must(err)
	fmt.Printf("  funcs: %v\n", funcs)

	// Member roots collapse a.b / c.d to binding names a, c — useful for env forms.
	roots, err := hyperex.ExtractVariablesOpts(`a.b + c.d`, hyperex.ExtractMemberRoots())
	must(err)
	full, err := hyperex.ExtractVariables(`a.b + c.d`)
	must(err)
	fmt.Printf("  ExtractMemberRoots: %v  (default ExtractVariables: %v)\n", roots, full)

	fmt.Println("\n=== CheckExpressionType ===")
	if checkErr := hyperex.CheckExpressionType(`1 + 2`, hyperex.TypeInt, nil); checkErr != nil {
		fmt.Printf("  1+2 as TypeInt: %v\n", checkErr)
	} else {
		fmt.Println("  1 + 2 as TypeInt -> ok")
	}
	env := Env{Age: 21, Name: "Ada"}
	if checkErr := hyperex.CheckExpressionType(`age`, hyperex.TypeInt, env); checkErr != nil {
		fmt.Printf("  age as TypeInt: %v\n", checkErr)
	} else {
		fmt.Println("  age as TypeInt with Env -> ok")
	}
	if checkErr := hyperex.CheckExpressionType(`name`, hyperex.TypeInt, env); checkErr != nil {
		fmt.Printf("  name as TypeInt: %v\n", checkErr)
	} else {
		fmt.Println("  name as TypeInt unexpectedly ok")
	}

	fmt.Println("\n=== TypeChecker via Parse (Node) ===")
	tc := hyperex.NewTypeChecker().
		WithVar("x", hyperex.TypeInt).
		WithFunc("double", hyperex.FuncSig{
			Params:  []hyperex.Type{hyperex.TypeInt},
			Returns: hyperex.TypeInt,
		}).
		WithStructEnv(Env{}).
		WithEnvTypes(map[string]any{"flag": true})

	node, err := hyperex.Parse(`double(x) + age`)
	must(err)
	if errs := tc.Check(node); errs != nil && len(errs.Errors) > 0 {
		fmt.Printf("  Check error: %v\n", errs)
	} else {
		fmt.Printf("  Check ok  Infer=%s\n", tc.Infer(node))
	}

	// Best-effort checker: operand mismatches surface reliably via Check.
	bad, err := hyperex.Parse(`name * age`)
	must(err)
	if errs := tc.Check(bad); errs != nil && len(errs.Errors) > 0 {
		fmt.Printf("  Check type error: %v\n", errs)
	} else {
		fmt.Println("  name * age -> ok (string repetition)")
	}

	fmt.Println("\n=== Engine WithTypeCheck / Compile ===")
	eng := hyperex.New(hyperex.WithTypeCheck(Env{}))
	if _, err := eng.Compile(`age > 0 && name != ""`); err != nil {
		fmt.Printf("  compile ok-path error: %v\n", err)
	} else {
		fmt.Println("  Compile(age > 0 && name != \"\") -> ok")
	}
	if _, err := eng.Compile(`name * age`); err != nil {
		fmt.Printf("  Compile type error: %v\n", err)
	} else {
		fmt.Println("  Compile(name * age) -> ok (string repetition)")
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
