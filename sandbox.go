package hyperex

import (
	"time"
)

// SandboxConfig defines resource limits and restrictions for safe execution.
type SandboxConfig struct {
	// MaxOps is the maximum number of bytecode operations allowed.
	// 0 means unlimited.
	MaxOps int64

	// MaxMemory is the maximum memory usage in bytes (approximate).
	// Estimation charges stack slots plus walked env Values; large heap behind a
	// single stack Value may be under-counted until it appears in env.
	// 0 means unlimited.
	MaxMemory int64

	// MaxStackDepth is the maximum stack depth allowed.
	// 0 means unlimited (stack grows dynamically).
	MaxStackDepth int

	// MaxCallDepth is the maximum function call nesting depth.
	// 0 means use engine default (typically 1000).
	MaxCallDepth int

	// Timeout is the maximum execution time.
	// 0 means no timeout.
	Timeout time.Duration

	// AllowedFuncs is a whitelist of allowed function names.
	// If nil, all functions are allowed (unless in DeniedFuncs).
	AllowedFuncs []string

	// DeniedFuncs is a blacklist of denied function names.
	// Takes precedence over AllowedFuncs.
	DeniedFuncs []string

	// ReadOnly prevents variable assignment (let expressions).
	ReadOnly bool

	// DisableBuiltins disables all built-in functions.
	DisableBuiltins bool

	// Secure enables hardened sandbox mode for untrusted expressions.
	//
	// When enabled, HyperEx will:
	// - Reject non-primitive / non-JSON-like values in the environment (no arbitrary structs, pointers, funcs, chans, etc.)
	// - Accept time.Time and time.Duration (Secure-compatible first-class kinds)
	// - Disallow reflection-based access on external Go objects (kindAny) during evaluation
	// - Disallow calling non-named functions (e.g. env-provided Go funcs), even if allow/deny lists are not set
	//
	// Note: this is still best-effort sandboxing; user-provided Go functions may perform arbitrary work.
	Secure bool
}

// sandboxLimits holds runtime limits for vm execution.
type sandboxLimits struct {
	maxOps        int64
	maxMemory     int64
	maxStackDepth int
	maxCallDepth  int
	deadline      time.Time
	allowedFuncs  map[string]bool // nil = all allowed
	deniedFuncs   map[string]bool
	readOnly      bool
	secure        bool
}

// WithSandbox configures the engine with sandbox restrictions.
//
// Default: no sandbox (nil config).
//
// Example:
//
//	engine := hyperex.New(
//	    hyperex.WithSandbox(hyperex.SandboxConfig{
//	        MaxOps:      10000,
//	        Timeout:     100 * time.Millisecond,
//	        DeniedFuncs: []string{"print"},
//	        ReadOnly:    true,
//	    }),
//	)
func WithSandbox(config SandboxConfig) Option {
	return func(c *engineConfig) {
		c.sandbox = &config
	}
}

// WithSecureSandbox enables hardened sandbox mode for untrusted expressions.
//
// Default: Secure=false (no hardened sandbox unless configured).
func WithSecureSandbox() Option {
	return func(c *engineConfig) {
		if c.sandbox == nil {
			c.sandbox = &SandboxConfig{}
		}
		c.sandbox.Secure = true
	}
}

// WithMaxOps sets the maximum number of bytecode operations.
// This prevents infinite loops and excessive computation.
//
// Default: 0 (unlimited). Values are stored on an implicit SandboxConfig if none exists.
//
// Example:
//
//	engine := hyperex.New(hyperex.WithMaxOps(10000))
func WithMaxOps(maxOps int64) Option {
	return func(c *engineConfig) {
		if c.sandbox == nil {
			c.sandbox = &SandboxConfig{}
		}
		c.sandbox.MaxOps = maxOps
	}
}

// WithTimeout sets the maximum execution time.
//
// Default: 0 (no timeout). Values are stored on an implicit SandboxConfig if none exists.
//
// Example:
//
//	engine := hyperex.New(hyperex.WithTimeout(100 * time.Millisecond))
func WithTimeout(timeout time.Duration) Option {
	return func(c *engineConfig) {
		if c.sandbox == nil {
			c.sandbox = &SandboxConfig{}
		}
		c.sandbox.Timeout = timeout
	}
}

// WithAllowedFuncs sets a whitelist of allowed functions.
// Only these functions can be called in expressions.
//
// Default: nil (all functions allowed, subject to DeniedFuncs).
//
// Example:
//
//	engine := hyperex.New(
//	    hyperex.WithAllowedFuncs("len", "upper", "lower", "sum"),
//	)
func WithAllowedFuncs(funcs ...string) Option {
	return func(c *engineConfig) {
		if c.sandbox == nil {
			c.sandbox = &SandboxConfig{}
		}
		c.sandbox.AllowedFuncs = funcs
	}
}

// WithDeniedFuncs sets a blacklist of denied functions.
// These functions cannot be called in expressions.
//
// Default: nil (none denied).
//
// Example:
//
//	engine := hyperex.New(
//	    hyperex.WithDeniedFuncs("print", "sprintf"),
//	)
func WithDeniedFuncs(funcs ...string) Option {
	return func(c *engineConfig) {
		if c.sandbox == nil {
			c.sandbox = &SandboxConfig{}
		}
		c.sandbox.DeniedFuncs = funcs
	}
}

// WithReadOnly prevents variable assignment in expressions.
// Let expressions will fail with ErrAssignmentNotAllowed.
//
// Default: false.
//
// Example:
//
//	engine := hyperex.New(hyperex.WithReadOnly(true))
func WithReadOnly(readOnly bool) Option {
	return func(c *engineConfig) {
		if c.sandbox == nil {
			c.sandbox = &SandboxConfig{}
		}
		c.sandbox.ReadOnly = readOnly
	}
}

// buildSandboxLimitsStatic converts SandboxConfig to runtime limits without a
// per-eval deadline. Allowed/Denied maps are immutable after construction and
// may be shared across concurrent Evals.
func buildSandboxLimitsStatic(config *SandboxConfig) *sandboxLimits {
	if config == nil {
		return nil
	}

	limits := &sandboxLimits{
		maxOps:        config.MaxOps,
		maxMemory:     config.MaxMemory,
		maxStackDepth: config.MaxStackDepth,
		maxCallDepth:  config.MaxCallDepth,
		readOnly:      config.ReadOnly,
		secure:        config.Secure,
	}

	if len(config.AllowedFuncs) > 0 {
		limits.allowedFuncs = make(map[string]bool, len(config.AllowedFuncs))
		for _, fn := range config.AllowedFuncs {
			limits.allowedFuncs[fn] = true
		}
	}

	if len(config.DeniedFuncs) > 0 {
		limits.deniedFuncs = make(map[string]bool, len(config.DeniedFuncs))
		for _, fn := range config.DeniedFuncs {
			limits.deniedFuncs[fn] = true
		}
	}

	return limits
}

// limitsWithFreshDeadline shallow-copies base (sharing allow/deny maps) and sets
// a fresh wall-clock deadline when timeout > 0.
func limitsWithFreshDeadline(base *sandboxLimits, timeout time.Duration) *sandboxLimits {
	if base == nil {
		return nil
	}
	out := *base
	if timeout > 0 {
		out.deadline = time.Now().Add(timeout)
	}
	return &out
}

// buildSandboxLimits converts SandboxConfig to runtime limits (fresh deadline).
func buildSandboxLimits(config *SandboxConfig) *sandboxLimits {
	if config == nil {
		return nil
	}
	return limitsWithFreshDeadline(buildSandboxLimitsStatic(config), config.Timeout)
}
