# hyperex — Audit / Threat Model / Contracts

Stable guarantees, non-goals, and invariants for `github.com/aasyanov/hyperex`.
Use this document (with [LANGUAGE.md](LANGUAGE.md), [doc.go](doc.go), and [CHANGELOG.md](CHANGELOG.md)) so agents do not rediscover or invent contracts.

Standalone module **v1.0.0**. Production code is **stdlib-only**. Greenfield API changes after 1.0.0 require a new major (or explicit pre-release); sync docs in the same change.

**Language syntax & semantics** are normative in [LANGUAGE.md](LANGUAGE.md). This file covers threat model, numeric edge contracts, and API invariants of the engine.

## 1) Scope & threat model

| Goal | Detail |
|------|--------|
| Primary | Panic-safe, fast evaluation of expressions compiled to bytecode and run on a vm |
| Sandbox protects against | Runaway / expensive expressions, excessive memory, and (in Secure) access to arbitrary Go objects via env |
| Sandbox does **not** provide | Process isolation, crypto boundary, or trustworthiness of `WithFunction` callbacks |

**Trust note:** Go functions registered via `WithFunction` / `WithFunctions` / `AddFunction` may perform arbitrary I/O, networking, allocation, and blocking. “Full safety” requires controlling the allow-list of functions **and** the env data.

## 2) Trust boundaries

| Surface | Trust | Controls |
|---------|-------|----------|
| Expression source (`Compile` / `Eval`) | Untrusted | Parse/compile errors; sandbox MaxOps / Timeout / MaxMemory / MaxStackDepth; allow/deny builtins |
| Env map / struct | Caller-controlled | Secure coercion to JSON-like; block `kindAny` member/index in Secure |
| `WithFunction` callbacks | Fully trusted escape hatch | Document; do not treat as sandboxed |
| TypeChecker | Best-effort | Arg counts; intentional `TypeAny` for dynamic builtins |

## 3) API invariants

### 3.1 Eval surfaces

All Eval entry points take `context.Context` as the first parameter. There are no `Background`-wrapper variants.

| API | Env type |
|-----|----------|
| `Program.Eval` / `EvalBool` / `EvalInt` / `EvalFloat` / `EvalString` | `any` — same `toEnvMap` rules as Engine (nil, `map[string]any`, string-key maps, structs/`*structs` with `expr` tags) |
| Package `Eval` / `Engine.Eval` (and typed variants) | `...any` — first env arg converted by internal `toEnvMap` |
| `ProfileExpr` / `Program.Profile` | `ctx` + env (`any` on Program) |

`Engine.Stats()` returns a lock-free snapshot: `Compiles`, `Evals`, `EvalErrors`.

There is **no** `BatchEval`. There is **no** `RegisterFunc` (use `TypeChecker.WithFunc` or Engine `WithFunction`). There is **no** `Result` type.

### 3.1b Public surface (Wave 3)

Exported façade: `Engine`, `Program`, `Value`, `Option` / `CompileOption`, error types,
and transforms/analytics (`Analyze`, `Simplify`, `Extract*`, `Walk`, `Transform`, `Node`,
`TypeChecker`, `ASTToString`, …).

**Not exported:** `Lexer`, `Parser` / `NewParser`, `Token*`, `Opcode`, `Instruction`,
`Bytecode`, `Compiler`, `VM` (now `execVM`), `Closure`, `Optimizer`, `BuiltinFunc`,
`DefaultBuiltins`, `IsSideEffectFree`, and concrete AST node types (`IntLit` → `intLit`, …).
Do not document or invent them as public API. There is no public `Parse` entry point —
prefer string APIs (`Analyze`, `Simplify`, `CheckExpressionType`, …).

### 3.2 Option split

- Engine construction: `Option` — `WithFunction`, `WithSandbox`, `WithSecureSandbox`, `WithEnv`, `WithTypeCheck`, …
- Compile: `CompileOption` — `AsBool`, `AsInt`, `AsFloat`, `AsString`, `Unoptimized` only

### 3.3 `Value` construction

Exported for performance, but construct only via:
`NewValue`, `Int`, `Float`, `String`, `Bool`, `Array`, `Map`, `Func`.

Invariant: `kind` and `ptr`/`bits` must agree. Hand-built inconsistent `Value` → undefined behavior (may panic).

### 3.4 Errors: `errors.Is` / `errors.As`

- Runtime sentinels are `*RuntimeError` and remain comparable via `errors.Is` after source wrapping (`RuntimeError.Is`).
- Sandbox failures are `*SandboxError`; use `errors.Is` for sentinels and `errors.As` for details.

## 4) Numerical semantics

### 4.1 int64 and overflow

- Expression integer type is **`int64`**.
- On detected `int64` overflow, some vm arithmetic ops **promote to `float64`** (avoid wrap-around).

### 4.2 `ceil` / `floor` / `round`

- Normally return `int64`.
- Return **`float64`** for NaN/Inf or results outside `int64` range.

### 4.3 `fromJSON` safe integers

