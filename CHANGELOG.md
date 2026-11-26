# Changelog

All notable changes to `github.com/aasyanov/hyperex` are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.5.1] — 2026-11-26

### Fixed

- **VM and lifecycle:** `chunk` no longer panics on extreme sizes; `..` observes ops/cancellation; higher-order lambdas share the parent context, ops budget, and call depth; nested `let` / destructuring restore outer bindings; `let` without a body evaluates its value once.
- **Sandbox and profiling:** Secure recursively validates nested `Value` arrays/maps; `Program.Profile` preserves Secure, allow/deny, read-only, resource limits, call depth, and `As*` result conversions; cyclic equality is cycle-safe and bounded.
- **Language and optimizer:** `not in` follows `in` precedence; constant folding preserves `NonFiniteError`, float bit equality, and NaN ordering; algebraic simplifications no longer change numeric types, hide runtime errors, or bypass overflow conversion; typed `EvalFloat` rejects non-finite conversion results.
- **Studio:** `Inline` substitutes only closed literals and preserves scope, effects, and evaluation errors; `Expand` respects lexical bindings and non-computed properties; `ASTToString` preserves computed keys, postfix precedence, lambda/unary forms, numeric types, `MinInt64`, and parseable non-finite expressions.
- **TypeChecker:** truthiness matches `LANGUAGE.md`; struct tags match runtime env; `NumericFloat64`, function/variadic argument types, and pipe argument insertion are modeled; destructuring no longer leaks bindings.
- **Collections and test oracle:** `unique` uses unambiguous structural keys; `sortBy` handles equal keys correctly in descending order; optimizer fuzz compares value types and semantic errors across optimized, unoptimized, and algebraic-only paths while allowing only resource-only divergence.

### Documentation

- Reworked `LANGUAGE.md` into a progressive tutorial and complete language reference with execution model, values, collections, lambdas, built-ins, numeric and temporal semantics, Studio, sandboxing, errors, and practical recipes.
- Corrected and clarified leading-zero octal literals, `MinInt64`, NaN equality/ordering, `#`/`$n` lambda rules, nested `let` scope, Secure nesting, non-recursive non-finite checks, and playground examples for `repeat`, `flatten`, and `_`.

## [1.5.0] — 2026-10-15

Studio / SCADA / numeric surface since `v1.0.0`. **SemVer note:** this minor includes breaking removals (module path stays `github.com/aasyanov/hyperex`; no `/v2`). Call sites using removed symbols must migrate before upgrading.

### Fixed

- SCADA benches (`BenchmarkSCADA_Secure_*` / TimeHour / QualifiedMap): disable wall-clock `Timeout` via `NewSecureCalc(WithTimeout(0))` so GC STW cannot flake the 25ms SecureCalc preset during long `-bench` runs. Alloc floors unchanged; product presets still use 25ms / 10ms.

### Removed

- **Breaking — removed APIs / language aliases:**
  - Builtin alias `size` (use `len`); catalog is **100** names
  - Keyword `null` (use `nil` only; `null` is a normal identifier)
  - Engine option `WithEnv` (use `WithTypeCheck`)
  - `TypeChecker.WithEnvType` varargs façade (use `WithStructEnv` / `WithVar` / `WithEnvTypes`)
  - Unused taxonomy sentinels never returned by Eval: `ErrInvalidOperation`, `ErrNilDereference`, `ErrNotIterable`, `ErrKeyNotFound`, `ErrIntegerOverflow`

### Added

- **Studio DX** — `Tokenize` / `Token` / `TokenKind`; public `Parse(expr) (Node, error)`; `ExtractVariablesOpts` + `ExtractMemberRoots()`; `WarnAmbiguousBitwiseOr` / `WarnBuiltinNamePipe`; `WithMaxSourceBytes` + `ErrSourceTooLarge`; `RuntimeError.Offset` / `Line` / `Column` (0 = unknown).
- **`WithNumericMode`** — `NumericDefault` (int-preserving) or `NumericFloat64` (`1/2` → `0.5`). String `+` and time/duration ops unchanged.
- **`WithNonFinite`** — `NonFiniteAllow` (default) or `NonFiniteError` (`errors.Is(…, ErrNonFinite)`).
- **`Program.ResultHint()`** / **`Program.EvalValue(ctx, env) (Value, error)`** — compile As* hint; Value without `Interface()` boxing (`Eval` kept).
- **First-class `time` / `duration`** — `NewValue(time.Time|time.Duration)`, constructors `Time` / `Duration`; temporal operators (`t±d`, `t-t`→`d`, `d±d`, `d*n` / `d/n`); builtins `timeUTC`, `timeUnix*`, calendar extractors, `timeAdd`, `duration*` / `dur*`. No `now()` builtin. Secure coerce accepts `time.Time` / `time.Duration`.
- **`BuiltinNames()` / `IsBuiltin(name)`** — public sorted name discovery.
- **Secure presets** — `PresetSecureCalc` / `PresetSecureAuto`, `NewSecureCalc` / `NewSecureAuto` (include `NumericFloat64` + `NonFiniteError`).
- **Quality helpers** — builtins `v` / `q` / `isGood` / `isBad` / `isUncertain` / `withQ`; exported `QualityClass*` masks. No arithmetic auto-unwrap.
- **`WithSafeFunction` / `WithSafeFunctionMeta` / `SafeFuncMeta`** — host funcs taxed against MaxOps; `WithFunction` remains trusted.
- **Examples / SCADA benches** — `scada_secure`, `scada_quality`, `scada_hotpath`; `BenchmarkSCADA_*` in `bench_scada_test.go`. Cookbook matrix raised (`numeric`, merged typecheck into `validate_extract`, Studio / `ResultHint` / `Warn*` coverage) — see [examples/README.md](examples/README.md).

### Changed

- **`Tokenize`**: operator / punctuation / keyword tokens fill `Token.Lit` from the glyph when the lexer leaves `Value` empty (highlighter-friendly).
- **`keys` / `values`** — sorted key order (determinism; previously unspecified map iteration).
- **`RuntimeError`** — optional additive `Expected` / `Got` on strict As* failures (`Error()` string unchanged for Op/Left/Right paths).
- **Sandbox limits cache** — allow/deny maps cached at `New`; Eval only refreshes Timeout deadline (~345→~88 B/op on Secure Simple; no API change).
- **Docs / layout** — published surface is README + LANGUAGE + CHANGELOG; numeric/sandbox/Secure contracts in LANGUAGE §3.5 / §11; internal test/source split into concept files (no API change); ship-kit / CI hygiene; README Quality notes honest fuzz count and reproduce commands.

## [1.0.0] — 2026-08-11

First public standalone release of `github.com/aasyanov/hyperex`: compile-once / eval-many engine, context-first `Eval`, best-effort sandbox, narrowed public façade (AST / VM / lexer unexported). Normative language spec in [LANGUAGE.md](LANGUAGE.md); `examples/` + playground. Covered by unit/race/fuzz, lint, cover ≥90%.

Notable vs earlier in-tree package: module path `github.com/aasyanov/hyperex`; pipe vs bitwise `|` and range `..` precedence fixed; optimizer aligned with coercion rules; `Analysis.Warnings` for ambiguous `|`. See git history / tag `v1.0.0` for the full API-narrowing and context-first Eval migration notes.
