package hyperex

import (
	"fmt"
	"strings"
)

func builtinPrint(args []Value) (Value, error) {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = arg.String()
	}
	fmt.Println(strings.Join(parts, " "))
	return Nil, nil
}

func builtinSprintf(args []Value) (Value, error) {
	if len(args) < 1 {
		return Nil, ErrArgumentCount
	}
	format := args[0].String()
	fmtArgs := make([]any, len(args)-1)
	for i, arg := range args[1:] {
		fmtArgs[i] = arg.Interface()
	}
	return String(fmt.Sprintf(format, fmtArgs...)), nil
}

// clamp(value, min, max) - restricts value to be within [min, max]
func builtinClamp(args []Value) (Value, error) {
	if len(args) != 3 {
		return Nil, ErrArgumentCount
	}
	val := args[0]
	minVal := args[1]
	maxVal := args[2]

	if val.IsInt() && minVal.IsInt() && maxVal.IsInt() {
		v := val.Int()
		min := minVal.Int()
		max := maxVal.Int()
		if v < min {
			return Int(min), nil
		}
		if v > max {
			return Int(max), nil
		}
		return Int(v), nil
	}

	v := val.Float()
	min := minVal.Float()
	max := maxVal.Float()
	if v < min {
		return Float(min), nil
	}
	if v > max {
		return Float(max), nil
	}
	return Float(v), nil
}

// between(value, min, max) - returns true if min <= value <= max
func builtinBetween(args []Value) (Value, error) {
	if len(args) != 3 {
		return Nil, ErrArgumentCount
	}
	val := args[0]
	minVal := args[1]
	maxVal := args[2]

	return Bool(val.Compare(minVal) >= 0 && val.Compare(maxVal) <= 0), nil
}

// empty(value) - returns true if value is nil, empty string, empty array, or empty map
func builtinEmpty(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	v := args[0]
	if v.IsNil() {
		return Bool(true), nil
	}
	if v.IsString() {
		return Bool(v.String() == ""), nil
	}
	if v.IsArray() {
		return Bool(len(v.Array()) == 0), nil
	}
	if v.IsMap() {
		return Bool(len(v.Map()) == 0), nil
	}
	return Bool(false), nil
}

// notEmpty(value) - returns true if value is not empty
func builtinNotEmpty(args []Value) (Value, error) {
	result, err := builtinEmpty(args)
	if err != nil {
		return Nil, err
	}
	return Bool(!result.Bool()), nil
}

// default(value, defaultValue) - returns value if not nil, otherwise defaultValue
func builtinDefault(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	if args[0].IsNil() {
		return args[1], nil
	}
	return args[0], nil
}

// coalesce(values...) - returns first non-nil value
func builtinCoalesce(args []Value) (Value, error) {
	for _, arg := range args {
		if !arg.IsNil() {
			return arg, nil
		}
	}
	return Nil, nil
}

// truthy(array) - removes all falsy values (nil, false, 0, "") from array
func builtinTruthy(vm *execVM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	result := make([]any, 0, len(arr))
	for i, v := range arr {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		val := NewValue(v)
		if !val.IsNil() && val.Bool() {
			result = append(result, v)
		}
	}
	return Array(result), nil
}
