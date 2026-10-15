# hyperex — Expression Evaluator for Go

![CI](https://github.com/aasyanov/hyperex/actions/workflows/ci.yml/badge.svg)
![Go Reference](https://pkg.go.dev/badge/github.com/aasyanov/hyperex.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![Changelog](https://img.shields.io/badge/Changelog-CHANGELOG-blue)

Compile-once / eval-many expression engine for Go **1.24+**: Pratt parse → optimize → bytecode VM. Stdlib-only. Current release: **[v1.5.0](CHANGELOG.md#150--2026-10-15)**.

```
go get github.com/aasyanov/hyperex
```

**Changelog:** [CHANGELOG.md](CHANGELOG.md)

> [!IMPORTANT]
> `WithFunction` registers a trusted Go callback that **leaves the sandbox**. Ops/memory/timeout budgets do not constrain arbitrary work inside that callback. Under MaxOps prefer `WithSafeFunction` / `WithSafeFunctionMeta`. Secure mode is best-effort env coercion + deny lists — not process isolation.



## What it is / what it is not

**It is** a single-expression language embedded in Go: you compile a string to a reusable `Program`, then evaluate it many times against maps or structs. Eval is context-first and panic-safe on the library path (`Must`* panics by contract; host-callback panics become `*RuntimeError`). Optional sandbox budgets ops / memory / timeout / allow-deny lists. Secure mode coerces env toward JSON-like values. Studio façades (`Tokenize` / `Parse`) and analytics (`Analyze`, `Simplify`, …) support formula editors without exporting the lexer or concrete AST.

**It is not** a scripting runtime, template engine, SQL dialect, or process isolator. Neighboring jobs stay in the host: persistence, HTTP, OS policy, and multi-expression workflows. There are no packages, imports, filesystem, or network builtins. There is no `BatchEval` — loop `Program.Eval`.


| It is                          | It is not                        |
| ------------------------------ | -------------------------------- |
| Expression through EOF         | Full programming language        |
| `Compile` → bytecode `Program` | Interpreter-only toy             |
| Best-effort sandbox budgets    | OS / crypto process isolation    |
| Stdlib-only module             | Plugin host with transitive deps |




### When to use / when not to


| Use hyperex when…                                                   | Prefer something else when…                                                      |
| ------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| Rules are **data**: pricing, gates, validation, routing, scoring    | You need statements, packages, imports, or a long-lived script runtime           |
| You can **compile once** at load / config change and eval per event | Every request must invent new language surface the host cannot bound             |
| Latency budgets care about **0–1 alloc** numeric/logical floors     | The “expression” is mostly I/O or a heavy Go callback (the VM will not save you) |
| Untrusted operators need **ops / memory / time** budgets            | You need OS process isolation, seccomp, or a separate worker process             |
| Studio / lint needs opaque `Parse` / `Tokenize` / `Analyze`         | You need a full IDE language server or typed module system                       |




### Guarantee profile


| Property           | Behavior                                                                                                                                  |
| ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Compile product    | `Program` is immutable (source + bytecode + engine back-ref); share across goroutines                                                     |
| Hot Eval           | `Program.Eval` / `EvalValue` / typed helpers; prefer `EvalValue` to skip `Interface()` boxing                                             |
| Env shapes         | `nil`, `map[string]any` (preferred hot), other string-key maps, struct / `*struct` (`expr` tag or name; `-` skips)                        |
| Context            | Every Eval is `ctx`-first; cancel/deadline → `errors.Is(…, ErrOperationCancelled)`; nil ctx → `Background()`                              |
| Panic policy       | Library path recovers into `*RuntimeError`; `Must*` panics on error by contract; trusted host callbacks that panic become `*RuntimeError` |
| Sandbox            | Engine `Option` only (never `CompileOption`); budgets expression work, not arbitrary `WithFunction` bodies                                |
| Secure             | JSON-like env coerce (+ `time.Time` / `time.Duration`); deny unsafe member access on arbitrary Go objects — **not** a OS jail             |
| SafeFunction       | Host callback taxed against `MaxOps` (default tax 16); use when MaxOps must meter host work                                               |
| Lifecycle          | No `Close` — drop references; pooled VMs return to `sync.Pool` after each Eval                                                            |
| Stats              | Lock-free `Compiles` / `Evals` / `EvalErrors` snapshot via `Engine.Stats()`                                                               |
| Normative language | [LANGUAGE.md](LANGUAGE.md) wins over README on syntax/semantics conflicts                                                                 |




## The Problem

Services repeatedly need **small, configurable decision logic**: pricing formulas, feature gates, validation predicates, routing rules, scoring, and data transforms. Teams either hard-code Go (redeploy for every change), embed a heavy scripting language (startup cost, attack surface, GC noise), or re-parse the same string on every request (latency and allocs).

The recurring requirements are:

1. **Author expressions as data** — store and reload rules without shipping new binaries for every tweak.
2. **Pay parse cost once** — compile at load / config change, evaluate per event.
3. **Stay predictable** — context cancellation, typed helpers, sentinel errors, no panic into callers.
4. **Bound untrusted input** — ops/memory/time limits when expressions come from operators or tenants.
5. **Stay embeddable** — flat Go API, zero production dependencies, concurrent `Engine` / `Program` reuse.

hyperex targets that niche: one expression, one compile, many evals on a specialized VM — not a general-purpose language.

## Architectural position / mental model

- **Engine** owns policy: options, builtin catalog, optional type-check hints, sandbox budgets, numeric/non-finite modes, and lock-free `Stats`. Construct once with `New` / `NewSecureCalc` / `NewSecureAuto`; share across goroutines. It has no `Close` — there is nothing to shut down beyond dropping references (pooled VMs return to `sync.Pool` after each Eval). `AddFunction` copy-on-writes the builtin map under the engine lock.
- **Program** is an immutable compile product (source + bytecode + engine back-ref). Share it; call `Eval` / `EvalValue` / typed helpers concurrently. Prefer `EvalValue` on hot paths to avoid `Interface()` boxing. `ResultHint` reflects compile-time `As`* flags; `Source` / `Analyze` / `Profile` stay on the program.
- **Value** is the VM tagged union (documented 32 B). Construct only via package constructors (`NewValue`, `Int`, `Float`, `String`, `Bool`, `Array`, `Map`, `Func`, `Time`, `Duration`, …); treat as immutable after construction for concurrent reads. Do not hand-build fields — inconsistent combinations are undefined behavior.
- **Sandbox** is an Engine `Option` surface (`WithSandbox`, `WithSecureSandbox`, helpers). It budgets untrusted *expressions*; it does not isolate `WithFunction` callbacks or the Go process. Prefer `WithSafeFunction` when `MaxOps` must meter host work.
- **Studio / analytics** (`Tokenize`, `Parse`, `Analyze`, `Simplify`, `Walk`, `Transform`, `TypeChecker`) operate on strings or opaque `Node` values. They do not expose lexer types, opcodes, or concrete AST structs — formula editors stay on the public façade.



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
                     ctx + env (map / struct)                    │
                                                                 ▼
                                                           ┌────────────┐
                                                           │  execVM    │◄── sync.Pool
                                                           │  Eval*     │
                                                           └────────────┘
```

1. **Compile path (warm)** — tokenize, Pratt-parse to an internal AST, optionally optimize (constant folding, strength reduction), emit bytecode into a `Program`. Lexer, parser, optimizer, compiler, and VM stay package-private.
2. **Eval path (hot)** — each `Program.Eval` borrows a pooled VM context, binds env, runs opcodes, returns `any` or a typed / `Value` result. Concurrent evals share the same `Program`.
3. **Engine** — holds options (functions, sandbox, type-check hints, strict / numeric / non-finite mode) and offers both `Compile` and one-shot `Eval`. Package-level `Eval` / `MustCompile` use a default engine.
4. **Analytics / Studio (cold)** — `Analyze`, `Simplify`, `Extract`*, `Walk`, `Transform`, `TypeChecker`, `Tokenize`, `Parse` operate on strings or opaque `Node` values without exposing concrete AST types.



## How it works



### Compile once, evaluate many — **warm** compile / **hot** eval

`Engine.Compile` / `MustCompile` produce a `Program`. Hot paths call `program.Eval(ctx, env)` (or `EvalValue` / `EvalBool` / `EvalInt` / `EvalFloat` / `EvalString`). Re-compiling on every request throws away the design: benches show ~10× (simple) and ~8× (complex) wins for compiled vs eval-once on Linux CI.

Optimizer is on by default. `Unoptimized()` is a `CompileOption` for A/B or debugging. Constant folding can turn literal-heavy expressions into near-constant eval (see Optimizer Effectiveness benches).

### Environments — **hot** (maps) / **warm** (structs)

`Program.Eval` and `Engine.Eval` accept the same env shapes:

- `nil`
- `map[string]any` (preferred on hot paths)
- other maps with string keys
- structs / `*structs` — exported fields; `expr` tag or field name; `-` skips

Struct env is convenient; map env avoids per-call reflection. Prefer maps when latency matters. SCADA-style reuse: pre-size a `map[string]any`, mutate values in place, call `EvalValue` — see [examples/scada_hotpath/](examples/scada_hotpath/) and LANGUAGE.md §14.

### Context and sandbox — **hot** when enabled

Every Eval surface takes `context.Context` first. Cancellation returns an error matching `ErrOperationCancelled`. Sandbox `WithTimeout` and ctx deadlines both apply — whichever fires first wins.

Sandbox is an **Engine** `Option` (`WithSandbox`, `WithSecureSandbox`, `WithMaxOps`, …), never a `CompileOption`. It budgets ops, memory, stack, and optionally allow/deny builtins. Secure mode coerces env toward JSON-like values (plus `time.Time` / `time.Duration`) and blocks unsafe member access on arbitrary Go objects. It is **not** process isolation; `WithFunction` callbacks remain a full trust escape hatch (prefer `WithSafeFunction` under MaxOps).

Allow/deny maps are cached at `New`; each Eval only refreshes the Timeout deadline (Secure Simple bench: ~88 B/op / 2 allocs after the cache polish).

### Host functions — trust boundary


| Registration                                     | Trust                      | MaxOps                                                 |
| ------------------------------------------------ | -------------------------- | ------------------------------------------------------ |
| `WithFunction` / `WithFunctions` / `AddFunction` | Full trust escape          | Body work is **not** taxed                             |
| `WithSafeFunction` / `WithSafeFunctionMeta`      | Still Go code, but metered | Default tax 16 ops per call (`SafeFuncMeta` overrides) |


DeniedFuncs / AllowedFuncs apply by **name** to both. Under untrusted expressions + `MaxOps`, prefer SafeFunction so a hostile rule cannot hide unbounded host work behind a trusted callback name.

### Numeric / time / quality — **hot** on Eval

- `WithNumericMode` — `NumericDefault` (int-preserving arithmetic) or `NumericFloat64` (`1/2` → `0.5`). String `+` and time/duration ops are unchanged. Normative: LANGUAGE.md §3.5.
- `WithNonFinite` — `NonFiniteAllow` (default) or `NonFiniteError` (`errors.Is(…, ErrNonFinite)` when Eval would return NaN/Inf).
- **Time / duration** — first-class kinds via `Time` / `Duration` / `NewValue(time.Time|time.Duration)`; operators and builtins in LANGUAGE.md. Secure coerce accepts both. No `now()` builtin — pass wall clock in env.
- **Quality helpers** — builtins `v` / `q` / `isGood` / `isBad` / `isUncertain` / `withQ` on map `{v,q}` or bare qcode. Exported `QualityClass`* masks. **No** arithmetic auto-unwrap: write `v(temp)+1`, not `temp+1`.



### Language surface (summary)

Normative syntax and semantics: **[LANGUAGE.md](LANGUAGE.md)** (wins over README on conflicts).


| Area                 | Examples                                                                                |
| -------------------- | --------------------------------------------------------------------------------------- |
| Arithmetic           | `+ - * / % **`                                                                          |
| Comparison / logical | `== != < <= > >=`, `&& || !`                                                            |
| Bitwise / membership | `& | ^ << >>`, `in` / `not in`                                                          |
| Control-ish          | ternary `? :`, coalesce `??`, `let`, optional `?.`                                      |
| Collections          | arrays, maps, index/slice, range `..`, pipe `|`, lambdas, spread                        |
| Builtins             | math, strings, arrays, aggregation, `map`/`filter`/`reduce`, JSON helpers, time/quality |


Pipe vs bitwise `|` is a common pitfall: `ident | ident` pipes when the RHS name matches a default builtin; integer/float literals never pipe. Match `Analysis.Warnings` with `WarnAmbiguousBitwiseOr` / `WarnBuiltinNamePipe`. Details in LANGUAGE.md §9.

### Public vs private

**Public:** `Engine`, `Program`, `Value` (construct via exported constructors only), `Option` / `CompileOption`, errors, analytics/transforms (`Analyze`, `Simplify`, `Extract`*, `Walk`, `Transform`, `Node`, `TypeChecker`, …), Studio DX façades `Tokenize` / `Parse` (opaque `Node`), `BuiltinNames` / `IsBuiltin`, presets / quality / numeric helpers.

**Not public:** lexer, parser types, bytecode, compiler, VM, optimizer, concrete AST node types. Prefer string APIs (`Analyze`, `Simplify`, `CheckExpressionType`, …) unless you need opaque `Node` / token streams.

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

	// One-shot — fine for demos; prefer Compile on hot paths.
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

	// Sandbox budgets untrusted expressions (no WithFunction here — that escapes the sandbox).
	engine := hyperex.New(
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

Same sequence as `ExampleEval` / `ExampleCompile` / `ExampleNew_withSandbox` in `example_test.go`.

## Usage scenarios

1. **Trusted hot path** — `MustCompile` once at startup; `Program.Eval` / `EvalValue` per event with `map[string]any`. Skip sandbox when expressions are ship-owned.
2. **Untrusted operator rules** — `NewSecureCalc` / `NewSecureAuto` or `WithSecureSandbox` + `WithMaxOps` / `WithTimeout`; prefer `WithSafeFunction` over `WithFunction`.
3. **SCADA / PDM formula** — Secure + `NumericFloat64` + `NonFiniteError` (presets); pass `time.Time` in env; quality maps `{v,q}` with `isGood`/`v`; compile-once + env reuse + `EvalValue` ([examples/numeric/](examples/numeric/), [scada_secure/](examples/scada_secure/), [scada_quality/](examples/scada_quality/), [scada_hotpath/](examples/scada_hotpath/)).
4. **Studio / lint** — `Tokenize` / `Parse` / `Analyze` / `Simplify` / `ExtractVariables` / `ValidateWithEnv` before persisting a rule string; match pipe warnings via `Warn*` ([examples/transform/](examples/transform/), [analyze_profile/](examples/analyze_profile/), [validate_extract/](examples/validate_extract/)).
5. **Typed one-shots** — package `EvalBool` / `EvalInt` / … or Engine helpers when you do not retain a `Program` ([examples/basic/](examples/basic/)).

Interactive REPL and cookbook recipes:

```bash
go run ./examples/playground/   # http://localhost:9090
go run ./examples/basic/
go run ./examples/numeric/
go build ./examples/...
```

Coverage matrix: [examples/README.md](examples/README.md). SCADA hot-path cookbook: [LANGUAGE.md](LANGUAGE.md) §14 · [examples/scada_hotpath/](examples/scada_hotpath/).

## API / contracts

Full GoDoc: [pkg.go.dev/github.com/aasyanov/hyperex](https://pkg.go.dev/github.com/aasyanov/hyperex). Language: [LANGUAGE.md](LANGUAGE.md) (§3.5 numeric, §11 sandbox / Secure / SafeFunction).

### Engine


| Method                                              | Description                                                                               |
| --------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `New(opts...)`                                      | Construct engine with `Option`s                                                           |
| `NewSecureCalc` / `NewSecureAuto`                   | Secure presets (+ optional extra `Option`s); also `PresetSecureCalc` / `PresetSecureAuto` |
| `Compile(expr, opts...)`                            | Parse → optimize → bytecode `Program`                                                     |
| `MustCompile`                                       | Panic on compile error                                                                    |
| `Eval` / `MustEval`                                 | One-shot compile+eval (prefer `Program` on hot paths)                                     |
| `EvalBool` / `EvalInt` / `EvalFloat` / `EvalString` | Typed one-shot helpers                                                                    |
| `AddFunction`                                       | Register a trusted Go callback after construction (COW builtins)                          |
| `Stats`                                             | Lock-free snapshot: `Compiles`, `Evals`, `EvalErrors` (engine-owned counters)             |


Package-level `Compile` / `Eval*` / `Must*` use a default engine.

### Program


| Method                                              | Description                                          |
| --------------------------------------------------- | ---------------------------------------------------- |
| `Eval(ctx, env)`                                    | Evaluate against env (`any` — same shapes as Engine) |
| `EvalValue(ctx, env)`                               | Hot-path `Value` result (no `Interface()` boxing)    |
| `EvalBool` / `EvalInt` / `EvalFloat` / `EvalString` | Typed results; strict mode affects coercion          |
| `ResultHint`                                        | Compile-time As* hint flags                          |
| `Source`                                            | Original expression string                           |
| `Analyze` / `Profile`                               | Per-program analytics                                |


`Engine` and `Program` are safe for concurrent use. Each Eval uses a pooled evaluation context. Nil `ctx` is treated as `context.Background()`; prefer an explicit ctx.

### Studio / discovery


| API                                              | Role                                                         |
| ------------------------------------------------ | ------------------------------------------------------------ |
| `Tokenize` / `Token` / `TokenKind`               | Opaque highlight stream (comments included)                  |
| `Parse(expr) (Node, error)`                      | Opaque AST façade                                            |
| `BuiltinNames` / `IsBuiltin`                     | Sorted builtin name discovery (implementations stay private) |
| `WarnAmbiguousBitwiseOr` / `WarnBuiltinNamePipe` | Stable strings for `Analysis.Warnings` matching              |




### Option vs CompileOption


| Kind              | Examples                                                                                                                                                                                                                                                        | When                                                                                                 |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `Option` (engine) | `WithFunction`, `WithSafeFunction*`, `WithSandbox`, `WithSecureSandbox`, `WithTypeCheck`, `WithStrictMode`, `WithOptimize`, `WithNumericMode`, `WithNonFinite`, `WithMaxSourceBytes`, `WithOperator`, `WithReadOnly`, `WithAllowedFuncs` / `WithDeniedFuncs`, … | Construction / engine policy                                                                         |
| `CompileOption`   | `AsBool`, `AsInt`, `AsFloat`, `AsString`, `Unoptimized`                                                                                                                                                                                                         | Per-compile: `As*` force eval-time conversion on every Eval; `Unoptimized` overrides engine optimize |


Sandbox knobs are never `CompileOption`s.

### Value

Exported for performance. Construct only via `NewValue`, `Int`, `Float`, `String`, `Bool`, `Array`, `Map`, `Func`, `Time`, `Duration`, plus `Nil`. Hand-built inconsistent `Value` values are undefined behavior. After `EvalValue` success, treat the returned `Value` as immutable for concurrent reads. Use `Try*` / `Is*` accessors rather than assuming Go type assertions on `Interface()`.

### Analytics / transforms

| API | Role |
|---|---|
| `Analyze` / `AnalyzeAST` | Variables, functions, complexity, warnings |
| `Simplify` / `Expand` / `Inline` | Source-level rewrites |
| `ExtractVariables` / `ExtractVariablesOpts` / `ExtractMemberRoots` | Name extraction (default `ExtractVariables` unchanged) |
| `ExtractFunctions` | Function-name extraction |
| `Validate` / `ValidateWithEnv` | Structural / env checks |
| `CheckExpressionType` / `TypeChecker` | Best-effort typing (`WithVar` / `WithFunc` / `WithStructEnv` / `WithEnvTypes`) |
| `Walk` / `Transform` / `ASTToString` | Opaque `Node` traversal / print |
| `ProfileExpr` / `Program.Profile` | Per-opcode timing under Eval |

## Type comparison


| Decision                             | Choose                               | Avoid when…                                                            |
| ------------------------------------ | ------------------------------------ | ---------------------------------------------------------------------- |
| `Eval` vs `EvalValue`                | `EvalValue` on hot paths (no boxing) | You need a plain `any` for JSON/logging — then `Eval` or `Interface()` |
| `WithFunction` vs `WithSafeFunction` | SafeFunction under `MaxOps`          | Trusted internal callbacks with no ops budget — Function is fine       |
| Map env vs struct env                | `map[string]any` when ns matter      | Prototyping / config structs where reflection cost is acceptable       |
| `New` vs `NewSecure*`                | Secure presets for operator formulas | Ship-owned expressions that need full member access on Go objects      |
| Package `Eval` vs `Program`          | Retain `Program` after first use     | One-shot CLI / test helpers                                            |




## Configuration


| Option                                           | Default         | When                                                      |
| ------------------------------------------------ | --------------- | --------------------------------------------------------- |
| `WithOptimize` / folding / strength              | on              | Disable only for A/B or debugging                         |
| `WithMaxCallDepth`                               | 1000            | Nested lambda/call ceiling                                |
| `WithMaxSourceBytes`                             | 0 (unlimited)   | Cap untrusted source size → `ErrSourceTooLarge`           |
| `WithStrictMode`                                 | false           | Fail closed on bad conversions → `ErrInvalidConversion`   |
| `WithNumericMode` / `WithNonFinite`              | default / allow | Arithmetic / NaN-Inf policy                               |
| `WithSandbox` / `WithMaxOps` / `WithTimeout` / … | off             | Untrusted expressions                                     |
| `WithAllowedFuncs` / `WithDeniedFuncs`           | —               | Name allow/deny for builtins and host funcs               |
| `WithReadOnly`                                   | false           | Ban assignment → `ErrAssignmentNotAllowed`                |
| `WithOperator`                                   | —               | Map an operator lexeme to a function name                 |
| `WithSafeFunction*`                              | —               | Host callbacks under MaxOps tax                           |
| `WithFunction` / `WithFunctions`                 | —               | Trusted callbacks (sandbox escape)                        |
| `WithTypeCheck`                                  | off             | Compile-time typing (map env still preferred on hot Eval) |




## Observability

`Engine.Stats()` returns a lock-free snapshot (`Compiles`, `Evals`, `EvalErrors`). Counters are engine-owned; concurrent Eval/Compile is safe. Use for process metrics — not as a substitute for sandbox budgets.

## Extensibility

- Register host callbacks with `WithFunction` / `WithSafeFunction` / `AddFunction`.
- Overload operators with `WithOperator(op, fnName)`.
- Extend type-check knowledge with `TypeChecker.WithFunc` (signatures), not by inventing a second registration API.
- Discover builtins with `BuiltinNames` / `IsBuiltin` — implementations remain private.



## Pitfalls

> [!WARNING]
> Pipe vs bitwise `|`: `ident | builtinName` may parse as pipe. Integer/float RHS never pipes. See LANGUAGE.md §9 and `WarnAmbiguousBitwiseOr` / `WarnBuiltinNamePipe`.

> [!WARNING]
> Secure sandbox ≠ OS isolation. `WithFunction` escapes budgets. Prefer `WithSafeFunction` when MaxOps must meter host work.

> [!WARNING]
> Struct env costs reflection per Eval. Prefer `map[string]any` on hot paths.

> [!WARNING]
> Hand-built `Value` field combinations are undefined behavior — use constructors only.

> [!WARNING]
> Quality maps do not auto-unwrap in arithmetic: `temp+1` is wrong; use `v(temp)+1` (or `isGood(temp) ? v(temp) : …`).

> [!WARNING]
> There is no `now()` builtin. Pass `time.Time` in env (Secure coerce accepts it). Blindly trusting operator strings without `WithMaxSourceBytes` / sandbox invites pathological compile or eval cost.



## Errors

Prefer `errors.Is` / `errors.As`. Runtime failures wrap as `*RuntimeError` (unwrap to sentinels); sandbox failures as `*SandboxError`. Compile/parse failures surface as `*CompileError` / `*ParseError` / `*LexerError` (line/column when known). `RuntimeError` may carry `Offset` / `Line` / `Column` (0 = unknown).


| Error                                                         | When                              | Recovery                             |
| ------------------------------------------------------------- | --------------------------------- | ------------------------------------ |
| `ErrDivisionByZero`                                           | `/` or `%` by zero                | Fix expression or guard divisor      |
| `ErrIndexOutOfBounds`                                         | Bad array/string index            | Clamp / validate indices             |
| `ErrNotCallable` / `ErrArgumentCount`                         | Bad call                          | Fix callee / arity                   |
| `ErrInvalidType` / `ErrNotIndexable` / `ErrInvalidSliceIndex` | Wrong operand shape               | Fix types / indices                  |
| `ErrOperationCancelled`                                       | `ctx` cancel/deadline             | Retry with fresh ctx or stop         |
| `ErrTimeout`                                                  | Sandbox `WithTimeout` fired       | Raise timeout or cut work            |
| `ErrOpsLimitExceeded`                                         | `MaxOps` budget spent             | Raise `MaxOps` or simplify expr      |
| `ErrMemoryLimitExceeded`                                      | Estimated memory over `MaxMemory` | Raise limit or shrink data           |
| `ErrStackOverflow`                                            | VM stack over `MaxStackDepth`     | Raise depth or flatten expr          |
| `ErrFunctionNotAllowed`                                       | Allow/deny / secure deny          | Adjust lists or rewrite call         |
| `ErrAssignmentNotAllowed`                                     | Assignment under read-only        | Drop assignment or disable read-only |
| `ErrMaxCallDepth`                                             | Nested calls over engine depth    | Raise `WithMaxCallDepth`             |
| `ErrInvalidConversion`                                        | Strict-mode coercion fail         | Fix types or disable strict          |
| `ErrNonFinite`                                                | NaN/Inf under `NonFiniteError`    | Guard math / change policy           |
| `ErrSourceTooLarge`                                           | Source over `WithMaxSourceBytes`  | Truncate or raise cap                |
| `ErrInvalidRegex`                                             | Bad `~=` pattern                  | Fix pattern                          |
| Other `Err*`                                                  | Type/call taxonomy sentinels      | See GoDoc                            |




## Safety and concurrency

- `Engine` / `Program`: safe for concurrent use; each Eval borrows a pooled VM.
- `AddFunction`: copy-on-write of the builtin map under the engine lock.
- `Value`: not safe for concurrent mutation; share only immutable results.
- No `Close`: drop references when done; pool entries are recycled automatically.
- Ownership after success: returned `any` / `Value` belongs to the caller; the library does not retain Eval results.



## Benchmarks

Two GitHub Actions CI environments, **same CPU SKU** (`AMD EPYC 9V74`, 4 vCPU), `-count=3`. Latency cells are **medians**. `B/op` / `allocs/op` come from the Linux median run (allocation shape is deterministic across OS). Bold marks the faster CI median per row (±3% ties stay unbolded).

These are CI snapshots, not contracts — re-run `-benchmem` before quoting a release claim.

### Environments


|            | CI Server (Linux)     | CI Server (Windows)        |
| ---------- | --------------------- | -------------------------- |
| CPU        | AMD EPYC 9V74, 4 vCPU | AMD EPYC 9V74, 4 vCPU      |
| OS / FS    | Ubuntu (ext4)         | Windows Server 2022 (NTFS) |
| Go         | 1.26                  | 1.26                       |
| GOMAXPROCS | 4                     | 4                          |
| Runs       | 3 (`-count=3`)        | 3 (`-count=3`)             |


Primary comparison axis: **Linux vs Windows on matched EPYC hosts** (same SKU on both OS runners). Hot Eval is almost OS-neutral; compile / analyze / large-env paths show clearer OS gaps.

Class legend (when a Class column is absent): Compile* / Analyze / TypeChecker / Simplify = **warm/cold**; Hot Eval / Parallel / Scenario / Memory* = **hot**.

### Compile


| Benchmark            | Linux       | Windows  | B/op  | allocs/op |
| -------------------- | ----------- | -------- | ----- | --------- |
| Compile Simple       | **1.75 µs** | 2.87 µs  | 5576  | 16        |
| Compile Medium       | **3.82 µs** | 5.07 µs  | 6344  | 33        |
| Compile Complex      | **7.34 µs** | 10.29 µs | 7912  | 71        |
| Compile VeryComplex  | **9.50 µs** | 13.37 µs | 11568 | 99        |
| Compile +optimizer   | **3.35 µs** | 4.61 µs  | 5960  | 34        |
| Compile no-optimizer | **3.72 µs** | 5.25 µs  | 5904  | 27        |
| Compile +typecheck   | **2.09 µs** | 3.46 µs  | 5568  | 19        |
| Compile no-typecheck | **1.80 µs** | 2.97 µs  | 5312  | 17        |




### Hot Eval

Class: **hot** = steady-state `Program.Eval`; **warm** = compile/front-end; **cold** = analytics/studio.


| Benchmark               | Class | Linux         | Windows   | B/op   | allocs/op |
| ----------------------- | ----- | ------------- | --------- | ------ | --------- |
| Eval IntAdd             | hot   | 223.6 ns      | 228.4 ns  | 8      | 1         |
| Eval IntSub             | hot   | 212.1 ns      | 208.6 ns  | 0      | 0         |
| Eval Arithmetic Complex | hot   | **517.5 ns**  | 542.5 ns  | 8      | 1         |
| Eval Comparison EQ      | hot   | 205.6 ns      | 206.5 ns  | 0      | 0         |
| Eval Logical And        | hot   | 192.6 ns      | 195.5 ns  | 0      | 0         |
| Eval ShortCircuit And   | hot   | 136.5 ns      | 137.1 ns  | 0      | 0         |
| Eval Bitwise And        | hot   | **199.5 ns**  | 206.4 ns  | 0      | 0         |
| Eval Ternary True       | hot   | 169.7 ns      | 170.3 ns  | 0      | 0         |
| Eval Coalesce First     | hot   | 199.0 ns      | 204.0 ns  | 0      | 0         |
| Eval Map Member         | hot   | 176.4 ns      | 174.4 ns  | 0      | 0         |
| Eval OptionalChain Hit  | hot   | 175.4 ns      | 173.9 ns  | 0      | 0         |
| Eval Let                | hot   | **192.3 ns**  | 215.9 ns  | 0      | 0         |
| Eval Array Access       | hot   | 223.9 ns      | 222.6 ns  | 24     | 1         |
| Eval String Concat      | hot   | **545.0 ns**  | 720.6 ns  | 128    | 8         |
| Eval Regex Simple       | hot   | 323.0 ns      | 321.4 ns  | 16     | 1         |
| Eval Builtin Len        | hot   | **267.5 ns**  | 321.2 ns  | 56     | 2         |
| Eval Builtin Sum        | hot   | 1.17 µs       | 1.15 µs   | 64     | 3         |
| Eval CustomFunc 1-arg   | hot   | **500.1 ns**  | 577.5 ns  | 64     | 3         |
| Eval Env Small          | hot   | 216.3 ns      | 212.6 ns  | 0      | 0         |
| Eval Env Large          | hot   | **120.93 µs** | 199.71 µs | 226092 | 22        |
| Eval Env Struct         | hot   | **2.58 µs**   | 4.18 µs   | 5661   | 19        |
| Eval Lambda Simple      | hot   | **595.9 ns**  | 782.9 ns  | 164    | 5         |
| Builtin Map Lambda      | hot   | **47.49 µs**  | 58.11 µs  | 9480   | 408       |
| Eval Pipe Simple        | hot   | 1.16 µs       | 1.16 µs   | 64     | 3         |
| Eval Destructure Object | hot   | **493.2 ns**  | 510.2 ns  | 0      | 0         |




### Compile-once vs Eval-once


| Benchmark                 | Linux        | Windows      | B/op | allocs/op |
| ------------------------- | ------------ | ------------ | ---- | --------- |
| Compare Simple Eval-once  | **2.17 µs**  | 3.50 µs      | 5324 | 17        |
| Compare Simple Compiled   | **205.5 ns** | 212.1 ns     | 0    | 0         |
| Compare Complex Eval-once | **6.45 µs**  | 9.78 µs      | 7199 | 54        |
| Compare Complex Compiled  | 823.5 ns     | **674.4 ns** | 8    | 1         |




### Parallel / scenarios


| Benchmark                | Linux         | Windows      | B/op  | allocs/op |
| ------------------------ | ------------- | ------------ | ----- | --------- |
| Parallel Simple          | 103.6 ns      | 103.3 ns     | 0     | 0         |
| Parallel Complex         | **242.4 ns**  | 263.3 ns     | 8     | 1         |
| Parallel Builtin         | **505.8 ns**  | 549.9 ns     | 64    | 3         |
| Parallel CustomFunc      | **251.5 ns**  | 282.0 ns     | 64    | 3         |
| Parallel VMPool          | **890.5 ns**  | 1.00 µs      | 96    | 4         |
| Scenario BusinessRule    | 637.6 ns      | **370.1 ns** | 0     | 0         |
| Scenario Pricing         | 780.0 ns      | **470.4 ns** | 8     | 1         |
| Scenario Validation      | **803.2 ns**  | 940.9 ns     | 160   | 5         |
| Scenario DataTransform   | **1.11 µs**   | 1.50 µs      | 440   | 10        |
| Scenario FilterMapReduce | **108.48 µs** | 127.55 µs    | 20809 | 924       |
| Scenario Pipeline        | **29.77 µs**  | 33.24 µs     | 5267  | 246       |




### Optimizer, sandbox, memory, helpers


| Benchmark               | Linux        | Windows      | B/op | allocs/op |
| ----------------------- | ------------ | ------------ | ---- | --------- |
| Optimizer Effect +opt   | 69.22 ns     | **66.10 ns** | 0    | 0         |
| Optimizer Effect no-opt | 328.8 ns     | 329.3 ns     | 0    | 0         |
| Eval +sandbox           | **313.6 ns** | 438.6 ns     | 80   | 1         |
| Eval no-sandbox         | **229.6 ns** | 266.9 ns     | 0    | 0         |
| Eval Sandbox OpsCheck   | **421.0 ns** | 541.4 ns     | 80   | 1         |
| Memory NoAlloc          | 209.9 ns     | 207.8 ns     | 0    | 0         |
| Memory StringAlloc      | **551.3 ns** | 692.8 ns     | 128  | 8         |
| TypeChecker Simple      | 22.35 ns     | 22.03 ns     | 0    | 0         |
| Analyze Simple          | **1.40 µs**  | 2.29 µs      | 2024 | 15        |




### SCADA / PDM paths

Recipe benches in `bench_scada_test.go` (`BenchmarkSCADA_*`). Allocation floors are deterministic; re-run on the matched CI SKU before quoting latency.


| Benchmark                  | Class | What it proves                                 | B/op | allocs/op |
| -------------------------- | ----- | ---------------------------------------------- | ---- | --------- |
| SCADA SimpleArith Compiled | hot   | `EvalValue` + reused map + `NumericFloat64`    | 0    | 0         |
| SCADA Secure Simple        | hot   | Secure preset + `EvalFloat` (allow/deny cache) | 88   | 2         |
| SCADA TimeHour             | hot   | `time.Time` env + `timeHour` under Secure      | 112  | 2         |
| SCADA QualifiedMap         | hot   | `isGood`/`v` quality map path                  | 840  | 10        |




### Analysis

**Matched SKU makes OS gaps readable.** Both runners report `AMD EPYC 9V74` at GOMAXPROCS=4. Same SKU on both runners, so Linux↔Windows deltas are mostly OS/runtime, not CPU class.

**Hot Eval is OS-neutral.** Integer/logical/map/optional-chain paths sit in a ~0.98–1.05× band (IntSub ≈210 ns, ShortCircuit And ≈137 ns, Map Member ≈175 ns, Memory NoAlloc ≈210 ns). Prefer compiled `Program.Eval` for steady-state work; do not expect Windows to be “slower at expressions” on these floors. `Eval IntAdd`’s 1 alloc/op vs `IntSub`’s 0 is a measured floor on the add path (not a silent baseline rewrite).

**Compile / Analyze pay an OS tax.** Simple compile 1.75→2.87 µs (~1.6×), Analyze 1.40→2.29 µs (~1.6×). Lexer/compiler allocate (Windows `mem.prof`: `newCompiler` / `newLexer` among top `alloc_space`). Compile once and reuse — do not re-`Compile` per request.

**Compile-once wins by ~10× on simple and ~8× on complex (Linux).** Simple Eval-once 2.17 µs vs Compiled 205.5 ns; Complex Eval-once 6.45 µs vs Compiled 823.5 ns. That is the product thesis: amortize parse/optimize/codegen across many evals.

**Optimizer folds constants hard.** Effectiveness bench: 69 ns with optimizer vs 329 ns without (~4.7×) at 0 allocs. Compile-time cost of enabling the optimizer is small vs no-optimizer (3.35 vs 3.72 µs Linux).

**Sandbox is measurable, not free.** Linux: no-sandbox 230 ns → +sandbox 314 ns (~37%); OpsCheck 421 ns. Windows sandbox overhead is larger in absolute ns. Use sandbox when untrusted expressions need budgets; skip it for trusted internal rules if latency matters.

**Allocating paths diverge more by OS.** String Concat / Memory StringAlloc ~1.3× slower on Windows; Env Large / Env Struct ~1.6×; Builtin Len and CustomFunc show mid-teens percent gaps. Zero-alloc numeric/logical work does not.

**Higher-order / lambda work is allocation-bound.** Map-lambda ~47–58 µs with ~408 allocs/op; FilterMapReduce scenario ~108–128 µs with ~924 allocs. Prefer builtins without per-element closures when the hot path is a tight loop.

**SCADA floors.** Compiled `EvalValue` arithmetic can sit at **0 allocs/op**; Secure Simple stays at **88 B / 2 allocs** after allow/deny cache; quality-map helpers pay more (~840 B / 10 allocs) because `{v,q}` maps and builtins allocate. Latency belongs on the CI SKU — do not mix laptop numbers into the dual-OS table.

**Windows pprof (CPU):** `(*execVM).execute` / `(*Program).Eval` / `(*execVM).run` dominate cumulative time; `NewValue` and map iteration show up on allocating benches. Matches the “VM dispatch + Value traffic” mental model.

**Scatter caution.** Some scenario / compare rows (e.g. Linux Pricing, Complex Compiled) move tens of percent across the three runs — medians smooth that, but do not over-interpret a single bold cell as a stable OS win.

### What the numbers mean

These benches measure engine overhead (compile, VM dispatch, builtins, sandbox), not end-to-end business latency. A real rule that touches I/O or a large Go callback will dwarf a 200 ns Eval.

**What HyperEx is fast at:** compiled numeric/logical/map member paths with 0–1 allocs; parallel Eval of shared `Program`s; constant-folded expressions after Optimize; SCADA-style `EvalValue` + map reuse.

**Where cost concentrates:** first compile, large env materialization, string/array builders, lambda higher-order builtins, sandbox ops accounting, quality-map helpers.

**What they do not mean:** a release SLA, proof that Windows is “slower at expressions” on hot numeric floors, or that Secure mode is process isolation.

## Quality


| Metric                                                  | Evidence                                                                                                   |
| ------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| Unit / edge / race / fuzz / bench / example / footprint | `go test ./` (+ `-race`); `fuzz_test.go` (8 `Fuzz*`); `bench_*.go`; `example_test.go`; `footprint_test.go` |
| Cover bar                                               | CI ≥90% on library package (local **90.2%** statements)                                                    |
| Deps                                                    | stdlib-only (`go.mod`)                                                                                     |
| CI                                                      | Lint + OS×Go race tests with coverage ≥90%; fuzz discover; main-only benches with pprof artifacts          |


```powershell
go test -race -count=1 ./
go vet ./
staticcheck ./
golangci-lint run ./
go test -run='^$' -bench="." -benchmem -count=5 ./
go test -fuzz=FuzzEval_NoPanic_NoHang -fuzztime=30s
go test -fuzz=FuzzCompile_NoPanic -fuzztime=30s
# remaining Fuzz* targets: see fuzz_test.go
```

User-visible history: [CHANGELOG.md](CHANGELOG.md).

## File structure

```
hyperex.go              Engine, New, package Eval/Compile
program.go              Program Eval / EvalValue / typed helpers
env.go                  env map / struct conversion
vm.go                   bytecode interpreter (hot)
value.go                Value tagged union
compiler.go             AST → bytecode
parser.go / lexer.go / token.go / ast.go
optimizer.go            constant fold / strength reduce
sandbox.go              limits + Secure helpers
options.go              Option / CompileOption
errors.go               error types + all Err* sentinels
builtin.go + builtin_*.go   builtin catalog (math/string/array/…)
studio.go               Tokenize / Parse façades
time.go / quality.go / numeric.go / presets.go / sort.go
analytics.go / transform.go / typechecker.go
stats.go / names.go / intconv.go
doc.go / README.md / LANGUAGE.md / CHANGELOG.md
examples/               playground + cookbook (see examples/README.md)
```



## Related docs


| Document                                 | Role                                                                                        |
| ---------------------------------------- | ------------------------------------------------------------------------------------------- |
| [LANGUAGE.md](LANGUAGE.md)               | Normative syntax and semantics (incl. numeric / sandbox host options); **wins on conflict** |
| [CHANGELOG.md](CHANGELOG.md)             | User-visible history                                                                        |
| [doc.go](doc.go)                         | Package GoDoc                                                                               |
| [examples/README.md](examples/README.md) | Cookbook coverage matrix                                                                    |




## License

MIT — Copyright (c) 2026 Anton Syanov. See [LICENSE](LICENSE).