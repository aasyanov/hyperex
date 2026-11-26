package hyperex

import (
	"fmt"
	"math"
)

func builtinKeys(vm *execVM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsMap() {
		return Nil, ErrInvalidType
	}
	m := args[0].Map()
	// Deterministic order: sort keys (Go map iteration order is unspecified).
	keys := make([]string, 0, len(m))
	i := 0
	for k := range m {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		i++
		keys = append(keys, k)
	}
	sortStrings(keys)
	out := make([]any, len(keys))
	for i, k := range keys {
		out[i] = k
	}
	return Array(out), nil
}

func builtinValues(vm *execVM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsMap() {
		return Nil, ErrInvalidType
	}
	m := args[0].Map()
	// Emit values in the same sorted key order as keys() for determinism.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	values := make([]any, len(keys))
	for i, k := range keys {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		values[i] = m[k]
	}
	return Array(values), nil
}

// hasKey(map, key) - returns true if map contains the key
func builtinHasKey(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsMap() {
		return Nil, ErrInvalidType
	}
	m := args[0].Map()
	key := args[1].String()
	_, ok := m[key]
	return Bool(ok), nil
}

// repeat(value, count) - creates array with value repeated count times
func builtinRepeat(vm *execVM, args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	val := args[0]
	count64 := args[1].Int()
	if count64 < 0 {
		count64 = 0
	}
	// Limit to prevent OOM
	const maxRepeatSize = 1_000_000
	if count64 > maxRepeatSize {
		return Nil, &RuntimeError{Message: fmt.Sprintf("repeat count %d exceeds maximum %d", count64, maxRepeatSize)}
	}
	count := int(count64) // safe: <= 1_000_000
	result := make([]any, count)
	for i := 0; i < count; i++ {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		result[i] = val.Interface()
	}
	return Array(result), nil
}

// range(start, end, step?) - creates array of numbers from start to end
func builtinRange(vm *execVM, args []Value) (Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return Nil, ErrArgumentCount
	}
	start := args[0].Int()
	end := args[1].Int()
	step := int64(1)
	if len(args) == 3 {
		step = args[2].Int()
		if step == 0 {
			return Nil, &RuntimeError{Message: "step cannot be zero"}
		}
	}
	// Special-case: -MinInt64 overflows for int64.
	if step == math.MinInt64 {
		return Nil, &RuntimeError{Message: "step too small"}
	}

	// Calculate range size and limit to prevent DoS
	const maxRangeSize int64 = 1_000_000
	var rangeSize int64

	// Correctly calculate range size based on direction with overflow protection
	if step > 0 {
		if start > end {
			// Positive step but start > end: empty range
			rangeSize = 0
		} else {
			// Safe calculation: (end - start) / step + 1
			diff := end - start
			if diff < 0 {
				// Overflow occurred
				return Nil, &RuntimeError{Message: errRangeOverflow}
			}
			// Check if diff/step would exceed maxRangeSize before computing
			if diff/step >= maxRangeSize {
				return Nil, &RuntimeError{Message: fmt.Sprintf("range size exceeds maximum %d", maxRangeSize)}
			}
			rangeSize = diff/step + 1
		}
	} else {
		if start < end {
			// Negative step but start < end: empty range
			rangeSize = 0
		} else {
			// Safe calculation: (start - end) / (-step) + 1
			diff := start - end
			if diff < 0 {
				// Overflow occurred
				return Nil, &RuntimeError{Message: errRangeOverflow}
			}
			negStep := -step
			// Check if diff/negStep would exceed maxRangeSize before computing
			if diff/negStep >= maxRangeSize {
				return Nil, &RuntimeError{Message: fmt.Sprintf("range size exceeds maximum %d", maxRangeSize)}
			}
			rangeSize = diff/negStep + 1
		}
	}

	if rangeSize > maxRangeSize {
		return Nil, &RuntimeError{Message: fmt.Sprintf("range size %d exceeds maximum %d", rangeSize, maxRangeSize)}
	}

	// Use a counted loop to avoid int64 overflow in i += step (e.g. end==MaxInt64, step>0).
	result := make([]any, 0, rangeSize)
	cur := start
	for k := int64(0); k < rangeSize; k++ {
		if vm != nil {
			if err := vm.sandboxLoopCheck(int(k)); err != nil {
				return Nil, err
			}
		}
		result = append(result, cur)
		// Advance except after last element
		if k == rangeSize-1 {
			break
		}
		// Guard overflow on cur += step (should not happen when rangeSize computed correctly, but be safe).
		if step > 0 {
			if cur > math.MaxInt64-step {
				return Nil, &RuntimeError{Message: errRangeOverflow}
			}
		} else {
			if cur < math.MinInt64-step {
				return Nil, &RuntimeError{Message: errRangeOverflow}
			}
		}
		cur += step
	}
	return Array(result), nil
}