JSON numbers decode as `float64`. Conversion to `int64` only for IEEE-754 safe integers: \(|n| \le 2^{53}-1\).

## 5) Sandbox guarantees & non-guarantees

### 5.1 Limits

- `MaxOps`, `Timeout`, `MaxStackDepth` — long-running work must be interruptible.
- `MaxMemory` checked **periodically** (not every opcode) to avoid CPU DoS from the checker itself.

### 5.2 Context cancellation

- All Eval surfaces accept `context.Context`. vm execution observes `ctx.Done()` and returns an error matching `ErrOperationCancelled` (compare with `errors.Is`).
- Sandbox `WithTimeout` and `ctx` deadline both apply — **whichever fires first** wins.
- Under `WithStrictMode(true)`, typed `EvalBool`/`EvalInt`/`EvalFloat`/`EvalString` return `ErrInvalidConversion` instead of silent coercion.

### 5.3 Secure mode

When `Secure=true` (or `WithSecureSandbox()`):

- Coerce env toward JSON-like subset
- Forbid member/index/slice/contains on external `kindAny` Go objects
- Forbid calling unnamed env-passed Go functions (named builtins / custom functions still work by name)

Secure remains **best-effort**.

## 6) TypeChecker limits

Best-effort static check:

- Some builtins intentionally typed as `TypeAny` so docs/types do not lie about runtime (e.g. `abs`, `sum`, `ceil`/`floor`/`round`).
- Checks argument counts; not a full unifier for collections.

## 7) Portability (32-bit)

Runtime values use `int64`, but sites that need Go `int` (indices / step / size) clamp/check to avoid silent `int` overflow on 32-bit.

## 8) Gate M proof table

Commands (from module root):

```powershell
go test -count=1 .
go test -race -count=1 .
go test -run Example -count=1 .
go test -fuzz=. -fuzztime=30s .   # when signing Gate 5
```

| Mission / scenario | Proves | Evidence (`file:TestName`) |
|--------------------|--------|----------------------------|
| Compile-once / eval-many | Program reuse | `example_test.go:ExampleCompile`, `unit_test.go:TestProgram_TypedEval` |
| Context-first Eval + cancel | `errors.Is(…, ErrOperationCancelled)` | `context_eval_test.go:TestProgram_Eval_ContextCancel` |
| Context deadline | deadline → cancel sentinel | `context_eval_test.go:TestProgram_Eval_ContextDeadline` |
| Concurrent Program.Eval | race-safe shared Program | `concurrency_test.go:TestConcurrency_SharedProgramEval` |
| Concurrent Engine.Compile | race-safe Compile | `concurrency_test.go:TestConcurrency_SharedEngineCompile` |
| Stats lock-free snapshot | concurrent Stats | `stats_test.go:TestEngine_Stats_ConcurrentSafe` |
| Custom functions | `WithFunction` | `unit_test.go:TestEngine_CustomFunctions` |
| Sandbox MaxOps / Timeout / deny | limit contracts | `unit_test.go:TestSandbox_MaxOps`, `TestSandbox_Timeout`, `TestSandbox_DeniedFuncs` |
| Secure env coercion | Secure blocks struct pointers | `unit_test.go:TestSandbox_SecureEnv_RejectsStructPointer` |
| Range overflow / MaxInt64 | no DoS hang | `unit_test.go:TestRange_OverflowProtection`, `TestRange_MaxInt64_DoesNotHang` |
| fromJSON safe ints | 2^53 boundary | `unit_test.go:TestFromJSON_IntegerPreservation` |
| ceil/floor out of int64 | float fallback | `unit_test.go:TestCeilFloorRound_OutOfInt64Range_ReturnsFloat` |
| Runtime faults → errors | no panic to caller | `unit_test.go:TestRuntime_Errors`, `errors_sentinel_test.go` |
| Pipe vs bitwise `|` | heuristic + warnings | `parser_quirks_test.go:TestPipeVsBitwise_*` |
| Fuzz | no process panic on junk | `fuzz_test.go` (8 targets) |
| Examples | public API honesty | `example_test.go` |
| Stdlib-only | no prod requires | `go.mod` |

## 9) Craft gates

| Gate | Bar |
|------|-----|
| 0 | `go build .` |
| 1 | `go test -race -count=1 .` |
| 2 | + `go vet` + `golangci-lint run ./...` |
| 3 | benches + pprof on hot paths when claiming alloc floors |
| 4 | README + `doc.go` + this AUDIT + LANGUAGE + CHANGELOG synced to exports |
| 5 | cover ≥90% (CI) + fuzz ≥30s + bench evidence when claiming perf |
| **M** | Mission proof table above — no `_pending_` rows |

**Ship = Gate M + Gate 5.** Do not claim “secure” without §1/§5 boundaries.

## 10) Related

- [README.md](README.md) — human overview
- [LANGUAGE.md](LANGUAGE.md) — normative language specification
- [doc.go](doc.go) — package GoDoc / API surface
- [CHANGELOG.md](CHANGELOG.md) — release history
- [.cursor/rules/](.cursor/rules/) — Cursor agent constitution (gitignored)
