// Package main demonstrates all/any/none/one truthiness and predicate forms.
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Truthiness forms ===")
	printBool(ctx, `all([true, 1, "ok"])`)
	printBool(ctx, `any([false, 0, ""])`)
	printBool(ctx, `none([false, 0, ""])`)
	printBool(ctx, `one([false, true, false])`)
	printBool(ctx, `all([])`)
	printBool(ctx, `any([])`)

	fmt.Println("\n=== Predicate forms ===")
	printBool(ctx, `all([2, 4, 6], x => x % 2 == 0)`)
	printBool(ctx, `any([1, 3, 5], x => x > 4)`)
	printBool(ctx, `none([1, 2, 3], x => x < 0)`)
	printBool(ctx, `one([1, 2, 3], x => x == 2)`)
	printBool(ctx, `all([10, 20], (x, i) => x > i)`)

	fmt.Println("\n=== Pipe ===")
	env := map[string]any{
		"nums": []any{2, 4, 6, 8},
		"users": []any{
			map[string]any{"name": "Ann", "active": true, "age": 22},
			map[string]any{"name": "Bob", "active": true, "age": 31},
		},
	}
	printBoolEnv(ctx, `nums | all(x => x % 2 == 0)`, env)
	printBoolEnv(ctx, `nums | one(x => x == 6)`, env)
	printBoolEnv(ctx, `users | all(u => u.active)`, env)
	printBoolEnv(ctx, `users | any(u => u.age >= 30)`, env)

	fmt.Println("\n=== Empty + predicate ===")
	printBool(ctx, `all([], x => false)`)
	printBool(ctx, `any([], x => true)`)
	printBool(ctx, `none([], x => true)`)
	printBool(ctx, `one([], x => true)`)
}

func printBool(ctx context.Context, expr string) {
	v, err := hyperex.EvalBool(ctx, expr)
	must(err)
	fmt.Printf("%-45s => %v\n", expr, v)
}

func printBoolEnv(ctx context.Context, expr string, env map[string]any) {
	v, err := hyperex.EvalBool(ctx, expr, env)
	must(err)
	fmt.Printf("%-45s => %v\n", expr, v)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
