package hyperex

import (
	"reflect"
)

const (
	defaultOptimize        = true
	defaultConstantFolding = true
	defaultStrengthReduce  = true
	defaultMaxCallDepth    = 1000
	defaultStrictMode      = false
	defaultTypeCheck       = false
	defaultNumericMode     = NumericDefault
	defaultNonFiniteMode   = NonFiniteAllow
	// defaultSafeFuncOpsTax is the per-call ops tax for WithSafeFunction.
	// Tax formula is 1+MaxOwnOps, so WithSafeFunction sets MaxOwnOps = tax-1.
	defaultSafeFuncOpsTax int64 = 16
)

// Option configures an Engine via the functional options pattern.
// Nil Option values passed to New are ignored.
type Option func(*engineConfig)

// SafeFuncMeta describes a host function registered via [WithSafeFunction] /
// [WithSafeFunctionMeta]. Pure is documentation-only (best-effort claim);
// MaxOwnOps contributes to the per-call ops tax counted against MaxOps.
type SafeFuncMeta struct {
	// MaxOwnOps is added to the base tax of 1 per call (tax = 1 + MaxOwnOps).
	// Negative values are clamped so tax is at least 1.
	MaxOwnOps int64
	// Pure is a documentation / test obligation claim only; the VM does not enforce purity.
	Pure bool
}

// engineConfig holds the configuration for an Engine instance.
type engineConfig struct {
	// Custom functions to add to the environment
	functions map[string]any

	// safeFuncs marks functions registered via WithSafeFunction* (ops tax on call).
	// Names absent from this map that appear in functions are trusted (WithFunction).
	safeFuncs map[string]SafeFuncMeta

	// Custom operators
	operators map[string]string // maps operator to function name

	// Optimization settings
	optimize        bool
	constantFolding bool
	strengthReduce  bool

	// Runtime settings
	maxCallDepth int

	// Sandbox settings
	sandbox *SandboxConfig

	// Type checking
	typeCheck bool

	// Strict mode - errors on invalid type conversions instead of silent defaults
	strictMode bool

	// Numeric arithmetic policy (default = v1.0 int-preserving)
	numericMode NumericMode

	// Non-finite float result policy (default = allow NaN/Inf)
	nonFiniteMode NonFiniteMode

	// maxSourceBytes rejects Compile when len(expr) exceeds n (0 = unlimited).
	maxSourceBytes int

	// Type environment for typed compilation
	envType reflect.Type
	envVal  any
}

// defaultConfig returns the default configuration.
func defaultConfig() *engineConfig {
	return &engineConfig{
		functions:       make(map[string]any),
		safeFuncs:       make(map[string]SafeFuncMeta),
		operators:       make(map[string]string),
		optimize:        defaultOptimize,
		constantFolding: defaultConstantFolding,
		strengthReduce:  defaultStrengthReduce,
		maxCallDepth:    defaultMaxCallDepth,
		strictMode:      defaultStrictMode,
		typeCheck:       defaultTypeCheck,
		numericMode:     defaultNumericMode,
		nonFiniteMode:   defaultNonFiniteMode,
	}
}

// WithFunction adds a custom function to the expression environment.
// The function can be called by name in expressions.
//
// Trusted escape hatch: no forced MaxOps tax beyond normal bytecode accounting.
// Prefer [WithSafeFunction] for domain helpers under Secure / MaxOps budgets.
//
// Default: no custom functions.
//
// Example:
//
//	engine := hyperex.New(
//	    hyperex.WithFunction("double", func(x int) int { return x * 2 }),
//	)
//	result, _ := engine.Eval(`double(21)`) // 42
func WithFunction(name string, fn any) Option {
	return func(c *engineConfig) {
		c.functions[name] = fn
		delete(c.safeFuncs, name) // trusted; clear any prior SafeFunction mark
	}
}

// WithFunctions adds multiple custom functions at once.
//
// Each name is registered as trusted ([WithFunction] semantics).
//
// Default: no custom functions.
//
// Example:
//
//	engine := hyperex.New(
//	    hyperex.WithFunctions(map[string]any{
//	        "add": func(a, b int) int { return a + b },
//	        "mul": func(a, b int) int { return a * b },
//	    }),
//	)
func WithFunctions(funcs map[string]any) Option {
	return func(c *engineConfig) {
		for name, fn := range funcs {
			c.functions[name] = fn
			delete(c.safeFuncs, name)
		}
	}
}

// WithSafeFunction registers a host function that counts against MaxOps.
// Each call adds defaultSafeFuncOpsTax ops (tax = 1+MaxOwnOps, so MaxOwnOps = tax-1).
// AllowedFuncs / DeniedFuncs apply by name as for any other function.
// Panic in the Go function becomes a RuntimeError (same as [WithFunction]).
//
// Pure is not enforced — the host must keep the function free of I/O if claiming purity.
func WithSafeFunction(name string, fn any) Option {
	return WithSafeFunctionMeta(name, fn, SafeFuncMeta{MaxOwnOps: defaultSafeFuncOpsTax - 1})
}

// WithSafeFunctionMeta is [WithSafeFunction] with explicit tax / purity metadata.
// Per-call tax is 1+MaxOwnOps (minimum 1).
func WithSafeFunctionMeta(name string, fn any, meta SafeFuncMeta) Option {
	return func(c *engineConfig) {
		c.functions[name] = fn
		c.safeFuncs[name] = meta
	}
}

