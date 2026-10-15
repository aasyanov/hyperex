// Package hyperex - see doc.go for documentation.
package hyperex

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Engine is the main entry point for HyperEx expression evaluation.
// It holds configuration, custom functions, and provides methods
// for compiling and evaluating expressions.
//
// Engine is safe for concurrent use. Compiled programs can be
// shared across goroutines. Counters exposed by [Engine.Stats] are lock-free.
type Engine struct {
	config      *engineConfig
	builtins    map[string]Value
	optimizer   *optimizer
	typeChecker *TypeChecker
	mu          sync.RWMutex

	// sandboxBase caches allow/deny maps + numeric limits from New (immutable).
	// Each Eval shallow-copies it and refreshes deadline via sandboxTimeout.
	sandboxBase    *sandboxLimits
	sandboxTimeout time.Duration

	compiles   atomic.Int64
	evals      atomic.Int64
	evalErrors atomic.Int64
}

// New creates a new HyperEx engine with the given options.
//
// Example:
//
//	engine := hyperex.New(
//	    hyperex.WithFunction("double", func(x int) int { return x * 2 }),
//	    hyperex.WithMaxCallDepth(500),
//	)
func New(opts ...Option) *Engine {
	config := defaultConfig()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(config)
	}

	// Initialize builtins (or empty map if disabled)
	var builtins map[string]Value
	if config.sandbox != nil && config.sandbox.DisableBuiltins {
		builtins = make(map[string]Value)
	} else {
		builtins = defaultBuiltins()
	}

	// Add custom functions (even when builtins disabled, custom funcs are allowed)
	for name, fn := range config.functions {
		// Wrap custom functions as builtinFunc so sandbox allow/deny by name works
		// (and so vm can distinguish "named" functions from env-provided functions).
		builtins[name] = wrapHostFunction(name, fn, config.safeFuncs)
	}

	// Setup optimizer from engine options (WithConstantFolding / WithStrengthReduction).
	optimizer := newOptimizer().
		withConstantFolding(config.constantFolding).
		withStrengthReduction(config.strengthReduce).
		withNumericMode(config.numericMode)
	// Tell optimizer about overloaded operators so it doesn't optimize them
	if len(config.operators) > 0 {
		optimizer.withSkipOperators(config.operators)
	}

	// Setup type checker if enabled
	var typeChecker *TypeChecker
	if config.typeCheck && config.envVal != nil {
		typeChecker = NewTypeChecker().WithStructEnv(config.envVal)
	}

	eng := &Engine{
		config:      config,
		builtins:    builtins,
		optimizer:   optimizer,
		typeChecker: typeChecker,
	}
	if config.sandbox != nil {
		eng.sandboxBase = buildSandboxLimitsStatic(config.sandbox)
		eng.sandboxTimeout = config.sandbox.Timeout
	}
	return eng
}

// Compile parses and compiles an expression string into a reusable Program.
// The compiled program can be executed multiple times with different environments.
//
// Example:
//
//	program, err := engine.Compile(`x + y * 2`)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	result1, _ := program.Eval(ctx, map[string]any{"x": 10, "y": 5})
//	result2, _ := program.Eval(ctx, map[string]any{"x": 3, "y": 7})
func (e *Engine) Compile(expr string, opts ...CompileOption) (*Program, error) {
	e.compiles.Add(1)

	// Parse compile options
	compileConfig := &compileConfig{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(compileConfig)
	}

	if e.config.maxSourceBytes > 0 && len(expr) > e.config.maxSourceBytes {
		return nil, ErrSourceTooLarge
	}

	// Parse expression
	parser := newParser(expr)
	ast, err := parser.parse()
	if err != nil {
		// Add source context to parse errors
		if pe, ok := err.(*ParseError); ok {
			pe.Source = expr
		}
		return nil, err
	}

	// Type check if enabled.
	// Use a per-call clone: Check mutates errors/env and must be race-free under concurrent Compile.
	if e.typeChecker != nil {
		tc := e.typeChecker.cloneForCheck()
		if errs := tc.Check(ast); errs != nil && len(errs.Errors) > 0 {
			// Return all type errors combined
			return nil, errs
		}
	}

	// Optimize if enabled.
	// Copy optimizer: Optimize mutates foldedCount/reducedCount on the shared instance.
	shouldOptimize := e.config.optimize
	if compileConfig.optimize != nil {
		shouldOptimize = *compileConfig.optimize
	}
	if shouldOptimize && e.optimizer != nil {
		localOpt := *e.optimizer
		ast = localOpt.optimize(ast)
	}

	// Compile to bytecode
	compiler := newCompiler()
	if len(e.config.operators) > 0 {
		compiler.withOperators(e.config.operators)
	}
	bytecode, err := compiler.compileAST(ast)
	if err != nil {
		return nil, err
	}

	return &Program{
		source:       expr,
		bytecode:     bytecode,
		engine:       e,
		expectBool:   compileConfig.expectBool,
		expectInt:    compileConfig.expectInt,
		expectFloat:  compileConfig.expectFloat,
		expectString: compileConfig.expectString,
	}, nil
}

// MustCompile is like Compile but panics on error.
// Useful for package-level initialization.
//
// Example:
//
//	var expr = engine.MustCompile(`x > 0 && x < 100`)
func (e *Engine) MustCompile(expr string, opts ...CompileOption) *Program {
	p, err := e.Compile(expr, opts...)
	if err != nil {
		panic(err)
	}
	return p
}

