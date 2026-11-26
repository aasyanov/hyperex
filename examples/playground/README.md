# HyperEx Playground

Interactive web REPL for the HyperEx expression engine (`github.com/aasyanov/hyperex`).

## Quick start

From the module root:

```bash
cd /path/to/hyperex
go run ./examples/playground/
```

Or build:

```bash
go build -o hyperex-playground.exe ./examples/playground/
.\hyperex-playground.exe
```

Open http://localhost:9090

### Flags

```
-port int       HTTP server port (default 9090)
-verbose        Enable verbose logging
```

## Features

- Restrained engineering workbench with compact panels, filtered catalog, and collapsible categories
- Catalog QA mode evaluates all 291 examples and reports every result with per-category totals
- 200+ categorized examples covering language features and builtins
- Interactive evaluation with sandbox, strict mode, and optimize options
- Static analysis and profiling
- Transforms: simplify, expand (macros), inline
- Validate, extract variables/functions, and typecheck helper
- Engine stats (`Compiles`, `Evals`, `EvalErrors`) on eval responses
- Embedded web UI (no external assets server)

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/eval` | Compile + evaluate; returns timings and `stats` |
| POST | `/api/analyze` | Static analysis |
| POST | `/api/profile` | `ProfileExpr` timings and op counts |
| POST | `/api/simplify` | Constant-fold / simplify |
| POST | `/api/expand` | Expand macros |
| POST | `/api/inline` | Inline `let` bindings |
| POST | `/api/validate` | Syntax validate |
| POST | `/api/extractVariables` | Variable names |
| POST | `/api/extractFunctions` | Function names |
| POST | `/api/typecheck` | Validate + compile probe + unbound env vars |
| GET | `/api/examples` | Example catalog |
| GET | `/api/functions` | Builtin function docs |

### POST `/api/eval`

```json
{
  "expression": "2 + 2 * 3",
  "env": {"x": 10},
  "options": {
    "sandbox": false,
    "strictMode": false,
    "optimize": true,
    "maxOps": 100000,
    "maxCallDepth": 100,
    "timeoutMs": 5000
  }
}
```

### POST `/api/typecheck`

```json
{
  "expression": "a + b",
  "env": {"a": 1}
}
```

Returns `valid`, `variables`, `functions`, and `unbound` (names missing from `env`).

## Layout

```
playground/
├── main.go         # HTTP server and API handlers
├── config.go       # Config and builtin function docs
├── examples.go     # Example catalog
├── examples_test.go # Executes every catalog example
├── api_test.go     # API endpoint coverage
├── README.md
└── web/
    └── index.html  # Embedded UI
```

## License

MIT — see the module [LICENSE](../../LICENSE).
