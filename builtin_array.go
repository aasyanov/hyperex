package hyperex

import "fmt"

// maxFlattenDepth caps recursive flatten nesting (stack-safety).
const maxFlattenDepth = 100

// Array builtins
func builtinFirst(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	if len(arr) == 0 {
		return Nil, nil
	}
	return NewValue(arr[0]), nil
}

func builtinLast(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	if len(arr) == 0 {
		return Nil, nil
	}
	return NewValue(arr[len(arr)-1]), nil
}

func builtinTake(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	n64 := args[1].Int()
	if n64 < 0 {
		n64 = 0
	}
	if n64 > int64(len(arr)) {
		n64 = int64(len(arr))
	}
	n := int(n64)
	return Array(arr[:n]), nil
}

func builtinSkip(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	n64 := args[1].Int()
	if n64 < 0 {
		n64 = 0
	}
	if n64 > int64(len(arr)) {
		n64 = int64(len(arr))
	}
	n := int(n64)
	return Array(arr[n:]), nil
}

func builtinReverse(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	result := make([]any, len(arr))
	for i, v := range arr {
		result[len(arr)-1-i] = v
	}
	return Array(result), nil
}

func builtinSort(vm *execVM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	result := make([]any, len(arr))
	copy(result, arr)
	// Best-effort sandbox check for potentially expensive sort.
	if vm != nil && vm.limits != nil {
		vm.opsCount += int64(len(result)) // rough accounting
		if err := vm.checkLimits(); err != nil {
			return Nil, err
		}
	}
	sortSlice(result, func(i, j int) bool {
		return NewValue(result[i]).Compare(NewValue(result[j])) < 0
	})
	return Array(result), nil
}

// uniqueKey generates a type-aware key for deduplication.
// This prevents collisions between different types with the same string representation.
// For example: int(1) vs string("1") vs bool(true) will have different keys.
type uniqueKey struct {
	typ string
	val string
}

func builtinUnique(vm *execVM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	seen := make(map[uniqueKey]bool)
	result := make([]any, 0, len(arr))
	for i, v := range arr {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		val := NewValue(v)
		// Use deterministic, cycle-safe key (sandbox-aware).
		keyStr, err := stableValueKey(vm, val, 0, make(map[uintptr]bool, 8))
		if err != nil {
			return Nil, err
		}
		key := uniqueKey{typ: val.Type(), val: keyStr}
		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}
	return Array(result), nil
}

func builtinConcat(vm *execVM, args []Value) (Value, error) {
	result := make([]any, 0)
	iter := 0
	for _, arg := range args {
		if vm != nil {
			if err := vm.sandboxLoopCheck(iter); err != nil {
				return Nil, err
			}
		}
		iter++
		if arg.IsArray() {
			arr := arg.Array()
			for i := range arr {
				if vm != nil {
					if err := vm.sandboxLoopCheck(iter); err != nil {
						return Nil, err
					}
				}
				iter++
				result = append(result, arr[i])
			}
		} else {
			result = append(result, arg.Interface())
		}
	}
	return Array(result), nil
}

func builtinFlatten(vm *execVM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	result, err := flattenArrayWithDepth(vm, args[0].Array(), 0, maxFlattenDepth)
	if err != nil {
		return Nil, err
	}
	return Array(result), nil
}

// flattenArrayWithDepth flattens nested arrays with depth limit to prevent stack overflow
func flattenArrayWithDepth(vm *execVM, arr []any, currentDepth, maxDepth int) ([]any, error) {
	if currentDepth > maxDepth {
		return nil, &RuntimeError{Message: fmt.Sprintf("flatten: maximum nesting depth %d exceeded", maxDepth)}
	}
	result := make([]any, 0, len(arr))
	for i, v := range arr {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return nil, err
			}
		}
		if inner, ok := v.([]any); ok {
			flattened, err := flattenArrayWithDepth(vm, inner, currentDepth+1, maxDepth)
			if err != nil {
				return nil, err
			}
			result = append(result, flattened...)
		} else {
			result = append(result, v)
		}
	}
	return result, nil
}