// safeFuncOpsTax returns the ops charged per call for a SafeFunction registration.
func safeFuncOpsTax(meta SafeFuncMeta) int64 {
	tax := int64(1) + meta.MaxOwnOps
	if tax < 1 {
		return 1
	}
	return tax
}

// WithOperator overloads an operator with a custom function.
// This allows customizing how operators like +, -, etc. behave.
//
// Default: no operator overloads (built-in operator semantics).
//
// Example:
//
//	engine := hyperex.New(
//	    hyperex.WithFunction("concat", func(a, b []int) []int { return append(a, b...) }),
//	    hyperex.WithOperator("+", "concat"),
//	)
func WithOperator(op, fnName string) Option {
	return func(c *engineConfig) {
		c.operators[op] = fnName
	}
}

// WithOptimize enables or disables all optimizations.
// When disabled, constant folding and strength reduction are also turned off.
//
// Default: true.
//
// Example:
//
//	engine := hyperex.New(hyperex.WithOptimize(false)) // disable all optimizations
func WithOptimize(enabled bool) Option {
	return func(c *engineConfig) {
		c.optimize = enabled
		if !enabled {
			c.constantFolding = false
			c.strengthReduce = false
		}
	}
}

// WithConstantFolding enables or disables constant folding optimization.
// When enabled, expressions like `2 + 2` are evaluated at compile time.
//
// Default: true.
//
// Example:
//
//	// With constant folding: `2 + 2 * 3` becomes `8` at compile time
//	engine := hyperex.New(hyperex.WithConstantFolding(true))
func WithConstantFolding(enabled bool) Option {
	return func(c *engineConfig) {
		c.constantFolding = enabled
	}
}

// WithStrengthReduction enables or disables strength reduction optimization.
// Replaces expensive operations with cheaper ones (e.g., x * 2 -> x << 1).
//
// Default: true.
func WithStrengthReduction(enabled bool) Option {
	return func(c *engineConfig) {
		c.strengthReduce = enabled
	}
}

// WithTypeCheck enables compile-time type checking using the provided env struct.
// Detects type mismatches at compile time rather than runtime.
//
// Default: false.
//
// Example:
//
//	type Env struct {
//	    Count int    `expr:"count"`
//	    Name  string `expr:"name"`
//	}
//	engine := hyperex.New(hyperex.WithTypeCheck(Env{}))
func WithTypeCheck(envType any) Option {
	return func(c *engineConfig) {
		c.typeCheck = true
		c.envType = reflect.TypeOf(envType)
		c.envVal = envType
	}
}

// WithMaxCallDepth sets the maximum recursion depth for function calls.
// This prevents stack overflow from infinite recursion.
// Values <= 0 are ignored (the previous/default depth is kept).
//
// Default: 1000.
//
// Example:
//
//	engine := hyperex.New(hyperex.WithMaxCallDepth(100))
func WithMaxCallDepth(depth int) Option {
	return func(c *engineConfig) {
		if depth > 0 {
			c.maxCallDepth = depth
		}
	}
}

// WithStrictMode enables strict type conversion mode.
// When enabled, invalid type conversions (like "abc" to int) will return an error
// instead of silently converting to default values (0 for int, 0.0 for float).
//
// Default: false.
//
// Example:
//
//	engine := hyperex.New(hyperex.WithStrictMode(true))
//	_, err := engine.Eval(`"abc" + 1`) // Error: cannot convert "abc" to number
func WithStrictMode(strict bool) Option {
	return func(c *engineConfig) {
		c.strictMode = strict
	}
}

// WithMaxSourceBytes rejects [Engine.Compile] when len(expr) exceeds n bytes.
// Values <= 0 disable the limit (default: unlimited).
// Oversized sources return an error matching [ErrSourceTooLarge].
func WithMaxSourceBytes(n int) Option {
	return func(c *engineConfig) {
		if n > 0 {
			c.maxSourceBytes = n
		} else {
			c.maxSourceBytes = 0
		}
	}
}

// CompileOption configures a single Compile call.
// Nil CompileOption values passed to Compile are ignored.
type CompileOption func(*compileConfig)

// compileConfig holds compilation-specific settings.
type compileConfig struct {
	// Expected return type
	expectBool   bool
	expectInt    bool
	expectFloat  bool
	expectString bool

	// Optimization override
	optimize *bool
}

// AsBool sets an eval-time conversion hint that the result should be bool.
func AsBool() CompileOption {
	return func(c *compileConfig) {
		c.expectBool = true
	}
}

// AsInt sets an eval-time conversion hint that the result should be int64.
// Non-integer results will be converted to int64.
func AsInt() CompileOption {
	return func(c *compileConfig) {
		c.expectInt = true
	}
}

// AsFloat specifies that the expression should return a float.
// Non-float results will be converted to float64.
func AsFloat() CompileOption {
	return func(c *compileConfig) {
		c.expectFloat = true
	}
}

// AsString specifies that the expression should return a string.
// Non-string results will be converted to string.
func AsString() CompileOption {
	return func(c *compileConfig) {
		c.expectString = true
	}
}

// Unoptimized disables optimization for this compilation only,
// overriding the engine-level optimize setting.
func Unoptimized() CompileOption {
	return func(c *compileConfig) {
		opt := false
		c.optimize = &opt
	}
}
