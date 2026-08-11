# HyperEx — expression evaluator for Go

[![CI](https://github.com/aasyanov/hyperex/actions/workflows/ci.yml/badge.svg)](https://github.com/aasyanov/hyperex/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/aasyanov/hyperex.svg)](https://pkg.go.dev/github.com/aasyanov/hyperex)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Changelog](https://img.shields.io/badge/Changelog-CHANGELOG-blue)](CHANGELOG.md)

Compile-once / eval-many expression engine: Pratt parse → optimize → bytecode VM. Stdlib-only. Go **1.24+**. Current release: **[v1.0.0](CHANGELOG.md)**.

```
go get github.com/aasyanov/hyperex
```

## What HyperEx is

| Property | Meaning |
|----------|---------|
| Expression language | One expression through EOF — not a scripting runtime |
| Compile + VM | `Compile` / `MustCompile` → reusable `Program` |
| Context-first Eval | Every Eval surface takes `context.Context` first |
| Best-effort sandbox | Ops / memory / timeout / allow-deny lists; Secure mode coerces env |
| Panic-safe library path | Expression faults return errors; no panic to callers |
| Zero production deps | Module `github.com/aasyanov/hyperex` |

## What HyperEx is not

| Not this | Use instead / note |
|----------|--------------------|
| Full programming language | No packages, imports, FS, or network builtins |
| Process / OS / crypto isolation | Secure mode is **best-effort** |
| Template engine or SQL dialect | Not a compatibility layer for other languages |
| Batch API | No `BatchEval` — loop `Program.Eval` |
| Safe with arbitrary Go callbacks | `WithFunction` escapes the sandbox |

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()

	result, err := hyperex.Eval(ctx, "2 + 2 * 3")
	if err != nil {
		panic(err)
	}
	fmt.Println(result) // 8

	program := hyperex.MustCompile("price * quantity")
	total, err := program.Eval(ctx, map[string]any{
		"price":    99.99,
		"quantity": 3,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(total) // 299.97
}
```

### Compile once, evaluate many

```go
engine := hyperex.New()
program, err := engine.Compile(`x > threshold && status == "active"`)
if err != nil {
	panic(err)
}

ok, err := program.EvalBool(ctx, map[string]any{
	"x":         150,
	"threshold": 100,
	"status":    "active",
})
```

### Custom functions and sandbox

```go
engine := hyperex.New(
	hyperex.WithFunction("double", func(x int) int { return x * 2 }),
	hyperex.WithSandbox(hyperex.SandboxConfig{
		MaxOps:  10_000,
		Timeout: time.Millisecond * 50,
	}),
)
```

Sandbox options are **Engine** `Option`s, never `CompileOption`.

## Mental model

| Surface | Role |
|---------|------|
| `Engine` | Holds options, functions, sandbox; `Compile` / `Eval` / `Stats` |
| `Program` | Compiled bytecode; `Eval(ctx, env)` with same env rules as Engine |
| `Value` | Runtime value — construct only via exported constructors |
| `Option` / `CompileOption` | Engine construction vs compile-time hints (`AsBool`, `Unoptimized`, …) |
| Env | `nil`, `map[string]any`, other string-key maps, struct / `*struct` (`expr` tags) |

**Public façade:** `Engine`, `Program`, `Value`, options, errors, analytics/transforms (`Analyze`, `Simplify`, `Extract*`, `Walk`, `Transform`, `Node`, `TypeChecker`, …).

**Not public:** lexer, parser, bytecode, compiler, VM, optimizer, concrete AST node types. No public `Parse`.

## Critical contracts

| Document | Role |
|----------|------|
| [LANGUAGE.md](LANGUAGE.md) | Normative syntax and semantics (wins over README on conflicts) |
| [AUDIT.md](AUDIT.md) | Threat model, numeric edges, sandbox boundaries, Gate M |
| [CHANGELOG.md](CHANGELOG.md) | User-visible history |
| [doc.go](doc.go) | Package GoDoc |

> Secure sandbox ≠ process isolation. Prefer AUDIT / LANGUAGE over marketing claims when they conflict.

## Examples

```bash
cd D:\SoftwareMy-GO\hyperex   # or your clone root
go run ./examples/playground/ # http://localhost:9090
go run ./examples/basic/
go build ./examples/...
```

Coverage matrix: [examples/README.md](examples/README.md).

## Quality

```powershell
go test -race -count=1 ./...
go vet ./...
golangci-lint run ./...
go test -run='^$' -bench=. -benchmem -count=5 ./...
go test -fuzz=FuzzEval_NoPanic_NoHang -fuzztime=30s
```

CI runs lint, OS×Go race tests with coverage ≥90%, fuzz discover, and main-only benches. See [CHANGELOG.md](CHANGELOG.md) and [AUDIT.md](AUDIT.md) §8–§9.

Perf claims belong with `-benchmem` numbers from a fresh run — do not treat stale README tables as contracts.

## License

MIT — Copyright (c) 2026 Anton Syanov. See [LICENSE](LICENSE).
