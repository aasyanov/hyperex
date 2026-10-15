package hyperex

import "time"

// PresetSecure* defaults (stdlib-only).
const (
	presetSecureCalcMaxOps    int64 = 100_000
	presetSecureCalcTimeout         = 25 * time.Millisecond
	presetSecureCalcCallDepth       = 64
	presetSecureAutoMaxOps    int64 = 20_000
	presetSecureAutoTimeout         = 10 * time.Millisecond
)

// PresetSecureCalc returns Engine options for integrator / PDM calc formulas:
// Secure sandbox, MaxOps 100000, Timeout 25ms, MaxCallDepth 64, ReadOnly,
// DeniedFuncs print, NumericFloat64, NonFiniteError.
func PresetSecureCalc() []Option {
	return []Option{
		WithSecureSandbox(),
		WithMaxOps(presetSecureCalcMaxOps),
		WithTimeout(presetSecureCalcTimeout),
		WithMaxCallDepth(presetSecureCalcCallDepth),
		WithReadOnly(true),
		WithDeniedFuncs(namePrint),
		WithNumericMode(NumericFloat64),
		WithNonFinite(NonFiniteError),
	}
}

// PresetSecureAuto returns Engine options for AUTO bool predicates:
// SecureCalc baseline with tighter MaxOps 20000, Timeout 10ms,
// DeniedFuncs print, fromJSON, toJSON, NumericFloat64, NonFiniteError.
func PresetSecureAuto() []Option {
	return []Option{
		WithSecureSandbox(),
		WithMaxOps(presetSecureAutoMaxOps),
		WithTimeout(presetSecureAutoTimeout),
		WithMaxCallDepth(presetSecureCalcCallDepth),
		WithReadOnly(true),
		WithDeniedFuncs(namePrint, nameFromJSON, nameToJSON),
		WithNumericMode(NumericFloat64),
		WithNonFinite(NonFiniteError),
	}
}

// NewSecureCalc constructs an Engine with PresetSecureCalc, then applies extra options.
func NewSecureCalc(extra ...Option) *Engine {
	opts := PresetSecureCalc()
	if len(extra) > 0 {
		opts = append(opts, extra...)
	}
	return New(opts...)
}

// NewSecureAuto constructs an Engine with PresetSecureAuto, then applies extra options.
func NewSecureAuto(extra ...Option) *Engine {
	opts := PresetSecureAuto()
	if len(extra) > 0 {
		opts = append(opts, extra...)
	}
	return New(opts...)
}
