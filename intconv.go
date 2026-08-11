package hyperex

import "math"

// intFromInt64Checked converts int64 to int and reports whether the conversion is exact.
// This is important on 32-bit platforms where int is 32-bit and silent truncation would occur.
func intFromInt64Checked(v int64) (int, bool) {
	i := int(v)
	return i, int64(i) == v
}

// float64ToInt64Exact attempts to convert a float64 that represents an integer into int64 safely.
// It avoids implementation-defined behavior for out-of-range conversions.
// Returns (value, ok).
func float64ToInt64Exact(f float64) (int64, bool) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	if math.Trunc(f) != f {
		return 0, false
	}
	// int64 range is [-2^63, 2^63-1]. Note: float64(math.MaxInt64) rounds to 2^63,
	// so we must compare using 2^63 explicitly as an upper exclusive bound.
	const maxInt64Plus1 = 9223372036854775808.0 // 2^63, exactly representable in float64
	if f < float64(math.MinInt64) || f >= maxInt64Plus1 {
		return 0, false
	}
	return int64(f), true
}


