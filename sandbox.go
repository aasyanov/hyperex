package hyperex

import (
	"errors"
	"time"
)

// Sandbox sentinel errors.
var (
	// ErrOpsLimitExceeded is returned when bytecode operation count exceeds MaxOps.
	ErrOpsLimitExceeded = errors.New("hyperex: operations limit exceeded")

	// ErrMemoryLimitExceeded is returned when estimated memory usage exceeds MaxMemory.
	ErrMemoryLimitExceeded = errors.New("hyperex: memory limit exceeded")

	// ErrTimeout is returned when evaluation exceeds the configured Timeout.
	ErrTimeout = errors.New("hyperex: execution timeout")

	// ErrFunctionNotAllowed is returned when a called function is denied by allow/deny lists or secure mode.
	ErrFunctionNotAllowed = errors.New("hyperex: function not allowed")

	// ErrAssignmentNotAllowed is returned when assignment occurs under WithReadOnly(true).
	ErrAssignmentNotAllowed = errors.New("hyperex: assignment not allowed in read-only mode")

	// ErrStackOverflow is returned when vm stack depth exceeds MaxStackDepth.
	ErrStackOverflow = errors.New("hyperex: stack overflow")
)

// SandboxConfig defines resource limits and restrictions for safe execution.
type SandboxConfig struct {
	// MaxOps is the maximum number of bytecode operations allowed.
	// 0 means unlimited.
	MaxOps int64

	// MaxMemory is the maximum memory usage in bytes (approximate).
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
	// - Reject non-primitive / non-JSON-like values in the environment (no structs, pointers, funcs, chans, etc.)
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

// buildSandboxLimits converts SandboxConfig to runtime limits.
func buildSandboxLimits(config *SandboxConfig) *sandboxLimits {
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

	if config.Timeout > 0 {
		limits.deadline = time.Now().Add(config.Timeout)
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

// SandboxError represents a sandbox violation error.
type SandboxError struct {
	Type    string // "ops_limit", "timeout", "function_denied", etc.
	Message string
	Limit   int64 // The limit that was exceeded
	Actual  int64 // The actual value
	cause   error
}

func (e *SandboxError) Error() string {
	return e.Message
}

// Unwrap returns the underlying sentinel cause, if any.
func (e *SandboxError) Unwrap() error {
	return e.cause
}

// Is implements errors.Is for SandboxError.
// Prefer matching the wrapped cause; fall back to Type for errors constructed without cause.
func (e *SandboxError) Is(target error) bool {
	if e.cause != nil && target == e.cause {
		return true
	}
	switch target {
	case ErrOpsLimitExceeded:
		return e.Type == sandboxOpsLimit
	case ErrMemoryLimitExceeded:
		return e.Type == sandboxMemoryLimit
	case ErrTimeout:
		return e.Type == sandboxTimeout
	case ErrFunctionNotAllowed:
		return e.Type == sandboxFunctionDenied
	case ErrAssignmentNotAllowed:
		return e.Type == sandboxAssignmentDenied
	case ErrStackOverflow:
		return e.Type == sandboxStackOverflow
	}
	return false
}

// sandboxErr constructs a SandboxError with an optional sentinel cause.
func sandboxErr(cause error, typ, msg string, limit, actual int64) *SandboxError {
	return &SandboxError{
		Type:    typ,
		Message: msg,
		Limit:   limit,
		Actual:  actual,
		cause:   cause,
	}
}
