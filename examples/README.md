# HyperEx examples

Runnable recipes and an interactive playground for `github.com/aasyanov/hyperex`.

Language reference: [../LANGUAGE.md](../LANGUAGE.md) (normative syntax & semantics).

From the module root:

```bash
cd /path/to/hyperex
go run ./examples/<name>/
```

Build all recipes (including playground):

```bash
go build ./examples/...
```

## Design

| Layer | Role |
|-------|------|
| [playground/](playground/) | Interactive **language** surface (operators, builtins, options UI) |
| Cookbook dirs below | **Go public API** embedding recipes — one primary lesson each |
| Root `example_test.go` | Tiny GoDoc `Example*` snippets (pkg.go.dev) |

Recipes stay English-only, stdlib + `hyperex` only, and print labeled output safe for a short `go run`.

## Interactive playground

| Path | Description |
|------|-------------|
| [playground/](playground/) | Web REPL — language surface, builtins, eval options, analyze/profile/transforms |

```bash
go run ./examples/playground/
```

Open http://localhost:9090 — see [playground/README.md](playground/README.md).

## Cookbook recipes

| Dir | Topic |
|-----|-------|
| [basic/](basic/) | Package vs Engine one-shot `Eval*` / `MustEval` |
| [compile_reuse/](compile_reuse/) | `Compile` / `MustCompile`, `Program.Eval*`, `Source`, `ResultHint` |
| [struct_env/](struct_env/) | Struct env with `expr` tags; `WithTypeCheck` |
| [custom_funcs/](custom_funcs/) | `WithFunction(s)`, `AddFunction`, `WithOperator` (trust note) |
| [sandbox/](sandbox/) | Sandbox options + `WithMaxSourceBytes` + sentinel `errors.Is` |
| [numeric/](numeric/) | `WithNumericMode` / `WithNonFinite` (+ SecureCalc preset link) |
| [scada_secure/](scada_secure/) | `NewSecureCalc` / `NewSecureAuto` + `time.Time` env + `BuiltinNames` |
| [scada_quality/](scada_quality/) | Quality maps `{v,q}` + `WithSafeFunction` / `WithSafeFunctionMeta` |
| [scada_hotpath/](scada_hotpath/) | Compile-once + env reuse + `EvalValue` / `EvalFloat` |
| [strict/](strict/) | `WithStrictMode` + `ErrInvalidConversion` |
| [context_cancel/](context_cancel/) | Cancel / deadline → `ErrOperationCancelled` |
| [stats/](stats/) | `Engine.Stats` (`Compiles` / `Evals` / `EvalErrors`) |
| [analyze_profile/](analyze_profile/) | `Analyze`, `ProfileExpr`, `Program.Analyze` / `Profile`, `Warn*` |
| [transform/](transform/) | `Simplify` / `Expand` / `Inline` + `Tokenize` / `Parse` / `Walk` |
| [validate_extract/](validate_extract/) | `Validate*` / `Extract*` / `TypeChecker` via `Parse` |
| [compile_options/](compile_options/) | `As*` + `Unoptimized` (visible OpsCount diff) |
| [errors_demo/](errors_demo/) | Common typed errors + `ErrNonFinite` / `ErrSourceTooLarge` |
| [value_api/](value_api/) | `Value` constructors (incl. `Time` / `Duration`), `Try*`, `Equal` |
| [predicates/](predicates/) | `all` / `any` / `none` / `one` truthiness + predicate + pipe |

Example:

```bash
go run ./examples/basic/
go run ./examples/numeric/
go run ./examples/sandbox/
```

## Coverage matrix

Public symbol → primary example (or playground note).

### Engine / Program / package helpers

| Symbol | Covered by |
|--------|------------|
| `New` | `compile_reuse`, most recipes |
| `Engine.Compile` / `MustCompile` | `compile_reuse` |
| `Engine.Eval` / `MustEval` | `basic` |
| `Engine.EvalBool` / `EvalInt` / `EvalFloat` / `EvalString` | `basic`, `strict` |
| `Engine.AddFunction` | `custom_funcs` |
| `Engine.Stats` / `Stats` | `stats` |
| `Program.Eval` | `compile_reuse`, `struct_env` |
| `Program.EvalValue` | `scada_hotpath` |
| `Program.EvalBool` / `EvalInt` / `EvalFloat` / `EvalString` | `compile_reuse`, `strict`, `scada_hotpath` |
| `Program.Source` / `ResultHint` | `compile_reuse`, `compile_options` |
| `Program.Analyze` / `Profile` | `analyze_profile` |
| `Eval` / `MustEval` (package) | `basic` |
| `EvalBool` / `EvalInt` / `EvalFloat` / `EvalString` (package) | `basic` |
| `Compile` / `MustCompile` (package) | `compile_reuse` |

### Options

