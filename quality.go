package hyperex

import (
	"fmt"
	"math"
)

// Quality class bits — OPC UA StatusCode–compatible convention (top 2 bits).
// Hosts may pass a full status code; only these bits classify Good / Uncertain / Bad.
const (
	// QualityClassMask selects the OPC-compatible quality class bits (0xC0000000).
	QualityClassMask uint32 = 0xC0000000
	// QualityClassGood is (code & QualityClassMask) == 0.
	QualityClassGood uint32 = 0
	// QualityClassUncertain is (code & QualityClassMask) == 0x40000000.
	QualityClassUncertain uint32 = 0x40000000
	// QualityClassBad is (code & QualityClassMask) == 0x80000000.
	QualityClassBad uint32 = 0x80000000
)

// QualityClass returns code & QualityClassMask.
func QualityClass(code uint32) uint32 {
	return code & QualityClassMask
}

// qualityBuiltins returns map-shaped quality helpers ({v,q} or bare qcode).
func qualityBuiltins() map[string]Value {
	return map[string]Value{
		nameV:           Func(&builtinFunc{name: nameV, fn: wrapBuiltin(builtinV)}),
		nameQ:           Func(&builtinFunc{name: nameQ, fn: wrapBuiltin(builtinQ)}),
		nameIsGood:      Func(&builtinFunc{name: nameIsGood, fn: wrapBuiltin(builtinIsGood)}),
		nameIsBad:       Func(&builtinFunc{name: nameIsBad, fn: wrapBuiltin(builtinIsBad)}),
		nameIsUncertain: Func(&builtinFunc{name: nameIsUncertain, fn: wrapBuiltin(builtinIsUncertain)}),
		nameWithQ:       Func(&builtinFunc{name: nameWithQ, fn: wrapBuiltin(builtinWithQ)}),
	}
}

func builtinV(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	x := args[0]
	if x.IsNil() {
		return Nil, nil
	}
	if x.IsMap() {
		m := x.Map()
		if m == nil {
			return Nil, &RuntimeError{Message: "v: expected map with key \"v\""}
		}
		val, ok := m[fieldV]
		if !ok {
			return Nil, &RuntimeError{Message: "v: map missing key \"v\""}
		}
		return NewValue(val), nil
	}
	switch x.kind {
	case kindInt, kindFloat, kindBool, kindString, kindTime, kindDuration, kindArray:
		return x, nil
	default:
		return Nil, &RuntimeError{Message: fmt.Sprintf("v: unsupported type %s", x.Type())}
	}
}

func builtinQ(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	x := args[0]
	if x.IsMap() {
		m := x.Map()
		if m == nil {
			return Nil, &RuntimeError{Message: "q: expected map with key \"q\""}
		}
		val, ok := m[fieldQ]
		if !ok {
			return Nil, &RuntimeError{Message: "q: map missing key \"q\""}
		}
		qv := NewValue(val)
		if !qv.IsNumber() {
			return Nil, &RuntimeError{Message: "q: map key \"q\" must be a number"}
		}
		return qv, nil
	}
	if x.IsNumber() {
		return x, nil
	}
	return Nil, &RuntimeError{Message: fmt.Sprintf("q: unsupported type %s", x.Type())}
}

func builtinIsGood(args []Value) (Value, error) {
	code, err := extractQualityCode(args)
	if err != nil {
		return Nil, err
	}
	return Bool(QualityClass(code) == QualityClassGood), nil
}

func builtinIsBad(args []Value) (Value, error) {
	code, err := extractQualityCode(args)
	if err != nil {
		return Nil, err
	}
	return Bool(QualityClass(code) == QualityClassBad), nil
}

func builtinIsUncertain(args []Value) (Value, error) {
	code, err := extractQualityCode(args)
	if err != nil {
		return Nil, err
	}
	return Bool(QualityClass(code) == QualityClassUncertain), nil
}

func builtinWithQ(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	if !args[1].IsNumber() {
		return Nil, &RuntimeError{Message: "withQ: qcode must be a number"}
	}
	var q any
	if args[1].IsFloat() {
		q = args[1].Float()
	} else {
		q = args[1].Int()
	}
	return Map(map[string]any{
		fieldV: args[0].Interface(),
		fieldQ: q,
	}), nil
}

func extractQualityCode(args []Value) (uint32, error) {
	if len(args) != 1 {
		return 0, ErrArgumentCount
	}
	x := args[0]
	if x.IsMap() {
		m := x.Map()
		if m == nil {
			return 0, &RuntimeError{Message: "quality: expected map with key \"q\""}
		}
		val, ok := m[fieldQ]
		if !ok {
			return 0, &RuntimeError{Message: "quality: map missing key \"q\""}
		}
		return numberToQualityCode(NewValue(val))
	}
	if x.IsNumber() {
		return numberToQualityCode(x)
	}
	return 0, &RuntimeError{Message: fmt.Sprintf("quality: unsupported type %s", x.Type())}
}

func numberToQualityCode(v Value) (uint32, error) {
	if v.IsInt() {
		n := v.Int()
		if n < 0 || n > math.MaxUint32 {
			return 0, &RuntimeError{Message: "quality: qcode out of uint32 range"}
		}
		return uint32(n), nil
	}
	if v.IsFloat() {
		f := v.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 || f > float64(math.MaxUint32) {
			return 0, &RuntimeError{Message: "quality: qcode out of uint32 range"}
		}
		return uint32(f), nil
	}
	return 0, &RuntimeError{Message: "quality: qcode must be a number"}
}
