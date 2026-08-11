# Changelog

All notable changes to `github.com/aasyanov/hyperex` are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] — 2026-08-11

First public standalone release of `github.com/aasyanov/hyperex`. Greenfield surface: compile-once / eval-many expression engine with context-first Eval, best-effort sandbox, and a narrowed public façade (Wave 2–3). Covered by Gate M proofs and Gate 5 craft bars (cover ≥90%, fuzz, lint).

### Added

- **[LANGUAGE.md](LANGUAGE.md)** — normative English expression language specification (lexicon, operators, forms, builtins, pipe vs bitwise, pitfalls). Wins over README on syntax/semantics conflicts.
- **[AUDIT.md](AUDIT.md)** — threat model, trust boundaries, numeric/sandbox contracts, Gate M proof table.
- **`examples/`** — interactive `playground/` plus cookbook recipes with a public-API coverage matrix ([examples/README.md](examples/README.md)).
- Predicate forms for `all` / `any` / `none` / `one`: optional second `kindFunc` (`pred(elem)` / `pred(elem, index)`); truthiness forms unchanged.
- `Analysis.Warnings` — stable English diagnostics for ambiguous `|` and builtin-name pipe.
- `example_test.go` — runnable Examples for Eval, Compile+Program.Eval, WithFunction, sandbox, ExtractVariables, Analyze, Simplify.
- Ship-kit: `.gitignore`, `.golangci.yml`, MIT `LICENSE`, `.github/workflows/ci.yml` (lint + OS×Go matrix + fuzz + bench/pprof on main).
- Gate M package digest under gitignored `.cursor/rules/` (`hyperex.mdc` proof table).

### Fixed

- **Pipe vs bitwise `|`.** `ident | ident` is PIPE when the RHS name matches a default builtin; integer/float literals never pipe. See LANGUAGE.md §9.
- **Range `..` precedence.** Left-assoc, prec 13 with `minPrec`: `1+2..4` → `1+(2..4)`, `1..2+3` → `(1..2)+3`.
- **Destructure `;`.** Object/array destructuring may omit `;` like plain `let`.
- **`Value.Equal` / `NewValue`:** no panic on function values (`*builtinFunc`); `NewValue` preserves Func kind for builtins/closures.
- **Optimizer correctness** (aligned with LANGUAGE operand / coercion rules):
  - no `0+x` / `x+0` / `x*0` / `1*x` unless the operand is definitely numeric (`+` is also string concat);
  - no `x && true → x` / `x || false → x` (`&&`/`||` return an operand);
  - `~~x → x` only for integer x (`~` uses `Int()`);
  - `x**0` / `x**1` / `x**2` strength reduction only for numeric x.

### Changed

- Standalone module path `github.com/aasyanov/hyperex` (was in-tree under `smp` / `backend2/package/hyperex`).
- Published docs English-only (`README`, `LANGUAGE`, `AUDIT`, `QUALITY`, `examples/**`); agent notes in gitignored `.cursor/rules/` (no `llm.md` / `a-report-*` / quality wrappers).
- Test layout: `a_*_test.go` → `unit_test.go` / `node_test.go` / `bench_test.go`; coverage lift via `gate_cover_test.go`.
- CI coverage excludes `examples/` (`package main` demos); examples still `go build` in CI.
- Gate 2 craft: `names.go` for language identifiers; optimizer option flags wired in `Engine.New`; golangci-lint clean.

### Changed (Wave 3 — breaking)

- **Public API narrowing.** Unexported: `Lexer`, `Parser`/`NewParser`, `Token*`, `Opcode`, `Instruction`, `Bytecode`, `Compiler`, `VM` → `execVM`, `Closure`, `Optimizer`, `BuiltinFunc`, `DefaultBuiltins`, `IsSideEffectFree`, concrete AST (`IntLit` → `intLit`, …). Façade: `Engine`, `Program`, `Value`, Options, errors, transforms/analytics. No public `Parse`.
- **`ParseError.Line()` / `Column()`** restore structured location after `Token` was unexported.

### Changed (Wave 2 — breaking)

- **Context-first Eval.** All Eval surfaces take `context.Context` first; no Background wrappers on the primary API.
- **Env symmetry on `Program.Eval`** — nil, maps, struct/*struct (same `toEnvMap` rules as Engine).
- **`Engine.Stats()`** — lock-free snapshot: `Compiles`, `Evals`, `EvalErrors`.
- **Cancellation / timeouts:** `ctx.Done()` → `ErrOperationCancelled`; sandbox `WithTimeout` and ctx deadline — whichever fires first.
- **Strict typed helpers:** `WithStrictMode(true)` → `ErrInvalidConversion` instead of silent coerce.
