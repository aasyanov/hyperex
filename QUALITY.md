# HyperEx quality gates

Local commands from the module root (`github.com/aasyanov/hyperex`). Functional options types are `Option` / `CompileOption` (singular).

## Gate 0–2 — build, race, lint

```powershell
go build .
go test -race -count=1 . -timeout 300s
go vet ./...
golangci-lint run ./...
```

Focused layers:

```powershell
go test . -run Footprint -v
go test . -run Concurrency -v
go test . -run ErrorSentinel -v
go test . -run Options -v
```

## Gate 5 — fuzz (smoke)

```powershell
go test . -fuzz=FuzzEval_NoPanic_NoHang -fuzztime=5s
go test . -fuzz=FuzzCompileEval_Pipeline -fuzztime=5s
go test . -fuzz=FuzzLexer_NoPanic -fuzztime=5s
go test . -fuzz=FuzzParser_NoPanic -fuzztime=5s
```

## Gate 3 — benchmarks (smoke)

```powershell
# Windows: use -bench=. (equals), not -bench .
go test . -run=^$ -bench=. -benchmem -benchtime=100ms
```

## Footprint

See `footprint_test.go` — `unsafe.Sizeof` budgets for `Value`, `Engine`, `Program`, `vm`, `instruction`, `bytecode`.

## Concurrency note

`Engine` is documented safe for concurrent Compile/Eval. Compile uses a per-call optimizer copy and TypeChecker clone so shared counters/`errors`/`env` do not race (`-race` must stay green).

## Pyramid map (this package)

| Layer | File |
|-------|------|
| Unit (bulk) | `unit_test.go`, `node_test.go`, `gate_cover_test.go` |
| Sentinel unit (API) | `errors_test.go` |
| Runtime error matrix | `errors_sentinel_test.go` |
| Options | `options_test.go` |
| Concurrency | `concurrency_test.go` |
| Footprint | `footprint_test.go` |
| Fuzz | `fuzz_test.go` |
| Bench | `bench_test.go` |
| Examples | `example_test.go` + `examples/` (playground + cookbook) |

## Examples build

```powershell
go build ./examples/...
go run ./examples/basic/
```
