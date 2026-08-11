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

## Interactive playground

| Path | Description |
|------|-------------|
| [playground/](playground/) | Interactive web REPL — language surface, builtins, eval options, analyze/profile/transforms |

```bash
go run ./examples/playground/
```

Open http://localhost:9090 — see [playground/README.md](playground/README.md).

The playground covers the **interactive language surface** (operators, builtins, higher-order helpers, sample catalogs). The cookbook dirs below focus on the **Go public API**.

## Cookbook recipes

| Dir | Topic |
|-----|-------|
| [basic/](basic/) | `Eval`, typed `Eval*`, `MustEval` (package + Engine) |
| [compile_reuse/](compile_reuse/) | `New`, `Compile` / `MustCompile`, `Program.Eval*`, `Source` |
| [struct_env/](struct_env/) | Struct env with `expr` tags; `WithEnv` |
| [custom_funcs/](custom_funcs/) | `WithFunction(s)`, `AddFunction`, `WithOperator` |
| [sandbox/](sandbox/) | Sandbox options + `errors.Is` on limit/deny sentinels |
| [strict/](strict/) | `WithStrictMode` + `ErrInvalidConversion` |
| [context_cancel/](context_cancel/) | Cancel / deadline → `ErrOperationCancelled` |
| [stats/](stats/) | `Engine.Stats` (`Compiles` / `Evals` / `EvalErrors`) |
| [analyze_profile/](analyze_profile/) | `Analyze`, `ProfileExpr`, `Program.Analyze` / `Profile`, `Complexity` |
| [transform/](transform/) | `Simplify`, `Expand`, `Inline` (string APIs) |
| [validate_extract/](validate_extract/) | `Validate*`, `Extract*`, `CheckExpressionType` |
| [typecheck/](typecheck/) | `NewTypeChecker`, `WithVar` / `WithFunc` / `WithStructEnv`, `WithTypeCheck` |
| [compile_options/](compile_options/) | `AsBool`/`AsInt`/`AsFloat`/`AsString`, `Unoptimized`, optimize options |
| [errors_demo/](errors_demo/) | Common sentinels, `ParseError.Line`/`Column`, constructed errors |
| [value_api/](value_api/) | `Value` constructors, `Try*`, `Equal`, `Compare`, `Index`, `Get` |
| [predicates/](predicates/) | `all`/`any`/`none`/`one` truthiness + predicate forms, pipe |

Example:

```bash
go run ./examples/basic/
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
| `Program.EvalBool` / `EvalInt` / `EvalFloat` / `EvalString` | `compile_reuse`, `strict` |
| `Program.Source` | `compile_reuse` |
| `Program.Analyze` / `Profile` | `analyze_profile` |
| `Eval` / `MustEval` (package) | `basic` |
| `EvalBool` / `EvalInt` / `EvalFloat` / `EvalString` (package) | `basic` |
| `Compile` / `MustCompile` (package) | `compile_reuse` |

### Options

| Symbol | Covered by |
|--------|------------|
| `Option` / `CompileOption` | all configured recipes |
| `WithFunction` / `WithFunctions` | `custom_funcs` |
| `WithOperator` | `custom_funcs` |
| `WithEnv` | `struct_env` |
| `WithOptimize` / `WithConstantFolding` / `WithStrengthReduction` | `compile_options` |
| `WithTypeCheck` | `typecheck`, `validate_extract` |
| `WithMaxCallDepth` | `sandbox` |
| `WithStrictMode` | `strict` |
| `WithSandbox` / `SandboxConfig` | `sandbox` |
| `WithSecureSandbox` | `sandbox` |
| `WithMaxOps` / `WithTimeout` | `sandbox` |
| `WithAllowedFuncs` / `WithDeniedFuncs` | `sandbox` |
| `WithReadOnly` | `sandbox` |
| `AsBool` / `AsInt` / `AsFloat` / `AsString` | `compile_options` |
| `Unoptimized` | `compile_options` |

### Analytics / transforms / validation

| Symbol | Covered by |
|--------|------------|
| `Analyze` / `Analysis` | `analyze_profile` |
| `Complexity` (+ constants / `String`) | `analyze_profile` |
| `ProfileExpr` / `Profile` | `analyze_profile` |
| `CheckExpressionType` | `validate_extract`, `typecheck` |
| `Simplify` / `Expand` / `Inline` | `transform` |
| `ExtractVariables` / `ExtractFunctions` | `validate_extract` |
| `Validate` / `ValidateWithEnv` | `validate_extract` |
| `Type` / `Type*` constants / `Type.String` | `validate_extract`, `typecheck` |
| `NewTypeChecker` / `TypeChecker` / `FuncSig` | `typecheck` |
| `TypeChecker.WithVar` / `WithFunc` / `WithStructEnv` / `WithEnvType` / `WithEnvTypes` | `typecheck` |

### Value API

| Symbol | Covered by |
|--------|------------|
| `Value` / `Nil` / `NewValue` | `value_api` |
| `Bool` / `Int` / `Float` / `String` / `Array` / `Map` / `Func` | `value_api` |
| `Is*` / `Type` / `Len` / `Interface` | `value_api` |
| `TryBool` / `TryInt` / `TryFloat` / `TryString` | `value_api` |
| `Bool`/`Int`/`Float`/`String`/`Array`/`Map` accessors | `value_api` |
| `Equal` / `Compare` / `Index` / `Get` | `value_api` |

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
| `TypeError` / `NewTypeError` | `errors_demo`, `typecheck` |
| `TypeCheckErrors` | `typecheck` (via Compile / ValidateWithEnv) |
| `SandboxError` (+ `Is` / `Unwrap`) | `sandbox` |
| `NewRuntimeErrorf` | `errors_demo` |
| `ErrDivisionByZero` / `ErrIndexOutOfBounds` / `ErrNotIndexable` | `errors_demo` |
| `ErrInvalidConversion` | `strict` |
| `ErrOperationCancelled` | `context_cancel` |
| `ErrOpsLimitExceeded` / `ErrTimeout` / `ErrFunctionNotAllowed` / `ErrAssignmentNotAllowed` | `sandbox` |
| `ErrMaxCallDepth` | `sandbox` |
| Other runtime/sandbox sentinels (`ErrInvalidOperation`, `ErrNilDereference`, `ErrNotCallable`, `ErrArgumentCount`, `ErrInvalidType`, `ErrNotIterable`, `ErrKeyNotFound`, `ErrInvalidSliceIndex`, `ErrIntegerOverflow`, `ErrInvalidRegex`, `ErrMemoryLimitExceeded`, `ErrStackOverflow`) | listed in `errors_demo` / `sandbox` for `errors.Is`; trigger conditions vary |

### AST / Node-level (unexported concretes)

Concrete AST node types are **unexported**. There is no public `Parse` returning `Node`, so these require a `Node` you cannot construct from outside the package:

| Symbol | Notes |
|--------|-------|
| `Node` | Interface only; concretes unexported |
| `Walk` / `Transform` | Node-level; use string APIs instead |
| `ASTToString` | Used internally by `Simplify` / `Expand` / `Inline` |
| `AnalyzeAST` | Prefer `Analyze` / `Program.Analyze` |
| `TypeChecker.Check` / `Infer` | Prefer `CheckExpressionType`, `ValidateWithEnv`, `WithTypeCheck` |

Language syntax, builtins catalogs, and interactive option toggles are covered by the **playground UI**.

## Conventions

- Import `github.com/aasyanov/hyperex` (+ stdlib) only.
- All Eval surfaces take `context.Context` first.
- Recipes print labeled output and are safe to `go run` briefly.
