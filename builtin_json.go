package hyperex

import (
	"encoding/json"
)

// toJSON(value) - converts value to JSON string
func builtinToJSON(vm *execVM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if vm != nil && vm.limits != nil {
		if err := vm.checkLimits(); err != nil {
			return Nil, err
		}
	}
	v := args[0].Interface()
	data, err := json.Marshal(v)
	if err != nil {
		return Nil, &RuntimeError{Message: "failed to marshal to JSON: " + err.Error()}
	}
	if vm != nil && vm.limits != nil {
		vm.opsCount += int64(len(data)) // rough accounting
		if err := vm.checkLimits(); err != nil {
			return Nil, err
		}
	}
	return String(string(data)), nil
}

// fromJSON(string) - parses JSON string to value
// Note: JSON numbers that are whole numbers are converted to int64 for consistency
func builtinFromJSON(vm *execVM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	s := args[0].String()
	if vm != nil && vm.limits != nil {
		vm.opsCount += int64(len(s)) // rough accounting
		if err := vm.checkLimits(); err != nil {
			return Nil, err
		}
	}
	var result any
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return Nil, &RuntimeError{Message: "failed to parse JSON: " + err.Error()}
	}
	// Convert float64 to int64 where appropriate (JSON numbers are always float64).
	// We only convert integers within the IEEE-754 "safe integer" range (|n| <= 2^53-1)
	// to avoid inventing incorrect int64 values from imprecise float64 representations.
	result = convertJSONNumbers(result)
	return NewValue(result), nil
}

// convertJSONNumbers recursively converts whole float64 JSON numbers to int64
// when they fall in the IEEE-754 safe-integer range.
func convertJSONNumbers(v any) any {
	switch val := v.(type) {
	case float64:
		// Convert only safe integers.
		// (float64 cannot precisely represent every int64; converting beyond 2^53 risks corruption.)
		const maxSafeInt = 9007199254740991.0  // 2^53 - 1
		const minSafeInt = -9007199254740991.0 // -(2^53 - 1)
		if val < minSafeInt || val > maxSafeInt {
			return val
		}
		if i, ok := float64ToInt64Exact(val); ok {
			return i
		}
		return val
	case []any:
		for i, elem := range val {
			val[i] = convertJSONNumbers(elem)
		}
		return val
	case map[string]any:
		for k, elem := range val {
			val[k] = convertJSONNumbers(elem)
		}
		return val
	default:
		return v
	}
}
