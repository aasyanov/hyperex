package hyperex_test

import (
	"context"
	"fmt"
	"time"

	"github.com/aasyanov/hyperex"
)

func ExampleEval() {
	result, err := hyperex.Eval(context.Background(), `2 + 2 * 3`)
	if err != nil {
		panic(err)
	}
	fmt.Println(result)

	result, err = hyperex.Eval(context.Background(), `price * quantity`, map[string]any{
		"price":    10,
		"quantity": 3,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
	// Output:
	// 8
	// 30
}

func ExampleCompile() {
	program, err := hyperex.Compile(`x + y * 2`)
	if err != nil {
		panic(err)
	}

	result, err := program.Eval(context.Background(), map[string]any{"x": 10, "y": 5})
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
	// Output:
	// 20
}

func ExampleNew_withFunction() {
	engine := hyperex.New(
		hyperex.WithFunction("double", func(x int) int { return x * 2 }),
	)
	result, err := engine.Eval(context.Background(), `double(21)`)
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
	// Output:
	// 42
}

func ExampleNew_withSandbox() {
	engine := hyperex.New(
		hyperex.WithSandbox(hyperex.SandboxConfig{
			MaxOps:  100_000,
			Timeout: time.Second,
		}),
	)
	result, err := engine.Eval(context.Background(), `1 + 2`)
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
	// Output:
	// 3
}

func ExampleNew_withSecureSandbox() {
	engine := hyperex.New(hyperex.WithSecureSandbox())
	result, err := engine.Eval(context.Background(), `upper("hi")`)
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
	// Output:
	// HI
}

func ExampleExtractVariables() {
	vars, err := hyperex.ExtractVariables(`a + b * c`)
	if err != nil {
		panic(err)
	}
	fmt.Println(vars)
	// Output:
	// [a b c]
}

func ExampleAnalyze() {
	analysis, err := hyperex.Analyze(`x + len(items)`)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(analysis.Variables) > 0)
	fmt.Println(len(analysis.Functions) > 0)
	// Output:
	// true
	// true
}

func ExampleEval_allWithPredicate() {
	ok, err := hyperex.EvalBool(context.Background(), `all([2, 4, 6], x => x % 2 == 0)`)
	if err != nil {
		panic(err)
	}
	fmt.Println(ok)

	ok, err = hyperex.EvalBool(context.Background(), `[1, 2, 3] | one(x => x == 2)`)
	if err != nil {
		panic(err)
	}
	fmt.Println(ok)
	// Output:
	// true
	// true
}
