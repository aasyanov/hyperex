// Package hyperex - see doc.go for documentation.
package hyperex

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
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
		builtins[name] = Func(&builtinFunc{
			name: name,
			fn: func(_ *execVM, args []Value) (Value, error) {
				return callNativeFromBuiltin(fn, args)
			},
		})
	}

	// Setup optimizer from engine options (WithConstantFolding / WithStrengthReduction).
	optimizer := newOptimizer().
		withConstantFolding(config.constantFolding).
		withStrengthReduction(config.strengthReduce)
	// Tell optimizer about overloaded operators so it doesn't optimize them
	if len(config.operators) > 0 {
		optimizer.withSkipOperators(config.operators)
	}

	// Setup type checker if enabled
	var typeChecker *TypeChecker
	if config.typeCheck && config.envVal != nil {
		typeChecker = NewTypeChecker().WithStructEnv(config.envVal)
	}

	return &Engine{
		config:      config,
		builtins:    builtins,
		optimizer:   optimizer,
		typeChecker: typeChecker,
	}
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

// AddFunction adds a function to the engine at runtime.
// This is safe for concurrent use (uses copy-on-write).
func (e *Engine) AddFunction(name string, fn any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Copy-on-write: create a new map to avoid data races with concurrent Eval
	newBuiltins := make(map[string]Value, len(e.builtins)+1)
	for k, v := range e.builtins {
		newBuiltins[k] = v
	}
	// Wrap as builtinFunc to preserve sandbox allow/deny behavior by name.
	newBuiltins[name] = Func(&builtinFunc{
		name: name,
		fn: func(_ *execVM, args []Value) (Value, error) {
			return callNativeFromBuiltin(fn, args)
		},
	})
	e.builtins = newBuiltins
}

// toEnvMap converts various environment types to map[string]any.
func toEnvMap(env any) map[string]any {
	if env == nil {
		return nil
	}

	if m, ok := env.(map[string]any); ok {
		return m
	}

	rv := reflect.ValueOf(env)
	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		m := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			m[iter.Key().String()] = iter.Value().Interface()
		}
		return m
	}

	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		return structToMap(rv)
	}

	return nil
}

func structToMap(rv reflect.Value) map[string]any {
	rt := rv.Type()
	m := make(map[string]any, rv.NumField())

	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Tag.Get("expr")
		if name == "" {
			name = field.Name
		}
		if name == "-" {
			continue
		}

		m[name] = rv.Field(i).Interface()
	}

	return m
}

// Program represents a compiled expression ready for evaluation.
type Program struct {
	source       string
	bytecode     *bytecode
	engine       *Engine
	expectBool   bool
	expectInt    bool
	expectFloat  bool
	expectString bool
}