// zip(arr1, arr2, ...) - combines arrays into array of tuples
func builtinZip(vm *execVM, args []Value) (Value, error) {
	if len(args) < 2 {
		return Nil, ErrArgumentCount
	}

	// Find minimum length
	minLen := -1
	arrays := make([][]any, len(args))
	for i, arg := range args {
		if !arg.IsArray() {
			return Nil, ErrInvalidType
		}
		arrays[i] = arg.Array()
		if minLen == -1 || len(arrays[i]) < minLen {
			minLen = len(arrays[i])
		}
	}

	result := make([]any, minLen)
	for i := 0; i < minLen; i++ {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		tuple := make([]any, len(arrays))
		for j, arr := range arrays {
			tuple[j] = arr[i]
		}
		result[i] = tuple
	}
	return Array(result), nil
}

// chunk(array, size) - splits array into chunks of given size
func builtinChunk(vm *execVM, args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	size64 := args[1].Int()
	size, ok := intFromInt64Checked(size64)
	if !ok {
		return Nil, &RuntimeError{Message: "chunk size out of range"}
	}
	if size <= 0 {
		return Nil, &RuntimeError{Message: "chunk size must be positive"}
	}

	numChunks := 0
	if len(arr) > 0 {
		numChunks = 1 + (len(arr)-1)/size
	}
	const maxChunks = 1_000_000
	if numChunks > maxChunks {
		return Nil, &RuntimeError{Message: fmt.Sprintf("chunk would create %d chunks, exceeds maximum %d", numChunks, maxChunks)}
	}

	result := make([]any, 0, numChunks)
	iter := 0
	for start := 0; start < len(arr); {
		if vm != nil {
			if err := vm.sandboxLoopCheck(iter); err != nil {
				return Nil, err
			}
		}
		iter++
		end := len(arr)
		if size < len(arr)-start {
			end = start + size
		}
		chunk := make([]any, end-start)
		copy(chunk, arr[start:end])
		result = append(result, chunk)
		start = end
	}
	return Array(result), nil
}

// compact(array) - removes nil values from array
// Use truthy() to remove all falsy values (nil, false, 0, "")
func builtinCompact(vm *execVM, args []Value) (Value, error) {
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
		if !val.IsNil() {
			result = append(result, v)
		}
	}
	return Array(result), nil
}

// pluck(array, key) - extracts values of a key from array of maps
func builtinPluck(vm *execVM, args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	key := args[1].String()

	result := make([]any, 0, len(arr))
	for i, item := range arr {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		itemVal := NewValue(item)
		if itemVal.IsMap() {
			if v, ok := itemVal.Map()[key]; ok {
				result = append(result, v)
			}
		}
	}
	return Array(result), nil
}

// groupBy(array, keyFunc) - groups array elements by key returned by keyFunc
// Returns array of objects {key: string, items: array} for easy iteration
// Example: groupBy(users, u => u.role) | map(g => {"role": g.key, "count": len(g.items)})
func builtinGroupBy(vm *execVM, args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	if args[1].kind != kindFunc {
		return Nil, ErrInvalidType
	}

	arr := args[0].Array()
	fn := args[1]

	// Group items by key
	groups := make(map[string][]any)
	order := make([]string, 0) // preserve insertion order

	for i, item := range arr {
		if err := vm.sandboxLoopCheck(i); err != nil {
			return Nil, err
		}
		// Call the key function
		keyResult, err := vm.call(fn, []Value{NewValue(item)})
		if err != nil {
			return Nil, err
		}
		key := keyResult.String()

		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}
		groups[key] = append(groups[key], item)
	}

	// Convert to array of {key, items} objects
	result := make([]any, len(order))
	for i, key := range order {
		result[i] = map[string]any{
			fieldKey: key,
			"items":  groups[key],
		}
	}

	return Array(result), nil
}

// sortBy(array, keyFunc, desc?) - sorts array by key returned by keyFunc
// keyFunc receives each element and returns a comparable value (number or string)
// desc is optional boolean for descending order (default: false = ascending)
func builtinSortBy(vm *execVM, args []Value) (Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	if args[1].kind != kindFunc {
		return Nil, ErrInvalidType
	}

	arr := args[0].Array()
	fn := args[1]
	desc := false
	if len(args) == 3 {
		desc = args[2].Bool()
	}

	// Create slice with computed keys for sorting
	type keyedItem struct {
		item any
		key  Value
	}
	keyed := make([]keyedItem, len(arr))

	for i, item := range arr {
		if err := vm.sandboxLoopCheck(i); err != nil {
			return Nil, err
		}
		keyResult, err := vm.call(fn, []Value{NewValue(item)})
		if err != nil {
			return Nil, err
		}
		keyed[i] = keyedItem{item: item, key: keyResult}
	}

	// Best-effort sandbox check for potentially expensive sort.
	if vm != nil && vm.limits != nil {
		vm.opsCount += int64(len(keyed)) // rough accounting
		if err := vm.checkLimits(); err != nil {
			return Nil, err
		}
	}
	// Sort using computed keys. Equal keys must return false in both directions.
	sortSlice(keyed, func(i, j int) bool {
		ki, kj := keyed[i].key, keyed[j].key
		if desc {
			return ki.Compare(kj) > 0
		}
		return ki.Compare(kj) < 0
	})

	// Extract sorted items
	result := make([]any, len(keyed))
	for i, k := range keyed {
		result[i] = k.item
	}

	return Array(result), nil
}