| Symbol | Covered by |
|--------|------------|
| `Option` / `CompileOption` | all configured recipes |
| `WithFunction` / `WithFunctions` | `custom_funcs` |
| `WithSafeFunction` / `WithSafeFunctionMeta` / quality builtins | `scada_quality` |
| `WithOperator` | `custom_funcs` |
| `WithOptimize` / `WithConstantFolding` / `WithStrengthReduction` | `compile_options` |
| `WithTypeCheck` | `struct_env`, `validate_extract` |
| `WithMaxCallDepth` | `sandbox` |
| `WithStrictMode` | `strict` |
| `WithNumericMode` / `WithNonFinite` | `numeric` |
| `WithMaxSourceBytes` | `sandbox`, `errors_demo` |
| `WithSandbox` / `SandboxConfig` | `sandbox` |
| `WithSecureSandbox` | `sandbox` |
| `WithMaxOps` / `WithTimeout` | `sandbox` |
| `WithAllowedFuncs` / `WithDeniedFuncs` | `sandbox` |
| `WithReadOnly` | `sandbox` |
| `PresetSecureCalc` / `PresetSecureAuto` / `NewSecureCalc` / `NewSecureAuto` | `scada_secure`, `numeric`, `scada_quality` |
| `AsBool` / `AsInt` / `AsFloat` / `AsString` | `compile_options`, `compile_reuse` |
| `Unoptimized` | `compile_options` |

### Analytics / transforms / validation / studio

| Symbol | Covered by |
|--------|------------|
| `Analyze` / `Analysis` / `Warnings` | `analyze_profile` |
| `WarnAmbiguousBitwiseOr` / `WarnBuiltinNamePipe` | `analyze_profile` |
| `Complexity` (+ constants / `String`) | `analyze_profile` |
| `ProfileExpr` / `Profile` | `analyze_profile`, `compile_options` |
| `CheckExpressionType` | `validate_extract` |
| `Simplify` / `Expand` / `Inline` | `transform`, `compile_options` |
| `ExtractVariables` / `ExtractFunctions` | `validate_extract` |
| `ExtractVariablesOpts` / `ExtractMemberRoots` | `validate_extract` |
| `Validate` / `ValidateWithEnv` | `validate_extract` |
| `Type` / `Type*` / `Type.String` | `validate_extract` |
| `NewTypeChecker` / `TypeChecker` / `FuncSig` | `validate_extract` |
| `TypeChecker.WithVar` / `WithFunc` / `WithStructEnv` / `WithEnvTypes` | `validate_extract` |
| `TypeChecker.Check` / `Infer` | `validate_extract` (via `Parse`) |
| `Tokenize` / `Token` / `TokenKind` | `transform` |
| `Parse` / `Walk` / `Transform` / `ASTToString` | `transform` |
| `AnalyzeAST` | prefer `Analyze` / `Program.Analyze` |

### Value API

| Symbol | Covered by |
|--------|------------|
| `Value` / `Nil` / `NewValue` | `value_api` |
| `Bool` / `Int` / `Float` / `String` / `Array` / `Map` / `Func` / `Time` / `Duration` | `value_api`, `scada_secure` |
| `Is*` / `Type` / `Len` / `Interface` | `value_api` |
| `BuiltinNames` / `IsBuiltin` | `scada_secure` |
| `TryBool` / `TryInt` / `TryFloat` / `TryString` | `value_api` |
| `Equal` / `Compare` / `Index` / `Get` | `value_api` |
| `ResultHint` / `ResultHint*` | `compile_reuse`, `compile_options` |

### Language builtins (selected)

| Symbol | Covered by |
|--------|------------|
| `all` / `any` / `none` / `one` (truthiness + predicate) | `predicates`, playground |

### Errors

| Symbol | Covered by |
|--------|------------|
| `ParseError` (+ `Line` / `Column` / `WithSource`) | `errors_demo` |
| `LexerError` | `errors_demo` |
| `UndefinedError` | `errors_demo` |
| `RuntimeError` (+ `Unwrap` / `WithSource`) | `errors_demo` |
| `TypeError` / `NewTypeError` | `errors_demo`, `validate_extract` |
| `TypeCheckErrors` | `validate_extract` |
| `SandboxError` (+ `Is` / `Unwrap`) | `sandbox` |
| `NewRuntimeErrorf` | `errors_demo` |
| `ErrDivisionByZero` / `ErrIndexOutOfBounds` / `ErrNotIndexable` | `errors_demo` |
| `ErrInvalidConversion` | `strict` |
| `ErrOperationCancelled` | `context_cancel` |
| `ErrOpsLimitExceeded` / `ErrTimeout` / `ErrFunctionNotAllowed` / `ErrAssignmentNotAllowed` | `sandbox` |
| `ErrMaxCallDepth` | `sandbox` |
| `ErrNonFinite` | `numeric`, `errors_demo` |
| `ErrSourceTooLarge` | `sandbox`, `errors_demo` |
| Other runtime/sandbox sentinels | listed in `errors_demo` / `sandbox`; trigger conditions vary |

Language syntax, builtins catalogs, and interactive option toggles are covered by the **playground UI**.

## Conventions

- Import `github.com/aasyanov/hyperex` (+ stdlib) only.
- All Eval surfaces take `context.Context` first.
- One primary lesson per directory; cross-links in file comments when needed.
- Recipes print labeled output and are safe to `go run` briefly.
- English only on the published surface (same rule as README / LANGUAGE / CHANGELOG).
