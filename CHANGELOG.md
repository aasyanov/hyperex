# Changelog

All notable changes to `github.com/aasyanov/hyperex` are documented here.

## [Unreleased]

### Changed
- Standalone module path `github.com/aasyanov/hyperex` (`go.mod`); imports updated from `smp/package/hyperex`.
- Ship-kit unified with org canon: identical `.gitignore`, `.golangci.yml`, MIT `LICENSE`, and `.github/workflows/ci.yml` (lint + OS×Go matrix + fuzz discover + bench/pprof on main).
- Agent guidance moved to gitignored `.cursor/rules/`; removed `llm.md` and ad-hoc `a-report-*` tooling where present.
- Published docs rewritten to English: `README.md` (honest identity/install/quick start), `LANGUAGE.md` (normative spec), `AUDIT.md` / `QUALITY.md` / `examples/**` paths for the standalone module.
- Gate M proof table recorded in `AUDIT.md` §8 and `.cursor/rules/hyperex.mdc`.
- Renamed `a_*_test.go` → `unit_test.go` / `node_test.go` / `bench_test.go` (no lexical `a_` test prefixes).
- Gate 2 craft: `names.go` for repeated language identifiers (goconst), wired optimizer option flags in `Engine.New`, removed unused private helpers, golangci-lint clean.
- Coverage raised to ≥90% (`gate_cover_test.go`); fuzz pipeline uses tighter sandbox + eval context timeout.
- **`Value.Equal` / `NewValue`:** no longer panic on function values (`*builtinFunc`); `NewValue` preserves Func kind for builtins/closures. Optimize fuzz compares successful results only (folds may mask UndefinedError).

### Fixed

- **Pipe vs bitwise `|`.** `ident | ident` is now PIPE when the right-hand name matches a
  default builtin (`sum`, `unique`, `filter`, …). Integer/float literals on either side
  never pipe. `Analysis.Warnings` reports true ambiguity (`x | y`) and builtin-name pipe
  choices. See LANGUAGE.md §9.
- **Range `..` precedence.** `..` now respects `minPrec` and parses RHS at prec+1
  (left-assoc, prec 13): `1+2..4` → `1+(2..4)`, `1..2+3` → `(1..2)+3`, `1..2..3` → `(1..2)..3`.
- **Destructure `;`.** Object/array destructuring may omit `;` like plain `let`
  (`Body = Value` when final).

### Added

- **[LANGUAGE.md](LANGUAGE.md)** — normative HyperEx expression language specification
  (lexicon, operators, forms, 76 builtins, pipe vs bitwise, pitfalls). Russian prose;
  code identifiers English. Wins over README on syntax/semantics conflicts.
- **`examples/`** under the package: interactive `playground/` (moved from `backend2/examples/hyperex`)
  plus cookbook recipes with a public-API coverage matrix (`examples/README.md`).
  Old path is a redirect README only.
- **Predicate forms for `all` / `any` / `none` / `one`:** optional second `kindFunc` argument
  (`pred(elem)` / `pred(elem, index)`), same callback pattern as `filter`/`map`. One-arg
  truthiness forms unchanged. Empty-array semantics preserved. See LANGUAGE.md §8.4;
  recipe `examples/predicates/`.
- **`Analysis.Warnings`** — stable English diagnostics for ambiguous `|` and builtin-name pipe.

### Changed (Wave 3 — breaking)

- **Public API narrowing.** Unexported low-level pipeline types: `Lexer`, `Parser`/`NewParser`,
  `Token*`, `Opcode`, `Instruction`, `Bytecode`, `Compiler`, `VM` → `execVM`, `Closure`,
  `Optimizer`, `BuiltinFunc`, `DefaultBuiltins`, `IsSideEffectFree`, and concrete AST nodes
  (`IntLit` → `intLit`, …). Public façade remains: `Engine`, `Program`, `Value`, Options,
  errors, and transforms/analytics (`Analyze`, `Simplify`, `Extract*`, `Walk`, `Transform`,
  `Node`, `TypeChecker`, `ASTToString`, …). No public `Parse`.
- **`ParseError.Line()` / `Column()`** accessors restore structured location after `Token` was unexported.

### Changed (Wave 2 — breaking)

- **Context-first Eval API.** All Eval surfaces now take `context.Context` as the first parameter:
  - `Eval(ctx, expr, env...)`, `MustEval(ctx, …)`, typed `EvalBool`/`EvalInt`/`EvalFloat`/`EvalString(ctx, …)`
  - `Engine.Eval` / `MustEval` / typed variants — same shape
  - `Program.Eval(ctx, env)` / typed `Program.Eval*(ctx, env)` — env is `any` (same `toEnvMap` rules as Engine)
  - `ProfileExpr(ctx, expr, env)`, `Program.Profile(ctx, env)`
- **No Background wrappers** on the primary API — pass `context.Background()` at the call site.
- **Env symmetry on Program.Eval** — accepts nil, maps, and struct/*struct (not map-only).
- **`Engine.Stats()`** — lock-free snapshot: `Compiles`, `Evals`, `EvalErrors`.
- **Cancellation:** `ctx.Done()` during vm → `errors.Is(err, ErrOperationCancelled)`.
- **Timeout precedence:** sandbox `WithTimeout` and ctx deadline both apply; whichever fires first wins.
- **Strict typed helpers:** `WithStrictMode(true)` → typed Eval* return `ErrInvalidConversion` instead of silent coerce.
- **tagreg:** calc entries store `compileErr`; `Registry.CompileError(id)` for diagnostics.

### Documentation

- Wave 3: docs synced to narrowed public surface (`README`, `doc.go`, `llm.md`, `AUDIT.md`, `.cursor/rules/hyperex.mdc`) — no invented `NewParser` / `Lexer` / `VM` / … as public API.
- Added package `LICENSE` (MIT, Copyright (c) 2026 Anton Syanov).
- Added `llm.md` with CRITICAL RULES for agents (sandbox honesty, Value construction, `errors.Is`, compile-once/eval-many, numeric edges, non-goals).
- Added `example_test.go` with runnable Examples for Eval, Compile+Program.Eval, WithFunction, sandbox, ExtractVariables, Analyze, Simplify.
- Added `.cursor/rules/` agent constitution (gates, godoc, errors, options, testing, package digest, documentation).
- Refreshed `AUDIT.md` (English threat model, trust boundaries, Gate M checklist, `Option` vs `CompileOption`).
- Fixed `doc.go` honesty bugs: removed BatchEval claims; prior docs incorrectly limited `Program.Eval` to maps (Wave 2 restored env symmetry).
- README uplift: What HyperEx Is / Is Not; Critical contracts pointers; `RegisterFunc` → `WithFunc`; sandbox on Engine not Compile; badge/bench count fixes.
- Final sync to live API: singular `Option` / `CompileOption`.

### Notes for API / Quality waves

- Footprint / mass test renames applied where API renames broke tests so Examples could run (`DebugMode` → `SetDebugModeForTest`, `Value.Index(int)`).
