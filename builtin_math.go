package hyperex

import "math"

// Math builtins
func builtinAbs(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	v := args[0]
	if v.IsInt() {
		n := v.Int()
		if n < 0 {
			// Handle MinInt64: abs(MinInt64) overflows for int64, so return float.
			if n == math.MinInt64 {
				return Float(-float64(n)), nil
			}
			return Int(-n), nil
		}
		return v, nil
	}
	return Float(math.Abs(v.Float())), nil
}

func builtinMin(args []Value) (Value, error) {
	if len(args) == 0 {
		return Nil, ErrArgumentCount
	}
	// If single array argument
	if len(args) == 1 && args[0].IsArray() {
		arr := args[0].Array()
		if len(arr) == 0 {
			return Nil, nil
		}
		min := NewValue(arr[0])
		for _, v := range arr[1:] {
			val := NewValue(v)
			if val.Compare(min) < 0 {
				min = val
			}
		}
		return min, nil
	}
	// Multiple arguments
	min := args[0]
	for _, v := range args[1:] {
		if v.Compare(min) < 0 {
			min = v
		}
	}
	return min, nil
}

func builtinMax(args []Value) (Value, error) {
	if len(args) == 0 {
		return Nil, ErrArgumentCount
	}
	// If single array argument
	if len(args) == 1 && args[0].IsArray() {
		arr := args[0].Array()
		if len(arr) == 0 {
			return Nil, nil
		}
		max := NewValue(arr[0])
		for _, v := range arr[1:] {
			val := NewValue(v)
			if val.Compare(max) > 0 {
				max = val
			}
		}
		return max, nil
	}
	// Multiple arguments
	max := args[0]
	for _, v := range args[1:] {
		if v.Compare(max) > 0 {
			max = v
		}
	}
	return max, nil
}

func builtinCeil(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	f := math.Ceil(args[0].Float())
	// Avoid implementation-defined float->int64 overflow for very large magnitudes.
	// If out of int64 range, return float to preserve magnitude.
	const maxInt64Plus1 = 9223372036854775808.0 // 2^63
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Float(f), nil
	}
	if f < float64(math.MinInt64) || f >= maxInt64Plus1 {
		return Float(f), nil
	}
	return Int(int64(f)), nil
}

func builtinFloor(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	f := math.Floor(args[0].Float())
	const maxInt64Plus1 = 9223372036854775808.0 // 2^63
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Float(f), nil
	}
	if f < float64(math.MinInt64) || f >= maxInt64Plus1 {
		return Float(f), nil
	}
	return Int(int64(f)), nil
}

func builtinRound(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	f := math.Round(args[0].Float())
	const maxInt64Plus1 = 9223372036854775808.0 // 2^63
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Float(f), nil
	}
	if f < float64(math.MinInt64) || f >= maxInt64Plus1 {
		return Float(f), nil
	}
	return Int(int64(f)), nil
}

func builtinSqrt(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Float(math.Sqrt(args[0].Float())), nil
}

func builtinPow(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	return Float(math.Pow(args[0].Float(), args[1].Float())), nil
}

func builtinLog(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Float(math.Log(args[0].Float())), nil
}

func builtinExp(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Float(math.Exp(args[0].Float())), nil
}

func builtinSin(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Float(math.Sin(args[0].Float())), nil
}

func builtinCos(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Float(math.Cos(args[0].Float())), nil
}