// Eval compiles and evaluates an expression in a single call.
// For repeated evaluation, use Compile + Program.Eval instead.
//
// A nil ctx is treated as context.Background() (soft-open); prefer a non-nil ctx.
// ctx cancellation returns an error matching [ErrOperationCancelled].
// Sandbox [WithTimeout] and ctx deadlines both apply — whichever fires first wins.
//
// Example:
//
//	result, err := engine.Eval(ctx, `2 + 2 * 2`)
//	// result = 6
func (e *Engine) Eval(ctx context.Context, expr string, env ...any) (any, error) {
	program, err := e.Compile(expr)
	if err != nil {
		return nil, err
	}

	var envArg any
	if len(env) > 0 {
		envArg = env[0]
	}
	return program.Eval(ctx, envArg)
}

// MustEval is like Eval but panics on error.
func (e *Engine) MustEval(ctx context.Context, expr string, env ...any) any {
	result, err := e.Eval(ctx, expr, env...)
	if err != nil {
		panic(err)
	}
	return result
}

// EvalBool compiles and evaluates an expression, returning a boolean result.
// Under [WithStrictMode], invalid conversions return [ErrInvalidConversion].
func (e *Engine) EvalBool(ctx context.Context, expr string, env ...any) (bool, error) {
	result, err := e.Eval(ctx, expr, env...)
	if err != nil {
		return false, err
	}
	return convertBool(NewValue(result), e.config.strictMode)
}

// EvalInt compiles and evaluates an expression, returning an int64 result.
// Under [WithStrictMode], invalid conversions return [ErrInvalidConversion].
func (e *Engine) EvalInt(ctx context.Context, expr string, env ...any) (int64, error) {
	result, err := e.Eval(ctx, expr, env...)
	if err != nil {
		return 0, err
	}
	return convertInt(NewValue(result), e.config.strictMode)
}

// EvalFloat compiles and evaluates an expression, returning a float64 result.
// Under [WithStrictMode], invalid conversions return [ErrInvalidConversion].
func (e *Engine) EvalFloat(ctx context.Context, expr string, env ...any) (float64, error) {
	result, err := e.Eval(ctx, expr, env...)
	if err != nil {
		return 0, err
	}
	return convertFloat(NewValue(result), e.config.strictMode)
}

// EvalString compiles and evaluates an expression, returning a string result.
// Under [WithStrictMode], invalid conversions return [ErrInvalidConversion].
func (e *Engine) EvalString(ctx context.Context, expr string, env ...any) (string, error) {
	result, err := e.Eval(ctx, expr, env...)
	if err != nil {
		return "", err
	}
	return convertString(NewValue(result), e.config.strictMode)
}

// AddFunction adds a trusted function to the engine at runtime ([WithFunction] semantics).
// This is safe for concurrent use (uses copy-on-write).
func (e *Engine) AddFunction(name string, fn any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Copy-on-write: create a new map to avoid data races with concurrent Eval
	newBuiltins := make(map[string]Value, len(e.builtins)+1)
	for k, v := range e.builtins {
		newBuiltins[k] = v
	}
	// Trusted: no SafeFunction ops tax.
	newBuiltins[name] = wrapHostFunction(name, fn, nil)
	e.builtins = newBuiltins
}

// wrapHostFunction wraps a Go callback as a named builtinFunc.
// If name is present in safeFuncs, each call increments opsCount by the SafeFunction tax.
func wrapHostFunction(name string, fn any, safeFuncs map[string]SafeFuncMeta) Value {
	meta, safe := safeFuncs[name]
	if !safe {
		return Func(&builtinFunc{
			name: name,
			fn: func(_ *execVM, args []Value) (Value, error) {
				return callNativeFromBuiltin(fn, args)
			},
		})
	}
	tax := safeFuncOpsTax(meta)
	return Func(&builtinFunc{
		name: name,
		fn: func(vm *execVM, args []Value) (Value, error) {
			if vm != nil && tax > 0 {
				vm.opsCount += tax
				if err := vm.checkLimits(); err != nil {
					return Nil, err
				}
			}
			return callNativeFromBuiltin(fn, args)
		},
	})
}

// --- Package-level convenience functions ---

// defaultEngine backs package-level Eval/Compile helpers.
// Engine is safe for concurrent use; programs compiled from it may be shared across goroutines.
var defaultEngine = New()

// Eval compiles and evaluates an expression using the default engine.
//
// Example:
//
//	result, err := hyperex.Eval(ctx, `2 + 2`)
func Eval(ctx context.Context, expr string, env ...any) (any, error) {
	return defaultEngine.Eval(ctx, expr, env...)
}

// MustEval is like Eval but panics on error.
func MustEval(ctx context.Context, expr string, env ...any) any {
	return defaultEngine.MustEval(ctx, expr, env...)
}

// Compile compiles an expression using the default engine.
func Compile(expr string, opts ...CompileOption) (*Program, error) {
	return defaultEngine.Compile(expr, opts...)
}

// MustCompile is like Compile but panics on error.
func MustCompile(expr string, opts ...CompileOption) *Program {
	return defaultEngine.MustCompile(expr, opts...)
}

// EvalBool compiles and evaluates an expression, returning a boolean result.
func EvalBool(ctx context.Context, expr string, env ...any) (bool, error) {
	return defaultEngine.EvalBool(ctx, expr, env...)
}

// EvalInt compiles and evaluates an expression, returning an int64 result.
func EvalInt(ctx context.Context, expr string, env ...any) (int64, error) {
	return defaultEngine.EvalInt(ctx, expr, env...)
}

// EvalFloat compiles and evaluates an expression, returning a float64 result.
func EvalFloat(ctx context.Context, expr string, env ...any) (float64, error) {
	return defaultEngine.EvalFloat(ctx, expr, env...)
}

// EvalString compiles and evaluates an expression, returning a string result.
func EvalString(ctx context.Context, expr string, env ...any) (string, error) {
	return defaultEngine.EvalString(ctx, expr, env...)
}
