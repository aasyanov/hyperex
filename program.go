package hyperex

import (
	"context"
	"math"
)

// Program represents a compiled expression ready for evaluation.
// Program is safe for concurrent use by multiple goroutines.
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
// A nil ctx is treated as context.Background() (soft-open); prefer a non-nil ctx.
// ctx cancellation returns an error matching [ErrOperationCancelled].
// Sandbox [WithTimeout] and ctx deadlines both apply — whichever fires first wins.
//
// Example:
//
//	result, err := program.Eval(ctx, map[string]any{"x": 10, "y": 20})
func (p *Program) Eval(ctx context.Context, env any) (any, error) {
	v, err := p.EvalValue(ctx, env)
	if err != nil {
		return nil, err
	}
	return v.Interface(), nil
}

// EvalValue executes the compiled program and returns a [Value] without Interface() boxing.
// Compile As* options still apply (result is converted to the hinted kind as a Value).
// Under [NonFiniteError], NaN/Inf float results return an error matching [ErrNonFinite].
//
// A nil ctx is treated as context.Background() (soft-open); prefer a non-nil ctx.
func (p *Program) EvalValue(ctx context.Context, env any) (Value, error) {
	p.engine.evals.Add(1)

	envMap := toEnvMap(env)

	// Build sandbox limits if configured (reuse cached allow/deny maps).
	var limits *sandboxLimits
	if p.engine.sandboxBase != nil {
		limits = limitsWithFreshDeadline(p.engine.sandboxBase, p.engine.sandboxTimeout)
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
	vm.numericMode = p.engine.config.numericMode
	vm.nonFiniteMode = p.engine.config.nonFiniteMode

	defer releaseVM(vm)

	result, err := vm.run(ctx, envMap)
	if err != nil {
		p.engine.evalErrors.Add(1)
		return Nil, annotateEvalError(err, p.source)
	}

	result, err = p.convertResult(result)
	if err != nil {
		p.engine.evalErrors.Add(1)
		return Nil, annotateEvalError(err, p.source)
	}
	return result, nil
}

func (p *Program) convertResult(result Value) (Value, error) {
	strict := p.engine.config.strictMode
	if p.expectBool {
		b, err := convertBool(result, strict)
		if err != nil {
			return Nil, err
		}
		return Bool(b), nil
	}
	if p.expectInt {
		n, err := convertInt(result, strict)
		if err != nil {
			return Nil, err
		}
		return Int(n), nil
	}
	if p.expectFloat {
		f, err := convertFloat(result, strict)
		if err != nil {
			return Nil, err
		}
		out := Float(f)
		if err := rejectNonFiniteResult(p.engine.config.nonFiniteMode, out); err != nil {
			return Nil, err
		}
		return out, nil
	}
	if p.expectString {
		s, err := convertString(result, strict)
		if err != nil {
			return Nil, err
		}
		return String(s), nil
	}
	if err := rejectNonFiniteResult(p.engine.config.nonFiniteMode, result); err != nil {
		return Nil, err
	}
	return result, nil
}

// ResultHint returns the compile-time As* expectation for this program.
func (p *Program) ResultHint() ResultHint {
	switch {
	case p.expectBool:
		return ResultHintBool
	case p.expectInt:
		return ResultHintInt
	case p.expectFloat:
		return ResultHintFloat
	case p.expectString:
		return ResultHintString
	default:
		return ResultHintAny
	}
}

// rejectNonFiniteResult returns [ErrNonFinite] when mode is NonFiniteError and v is NaN/Inf.
func rejectNonFiniteResult(mode NonFiniteMode, v Value) error {
	if mode != NonFiniteError || !v.IsFloat() {
		return nil
	}
	f := v.Float()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return wrapRuntime(ErrNonFinite)
	}
	return nil
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
	f, err := convertFloat(NewValue(result), p.engine.config.strictMode)
	if err != nil {
		return 0, err
	}
	if err := rejectNonFiniteResult(p.engine.config.nonFiniteMode, Float(f)); err != nil {
		return 0, err
	}
	return f, nil
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

const invalidConversionMsg = "invalid type conversion"

func convertBool(v Value, strict bool) (bool, error) {
	if !strict {
		return v.Bool(), nil
	}
	b, err := v.TryBool()
	if err != nil {
		return false, &RuntimeError{
			Message:  invalidConversionMsg,
			Expected: nameBool,
			Got:      v.Type(),
			cause:    ErrInvalidConversion,
		}
	}
	return b, nil
}

func convertInt(v Value, strict bool) (int64, error) {
	if !strict {
		return v.Int(), nil
	}
	n, err := v.TryInt()
	if err != nil {
		return 0, &RuntimeError{
			Message:  invalidConversionMsg,
			Expected: nameInt,
			Got:      v.Type(),
			cause:    ErrInvalidConversion,
		}
	}
	return n, nil
}

func convertFloat(v Value, strict bool) (float64, error) {
	if !strict {
		return v.Float(), nil
	}
	f, err := v.TryFloat()
	if err != nil {
		return 0, &RuntimeError{
			Message:  invalidConversionMsg,
			Expected: nameFloat,
			Got:      v.Type(),
			cause:    ErrInvalidConversion,
		}
	}
	return f, nil
}

func convertString(v Value, strict bool) (string, error) {
	if !strict {
		return v.String(), nil
	}
	s, err := v.TryString()
	if err != nil {
		return "", &RuntimeError{
			Message:  invalidConversionMsg,
			Expected: nameString,
			Got:      v.Type(),
			cause:    ErrInvalidConversion,
		}
	}
	return s, nil
}
