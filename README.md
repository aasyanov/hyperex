# hyperex — Expression Evaluator for Go

[![CI](https://github.com/aasyanov/hyperex/actions/workflows/ci.yml/badge.svg)](https://github.com/aasyanov/hyperex/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/aasyanov/hyperex.svg)](https://pkg.go.dev/github.com/aasyanov/hyperex)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Changelog](https://img.shields.io/badge/Changelog-CHANGELOG-blue)](CHANGELOG.md)

Compile-once / eval-many expression engine for Go **1.24+**: Pratt parse → optimize → bytecode VM. Stdlib-only. Current release: **[v1.0.0](CHANGELOG.md)**.

```
go get github.com/aasyanov/hyperex
```

**Changelog:** [CHANGELOG.md](CHANGELOG.md)

## What it is / what it is not

**It is** a single-expression language embedded in Go: you compile a string to a reusable `Program`, then evaluate it many times against maps or structs. Eval is context-first, panic-safe on the library path, and optionally sandboxed (ops / memory / timeout / allow-deny lists). Secure mode coerces env toward JSON-like values.

**It is not** a scripting runtime, template engine, SQL dialect, or process isolator. There are no packages, imports, filesystem, or network builtins. There is no `BatchEval` — loop `Program.Eval`. `WithFunction` Go callbacks are a full trust escape hatch and leave the sandbox.

| It is | It is not |
|---|---|
| Expression through EOF | Full programming language |
| `Compile` → bytecode `Program` | Interpreter-only toy |
| Best-effort sandbox budgets | OS / crypto process isolation |
| Stdlib-only module | Plugin host with transitive deps |

## The Problem

Services repeatedly need **small, configurable decision logic**: pricing formulas, feature gates, validation predicates, routing rules, scoring, and data transforms. Teams either hard-code Go (redeploy for every change), embed a heavy scripting language (startup cost, attack surface, GC noise), or re-parse the same string on every request (latency and allocs).

The recurring requirements are:

1. **Author expressions as data** — store and reload rules without shipping new binaries for every tweak.
2. **Pay parse cost once** — compile at load / config change, evaluate per event.
3. **Stay predictable** — context cancellation, typed helpers, sentinel errors, no panic into callers.
4. **Bound untrusted input** — ops/memory/time limits when expressions come from operators or tenants.
5. **Stay embeddable** — flat Go API, zero production dependencies, concurrent `Engine` / `Program` reuse.

hyperex targets that niche: one expression, one compile, many evals on a specialized VM — not a general-purpose language.

## Architecture

```
expression string
       │
       ▼
┌──────────────┐    ┌─────────────┐    ┌──────────────┐    ┌────────────┐
│ Lexer/Parser │───►│  Optimizer  │───►│   Compiler   │───►│  Program   │
│  (Pratt AST) │    │ fold/reduce │    │  bytecode    │    │ (immutable)│
└──────────────┘    └─────────────┘    └──────────────┘    └─────┬──────┘
                                                                  │
                     ctx + env (map / struct)                      │
                                                                  ▼
                                                           ┌────────────┐
                                                           │  execVM    │◄── sync.Pool
                                                           │  Eval*     │
                                                           └────────────┘
```

1. **Compile path** — tokenize, Pratt-parse to an internal AST, optionally optimize (constant folding, strength reduction), emit bytecode into a `Program`. Lexer, parser, optimizer, compiler, and VM stay package-private.
2. **Eval path** — each `Program.Eval` borrows a pooled VM context, binds env, runs opcodes, returns `any` or a typed helper result. Concurrent evals share the same `Program`.
3. **Engine** — holds options (functions, sandbox, type-check hints, strict mode) and offers both `Compile` and one-shot `Eval`. Package-level `Eval` / `MustCompile` use a default engine.
4. **Analytics / transforms** — `Analyze`, `Simplify`, `Extract*`, `Walk`, `Transform`, `TypeChecker` operate on strings or opaque `Node` values without exposing concrete AST types.

## How it works

### Compile once, evaluate many

`Engine.Compile` / `MustCompile` produce a `Program`. Hot paths call `program.Eval(ctx, env)` (or `EvalBool` / `EvalInt` / `EvalFloat` / `EvalString`). Re-compiling on every request throws away the design: benches show ~10× (simple) and ~8× (complex) wins for compiled vs eval-once on Linux CI.

Optimizer is on by default. `Unoptimized()` is a `CompileOption` for A/B or debugging. Constant folding can turn literal-heavy expressions into near-constant eval (see Optimizer Effectiveness benches).

### Environments

`Program.Eval` and `Engine.Eval` accept the same env shapes:

- `nil`
- `map[string]any` (preferred on hot paths)
- other maps with string keys
- structs / `*structs` — exported fields; `expr` tag or field name; `-` skips

Struct env is convenient; map env avoids per-call reflection. Prefer maps when latency matters.

### Context and sandbox

Every Eval surface takes `context.Context` first. Cancellation returns an error matching `ErrOperationCancelled`. Sandbox `WithTimeout` and ctx deadlines both apply — whichever fires first wins.

Sandbox is an **Engine** `Option` (`WithSandbox`, `WithSecureSandbox`, `WithMaxOps`, …), never a `CompileOption`. It budgets ops, memory, stack, and optionally allow/deny builtins. Secure mode coerces env and blocks unsafe member access. It is **not** process isolation — see [AUDIT.md](AUDIT.md).

### Language surface (summary)

Normative syntax and semantics: **[LANGUAGE.md](LANGUAGE.md)** (wins over README on conflicts).

| Area | Examples |
|---|---|
| Arithmetic | `+ - * / % **` |
| Comparison / logical | `== != < <= > >=`, `&& \|\| !` |
| Bitwise / membership | `& \| ^ << >>`, `in` / `not in` |
| Control-ish | ternary `? :`, coalesce `??`, `let`, optional `?.` |
| Collections | arrays, maps, index/slice, range `..`, pipe `\|`, lambdas, spread |
| Builtins | math, strings, arrays, aggregation, `map`/`filter`/`reduce`, JSON helpers |

Pipe vs bitwise `|` is a common pitfall: `ident | ident` pipes when the RHS name matches a default builtin; integer/float literals never pipe. Details in LANGUAGE.md §9.

### Public vs private

**Public:** `Engine`, `Program`, `Value` (construct via exported constructors only), `Option` / `CompileOption`, errors, analytics/transforms (`Analyze`, `Simplify`, `Extract*`, `Walk`, `Transform`, `Node`, `TypeChecker`, …).

**Not public:** lexer, parser, bytecode, compiler, VM, optimizer, concrete AST node types. No public `Parse` — use string APIs (`Analyze`, `Simplify`, `CheckExpressionType`, …).

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()

	// One-shot
	result, err := hyperex.Eval(ctx, "2 + 2 * 3")
	if err != nil {
		panic(err)
	}
	fmt.Println(result) // 8

	// Compile once, evaluate many
	program := hyperex.MustCompile("price * quantity")
	total, err := program.Eval(ctx, map[string]any{
		"price":    99.99,
		"quantity": 3,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(total) // 299.97

	engine := hyperex.New(
		hyperex.WithFunction("double", func(x int) int { return x * 2 }),
		hyperex.WithSandbox(hyperex.SandboxConfig{
			MaxOps:  10_000,
			Timeout: 50 * time.Millisecond,
		}),
	)
	pred, err := engine.Compile(`x > threshold && status == "active"`)
	if err != nil {
		panic(err)
	}
	ok, err := pred.EvalBool(ctx, map[string]any{
		"x": 150, "threshold": 100, "status": "active",
	})
	fmt.Println(ok, err) // true <nil>
}
```

Interactive REPL and cookbook recipes:

```bash
go run ./examples/playground/   # http://localhost:9090
go run ./examples/basic/
go build ./examples/...
```

Coverage matrix: [examples/README.md](examples/README.md).

## API / contracts

Full GoDoc: [pkg.go.dev/github.com/aasyanov/hyperex](https://pkg.go.dev/github.com/aasyanov/hyperex). Threat model and numeric edges: [AUDIT.md](AUDIT.md). Language: [LANGUAGE.md](LANGUAGE.md).

### Engine

| Method | Description |
|---|---|
| `New(opts...)` | Construct engine with `Option`s |
| `Compile(expr, opts...)` | Parse → optimize → bytecode `Program` |
| `MustCompile` | Panic on compile error |
| `Eval` / `MustEval` | One-shot compile+eval (prefer `Program` on hot paths) |
| `EvalBool` / `EvalInt` / `EvalFloat` / `EvalString` | Typed one-shot helpers |
| `AddFunction` | Register a Go callback after construction |
| `Stats` | Lock-free snapshot: `Compiles`, `Evals`, `EvalErrors` |

### Program

| Method | Description |
|---|---|
| `Eval(ctx, env)` | Evaluate against env (`any` — same shapes as Engine) |
| `EvalBool` / `EvalInt` / `EvalFloat` / `EvalString` | Typed results; strict mode affects coercion |
| `Source` | Original expression string |
| `Analyze` / `Profile` | Per-program analytics |

`Engine` and `Program` are safe for concurrent use. Each Eval uses a pooled evaluation context.

### Option vs CompileOption

| Kind | Examples | When |
|---|---|---|
| `Option` (engine) | `WithFunction`, `WithSandbox`, `WithSecureSandbox`, `WithEnv`, `WithTypeCheck`, `WithStrictMode`, `WithOptimize`, … | Construction / engine policy |
| `CompileOption` | `AsBool`, `AsInt`, `AsFloat`, `AsString`, `Unoptimized` | Per-compile hints only |

Sandbox knobs are never `CompileOption`s.

### Value

Exported for performance. Construct only via `NewValue`, `Int`, `Float`, `String`, `Bool`, `Array`, `Map`, `Func`. Hand-built inconsistent `Value` values are undefined behavior.

### Analytics / transforms

| API | Role |
|---|---|
| `Analyze` / `AnalyzeAST` | Variables, functions, complexity, warnings |
| `Simplify` / `Expand` / `Inline` | Source-level rewrites |
| `ExtractVariables` / `ExtractFunctions` | Name extraction |
| `Validate` / `ValidateWithEnv` | Structural / env checks |
| `CheckExpressionType` / `TypeChecker` | Best-effort typing |
| `Walk` / `Transform` | Opaque `Node` traversal |
| `ProfileExpr` / `Program.Profile` | Per-opcode timing under Eval |

### Errors

Prefer `errors.Is` / `errors.As`. Runtime failures wrap as `*RuntimeError`; sandbox failures as `*SandboxError`. Common sentinels include `ErrDivisionByZero`, `ErrIndexOutOfBounds`, `ErrOperationCancelled`, `ErrInvalidConversion`, `ErrMaxCallDepth`, and related type/call errors. See `errors.go` and AUDIT.md §3.4.

### Critical documents

| Document | Role |
|---|---|
| [LANGUAGE.md](LANGUAGE.md) | Normative syntax and semantics |
| [AUDIT.md](AUDIT.md) | Threat model, numeric edges, sandbox boundaries, Gate M |
| [CHANGELOG.md](CHANGELOG.md) | User-visible history |
| [doc.go](doc.go) | Package GoDoc |

> Secure sandbox ≠ process isolation. Prefer AUDIT / LANGUAGE over marketing claims when they conflict.

## Benchmarks

Two GitHub Actions CI environments, **same CPU SKU** (`AMD EPYC 9V74`, 4 vCPU), `-count=3`. Latency cells are **medians**. `B/op` / `allocs/op` come from the Linux median run (allocation shape is deterministic across OS). Bold marks the faster CI median per row (±3% ties stay unbolded).

These are CI snapshots, not contracts — re-run `-benchmem` before quoting a release claim.

### Environments

| | CI Server (Linux) | CI Server (Windows) |
|---|---|---|
| CPU | AMD EPYC 9V74, 4 vCPU | AMD EPYC 9V74, 4 vCPU |
| OS / FS | Ubuntu (ext4) | Windows Server 2022 (NTFS) |
| Go | 1.26 | 1.26 |
| GOMAXPROCS | 4 | 4 |
| Runs | 3 (`-count=3`) | 3 (`-count=3`) |

Primary comparison axis: **Linux vs Windows on matched EPYC hosts** (no SKU mix like some sibling packages). Hot Eval is almost OS-neutral; compile / analyze / large-env paths show clearer OS gaps.

### Compile

| Benchmark | Linux | Windows | B/op | allocs/op |
|---|---|---|---|---|
| Compile Simple | **1.75 µs** | 2.87 µs | 5576 | 16 |
| Compile Medium | **3.82 µs** | 5.07 µs | 6344 | 33 |
| Compile Complex | **7.34 µs** | 10.29 µs | 7912 | 71 |
| Compile VeryComplex | **9.50 µs** | 13.37 µs | 11568 | 99 |
| Compile +optimizer | **3.35 µs** | 4.61 µs | 5960 | 34 |
| Compile no-optimizer | **3.72 µs** | 5.25 µs | 5904 | 27 |
| Compile +typecheck | **2.09 µs** | 3.46 µs | 5568 | 19 |
| Compile no-typecheck | **1.80 µs** | 2.97 µs | 5312 | 17 |

### Hot Eval

| Benchmark | Linux | Windows | B/op | allocs/op |
|---|---|---|---|---|
| Eval IntAdd | 223.6 ns | 228.4 ns | 8 | 1 |
| Eval IntSub | 212.1 ns | 208.6 ns | 0 | 0 |
| Eval Arithmetic Complex | **517.5 ns** | 542.5 ns | 8 | 1 |
| Eval Comparison EQ | 205.6 ns | 206.5 ns | 0 | 0 |
| Eval Logical And | 192.6 ns | 195.5 ns | 0 | 0 |
| Eval ShortCircuit And | 136.5 ns | 137.1 ns | 0 | 0 |
| Eval Bitwise And | **199.5 ns** | 206.4 ns | 0 | 0 |
| Eval Ternary True | 169.7 ns | 170.3 ns | 0 | 0 |
| Eval Coalesce First | 199.0 ns | 204.0 ns | 0 | 0 |
| Eval Map Member | 176.4 ns | 174.4 ns | 0 | 0 |
| Eval OptionalChain Hit | 175.4 ns | 173.9 ns | 0 | 0 |
| Eval Let | **192.3 ns** | 215.9 ns | 0 | 0 |
| Eval Array Access | 223.9 ns | 222.6 ns | 24 | 1 |
| Eval String Concat | **545.0 ns** | 720.6 ns | 128 | 8 |
| Eval Regex Simple | 323.0 ns | 321.4 ns | 16 | 1 |
| Eval Builtin Len | **267.5 ns** | 321.2 ns | 56 | 2 |
| Eval Builtin Sum | 1.17 µs | 1.15 µs | 64 | 3 |
| Eval CustomFunc 1-arg | **500.1 ns** | 577.5 ns | 64 | 3 |
| Eval Env Small | 216.3 ns | 212.6 ns | 0 | 0 |
| Eval Env Large | **120.93 µs** | 199.71 µs | 226092 | 22 |
| Eval Env Struct | **2.58 µs** | 4.18 µs | 5661 | 19 |
| Eval Lambda Simple | **595.9 ns** | 782.9 ns | 164 | 5 |
| Builtin Map Lambda | **47.49 µs** | 58.11 µs | 9480 | 408 |
| Eval Pipe Simple | 1.16 µs | 1.16 µs | 64 | 3 |
| Eval Destructure Object | **493.2 ns** | 510.2 ns | 0 | 0 |

### Compile-once vs Eval-once

| Benchmark | Linux | Windows | B/op | allocs/op |
|---|---|---|---|---|
| Compare Simple Eval-once | **2.17 µs** | 3.50 µs | 5324 | 17 |
| Compare Simple Compiled | **205.5 ns** | 212.1 ns | 0 | 0 |
| Compare Complex Eval-once | **6.45 µs** | 9.78 µs | 7199 | 54 |
| Compare Complex Compiled | 823.5 ns | **674.4 ns** | 8 | 1 |

### Parallel / scenarios

| Benchmark | Linux | Windows | B/op | allocs/op |
|---|---|---|---|---|
| Parallel Simple | 103.6 ns | 103.3 ns | 0 | 0 |
| Parallel Complex | **242.4 ns** | 263.3 ns | 8 | 1 |
| Parallel Builtin | **505.8 ns** | 549.9 ns | 64 | 3 |
| Parallel CustomFunc | **251.5 ns** | 282.0 ns | 64 | 3 |
| Parallel VMPool | **890.5 ns** | 1.00 µs | 96 | 4 |
| Scenario BusinessRule | 637.6 ns | **370.1 ns** | 0 | 0 |
| Scenario Pricing | 780.0 ns | **470.4 ns** | 8 | 1 |
| Scenario Validation | **803.2 ns** | 940.9 ns | 160 | 5 |
| Scenario DataTransform | **1.11 µs** | 1.50 µs | 440 | 10 |
| Scenario FilterMapReduce | **108.48 µs** | 127.55 µs | 20809 | 924 |
| Scenario Pipeline | **29.77 µs** | 33.24 µs | 5267 | 246 |

### Optimizer, sandbox, memory, helpers

| Benchmark | Linux | Windows | B/op | allocs/op |
|---|---|---|---|---|
| Optimizer Effect +opt | 69.22 ns | **66.10 ns** | 0 | 0 |
| Optimizer Effect no-opt | 328.8 ns | 329.3 ns | 0 | 0 |
| Eval +sandbox | **313.6 ns** | 438.6 ns | 80 | 1 |
| Eval no-sandbox | **229.6 ns** | 266.9 ns | 0 | 0 |
| Eval Sandbox OpsCheck | **421.0 ns** | 541.4 ns | 80 | 1 |
| Memory NoAlloc | 209.9 ns | 207.8 ns | 0 | 0 |
| Memory StringAlloc | **551.3 ns** | 692.8 ns | 128 | 8 |
| TypeChecker Simple | 22.35 ns | 22.03 ns | 0 | 0 |
| Analyze Simple | **1.40 µs** | 2.29 µs | 2024 | 15 |

### Analysis

**Matched SKU makes OS gaps readable.** Both runners report `AMD EPYC 9V74` at GOMAXPROCS=4. Unlike sibling packages that landed on mixed Xeon/EPYC hosts, Linux↔Windows deltas here are mostly OS/runtime, not CPU class.

**Hot Eval is OS-neutral.** Integer/logical/map/optional-chain paths sit in a ~0.98–1.05× band (IntSub ≈210 ns, ShortCircuit And ≈137 ns, Map Member ≈175 ns, Memory NoAlloc ≈210 ns). Prefer compiled `Program.Eval` for steady-state work; do not expect Windows to be “slower at expressions” on these floors.

**Compile / Analyze pay an OS tax.** Simple compile 1.75→2.87 µs (~1.6×), Analyze 1.40→2.29 µs (~1.6×). Lexer/compiler allocate (Windows `mem.prof`: `newCompiler` / `newLexer` among top `alloc_space`). Compile once and reuse — do not re-`Compile` per request.

**Compile-once wins by ~10× on simple and ~8× on complex (Linux).** Simple Eval-once 2.17 µs vs Compiled 205.5 ns; Complex Eval-once 6.45 µs vs Compiled 823.5 ns. That is the product thesis: amortize parse/optimize/codegen across many evals.

**Optimizer folds constants hard.** Effectiveness bench: 69 ns with optimizer vs 329 ns without (~4.7×) at 0 allocs. Compile-time cost of enabling the optimizer is small vs no-optimizer (3.35 vs 3.72 µs Linux).

**Sandbox is measurable, not free.** Linux: no-sandbox 230 ns → +sandbox 314 ns (~37%); OpsCheck 421 ns. Windows sandbox overhead is larger in absolute ns. Use sandbox when untrusted expressions need budgets; skip it for trusted internal rules if latency matters.

**Allocating paths diverge more by OS.** String Concat / Memory StringAlloc ~1.3× slower on Windows; Env Large / Env Struct ~1.6×; Builtin Len and CustomFunc show mid-teens percent gaps. Zero-alloc numeric/logical work does not.

**Higher-order / lambda work is allocation-bound.** Map-lambda ~47–58 µs with ~408 allocs/op; FilterMapReduce scenario ~108–128 µs with ~924 allocs. Prefer builtins without per-element closures when the hot path is a tight loop.

**Windows pprof (CPU):** `(*execVM).execute` / `(*Program).Eval` / `(*execVM).run` dominate cumulative time; `NewValue` and map iteration show up on allocating benches. Matches the “VM dispatch + Value traffic” mental model.

**Scatter caution.** Some scenario / compare rows (e.g. Linux Pricing, Complex Compiled) move tens of percent across the three runs — medians smooth that, but do not over-interpret a single bold cell as a stable OS win.

### What the numbers mean

These benches measure engine overhead (compile, VM dispatch, builtins, sandbox), not end-to-end business latency. A real rule that touches I/O or a large Go callback will dwarf a 200 ns Eval.

**What HyperEx is fast at:** compiled numeric/logical/map member paths with 0–1 allocs; parallel Eval of shared `Program`s; constant-folded expressions after Optimize.

**Where cost concentrates:** first compile, large env materialization, string/array builders, lambda higher-order builtins, sandbox ops accounting.

## Quality

```powershell
go test -race -count=1 ./...
go vet ./...
golangci-lint run ./...
go test -run='^$' -bench=. -benchmem -count=5 ./...
go test -fuzz=FuzzEval_NoPanic_NoHang -fuzztime=30s
```

CI runs lint, OS×Go race tests with coverage ≥90% on the library package, fuzz discover, and main-only benches with pprof artifacts. See [CHANGELOG.md](CHANGELOG.md), [QUALITY.md](QUALITY.md), and [AUDIT.md](AUDIT.md) §8–§9.

## License

MIT — Copyright (c) 2026 Anton Syanov. See [LICENSE](LICENSE).