// Eval executes the compiled program with the given environment.
// env may be nil, map[string]any, another string-keyed map, or a struct/*struct
// (exported fields; `expr` tag or field name; `-` skips) — same rules as [Engine.Eval].
//
// ctx cancellation returns an error matching [ErrOperationCancelled].
// Sandbox [WithTimeout] and ctx deadlines both apply — whichever fires first wins.
//
// Example:
//
//	result, err := program.Eval(ctx, map[string]any{"x": 10, "y": 20})
func (p *Program) Eval(ctx context.Context, env any) (any, error) {
	p.engine.evals.Add(1)

	envMap := toEnvMap(env)

	// Build sandbox limits if configured
	var limits *sandboxLimits
	if p.engine.config.sandbox != nil {
		limits = buildSandboxLimits(p.engine.config.sandbox)
	}

	// Read builtins with read lock for thread safety
	p.engine.mu.RLock()
	builtins := p.engine.builtins
	p.engine.mu.RUnlock()

	var vm *execVM
	if limits != nil {
		vm = newVMWithLimits(p.bytecode, builtins, limits)
	} else {
		vm = newVM(p.bytecode, builtins)
	}

	// Apply maxCallDepth: sandbox takes priority, then engine config
	if limits != nil && limits.maxCallDepth > 0 {
		// Already set by newVMWithLimits, don't override
	} else if p.engine.config.maxCallDepth > 0 {
		vm.maxDepth = p.engine.config.maxCallDepth
	}

	if p.engine.config.strictMode {
		vm.strictMode = true
	}

	defer releaseVM(vm)

	result, err := vm.run(ctx, envMap)
	if err != nil {
		p.engine.evalErrors.Add(1)
		return nil, annotateEvalError(err, p.source)
	}

	strict := p.engine.config.strictMode

	if p.expectBool {
		b, cerr := convertBool(result, strict)
		if cerr != nil {
			p.engine.evalErrors.Add(1)
			return nil, annotateEvalError(cerr, p.source)
		}
		return b, nil
	}
	if p.expectInt {
		n, cerr := convertInt(result, strict)
		if cerr != nil {
			p.engine.evalErrors.Add(1)
			return nil, annotateEvalError(cerr, p.source)
		}
		return n, nil
	}
	if p.expectFloat {
		f, cerr := convertFloat(result, strict)
		if cerr != nil {
			p.engine.evalErrors.Add(1)
			return nil, annotateEvalError(cerr, p.source)
		}
		return f, nil
	}
	if p.expectString {
		s, cerr := convertString(result, strict)
		if cerr != nil {
			p.engine.evalErrors.Add(1)
			return nil, annotateEvalError(cerr, p.source)
		}
		return s, nil
	}

	return result.Interface(), nil
}

// EvalBool evaluates and returns a boolean result.
// Under [WithStrictMode], invalid conversions return [ErrInvalidConversion].
func (p *Program) EvalBool(ctx context.Context, env any) (bool, error) {
	result, err := p.Eval(ctx, env)
	if err != nil {
		return false, err
	}
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return convertBool(NewValue(result), p.engine.config.strictMode)
}

// EvalInt evaluates and returns an int64 result.
// Under [WithStrictMode], invalid conversions return [ErrInvalidConversion].
func (p *Program) EvalInt(ctx context.Context, env any) (int64, error) {
	result, err := p.Eval(ctx, env)
	if err != nil {
		return 0, err
	}
	return convertInt(NewValue(result), p.engine.config.strictMode)
}

// EvalFloat evaluates and returns a float64 result.
// Under [WithStrictMode], invalid conversions return [ErrInvalidConversion].
func (p *Program) EvalFloat(ctx context.Context, env any) (float64, error) {
	result, err := p.Eval(ctx, env)
	if err != nil {
		return 0, err
	}
	return convertFloat(NewValue(result), p.engine.config.strictMode)
}

// EvalString evaluates and returns a string result.
// Under [WithStrictMode], invalid conversions return [ErrInvalidConversion].
func (p *Program) EvalString(ctx context.Context, env any) (string, error) {
	result, err := p.Eval(ctx, env)
	if err != nil {
		return "", err
	}
	return convertString(NewValue(result), p.engine.config.strictMode)
}

// Source returns the original expression source.
func (p *Program) Source() string {
	return p.source
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

// annotateEvalError adds source context and normalizes runtime sentinels.
func annotateEvalError(err error, source string) error {
	switch e := err.(type) {
	case *RuntimeError:
		return e.WithSource(source)
	case *SandboxError:
		return e
	case *UndefinedError:
		return &RuntimeError{
			Message: e.Error(),
			Source:  source,
			cause:   e,
		}
	}
	if isKnownRuntimeErr(err) {
		return wrapRuntime(err).WithSource(source)
	}
	return err
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

func convertBool(v Value, strict bool) (bool, error) {
	if !strict {
		return v.Bool(), nil
	}
	return v.TryBool()
}

func convertInt(v Value, strict bool) (int64, error) {
	if !strict {
		return v.Int(), nil
	}
	return v.TryInt()
}

func convertFloat(v Value, strict bool) (float64, error) {
	if !strict {
		return v.Float(), nil
	}
	return v.TryFloat()
}

func convertString(v Value, strict bool) (string, error) {
	if !strict {
		return v.String(), nil
	}
	return v.TryString()
}
