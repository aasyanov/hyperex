// Package hyperex provides a high-performance expression evaluation engine for Go.
//
// Import path: github.com/aasyanov/hyperex
//
// The public surface is Engine / Program / Value / Options, error types, and
// transform/analytics helpers (Analyze, Simplify, ExtractVariables, Walk,
// Transform, Node, TypeChecker, …), plus Studio DX façades Tokenize / Parse.
// Lexer, parser, bytecode, compiler, optimizer, and the VM (execVM) are
// package-private; concrete AST node types are unexported.
//
// Expression language syntax and semantics are defined in LANGUAGE.md (normative),
// including numeric edges and sandbox / Secure / SafeFunction host options (§3.5, §11).
//
// HyperEx compiles expressions to bytecode and runs them on an optimized vm.
// Design goals include aggressive compile-time optimization, specialized opcodes,
// and pooled evaluation contexts to keep repeated Eval cheap.
//
// # Basic Usage
//
// The simplest way to use HyperEx is through the Eval function for one-off evaluations.
// All Eval surfaces take context.Context as the first argument:
//
//	result, err := hyperex.Eval(context.Background(), `2 + 2 * 2`)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result) // Output: 6
//
// For repeated evaluations, compile once and reuse the Program:
//
//	engine := hyperex.New()
//	program, err := engine.Compile(`x + y * 2`)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	ctx := context.Background()
//	result1, _ := program.Eval(ctx, map[string]any{"x": 10, "y": 5}) // 20
//	result2, _ := program.Eval(ctx, map[string]any{"x": 3, "y": 7})  // 17
//
// # Environments
//
// [Program.Eval] and [Engine.Eval] accept the same env shapes (converted via internal toEnvMap):
//   - nil
//   - map[string]any (as-is)
//   - other maps with string keys
//   - structs / *structs (exported fields; `expr` tag or field name; `-` skips)
//
// Prefer map[string]any on hot paths; struct env avoids manual map construction when convenience matters.
//
// Example with a struct env:
//
//	type Env struct {
//	    X int     `expr:"x"`
//	    Y float64 `expr:"y"`
//	}
//
//	engine := hyperex.New(hyperex.WithTypeCheck(Env{})) // enables type checking
//	result, _ := engine.Eval(ctx, `x + y`, Env{X: 10, Y: 5.5})
//	result, _ = program.Eval(ctx, Env{X: 10, Y: 5.5}) // same rules on Program.Eval
//
// # Context Cancellation and Timeouts
//
// ctx cancellation during vm execution returns an error matching [ErrOperationCancelled].
// Sandbox [WithTimeout] and ctx deadlines both apply — whichever fires first wins.
//
//	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
//	defer cancel()
//	result, err := program.Eval(ctx, env)
//
// # Custom Functions
//
// Extend the expression language with custom functions via Engine options:
//
//	engine := hyperex.New(
//	    hyperex.WithFunction("double", func(x int) int { return x * 2 }),
//	    hyperex.WithFunction("greet", func(name string) string { return "Hello, " + name }),
//	)
//
//	result, _ := engine.Eval(ctx, `double(21)`)     // 42
//	result, _ := engine.Eval(ctx, `greet("World")`) // "Hello, World"
//
// [WithFunction] is a trusted escape hatch (no forced MaxOps tax). Prefer
// [WithSafeFunction] for domain helpers that should count against MaxOps.
// For static type checking of custom function signatures, use
// [TypeChecker.WithFunc] (not an Engine option).
//
// # Strict Mode
//
// [WithStrictMode] makes typed Eval* helpers ([EvalBool], [EvalInt], [EvalFloat], [EvalString])
// return [ErrInvalidConversion] instead of silently coercing mismatched results.
//
// # Engine Stats
//
// [Engine.Stats] returns a lock-free snapshot of cumulative counters since Engine construction:
// Compiles, Evals, and EvalErrors.
//
//	stats := engine.Stats()
//	log.Printf("evals=%d errors=%d", stats.Evals, stats.EvalErrors)
//
// # Operators
//
// HyperEx supports a rich set of operators:
//
//	Arithmetic:  +, -, *, /, %, ** (power)
//	Comparison:  ==, !=, <, <=, >, >=
//	Logical:     &&, ||, !
//	Bitwise:     &, |, ^, <<, >>
//	Membership:  in, not in
//	Ternary:     condition ? then : else
//	Nil-coalesce: a ?? b
//	Optional:    a?.b, a?.[index]
//
// # Built-in Functions
//
// Common functions are available out of the box:
//
//	Math:     abs, min, max, ceil, floor, round, sqrt, pow
//	Strings:  len, upper, lower, trim, split, join, contains, startsWith, endsWith
//	Arrays:   len, first, last, take, skip, reverse, sort, unique
//	Aggregation: sum, avg, count, all, any, none, one
//	Transform:   map, filter, reduce, find, findIndex
//
// # Thread Safety
//
// Engine instances and compiled programs are safe for concurrent use.
// Each Eval call uses its own evaluation context from a sync.Pool.
//
// # Sandbox Notes
//
// Sandbox (including Secure mode) is best-effort protection against untrusted expressions:
// it enforces resource limits and restricts access to external Go objects in Secure mode,
// but it is not a process-level security boundary. User-provided Go functions registered
// via [WithFunction] can perform arbitrary work and escape the sandbox;
// [WithSafeFunction] still uses reflection but charges an ops tax per call.
//
// Sandbox (including Secure) is best-effort, not process isolation — see LANGUAGE.md §11.
//
// # Numeric Edge Cases
//
// Some builtins intentionally return dynamic types to preserve correctness:
//   - ceil/floor/round normally return int64, but return float64 for NaN/Inf or values out of int64 range.
//   - fromJSON converts JSON numbers to int64 only for IEEE-754 safe integers (|n| <= 2^53-1).
//
// # Performance Tips
//
//  1. Compile expressions once, evaluate many times via [Program.Eval]
//  2. Prefer map[string]any for hot paths (no per-call struct reflection)
//  3. Use struct env when convenience outweighs conversion cost
//  4. Enable constant folding (on by default) for expressions with literals
//  5. Use [WithOperator] sparingly for domain-specific operator overloads
package hyperex
