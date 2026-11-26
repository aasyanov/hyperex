package hyperex

import "math"

// Aggregation builtins
func builtinSum(vm *execVM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()

	// Sum integers separately to preserve precision for large int64 values
	// Switch to float on overflow (consistent with + operator behavior)
	var intSum int64
	var floatSum float64
	allInts := true

	for i, v := range arr {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		val := NewValue(v)
		if val.IsInt() && allInts {
			n := val.Int()
			// Check for overflow before adding (same logic as addOverflows in vm)
			if (n > 0 && intSum > math.MaxInt64-n) || (n < 0 && intSum < math.MinInt64-n) {
				// Overflow: switch to float
				floatSum = float64(intSum) + float64(n)
				allInts = false
			} else {
				intSum += n
			}
		} else {
			if allInts {
				// First non-int or overflow: convert accumulated int sum to float
				floatSum = float64(intSum)
				allInts = false
			}
			floatSum += val.Float()
		}
	}

	if allInts {
		return Int(intSum), nil
	}
	return Float(floatSum), nil
}

func builtinAvg(vm *execVM, args []Value) (Value, error) {
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
	var sum float64
	for i, v := range arr {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		sum += NewValue(v).Float()
	}
	return Float(sum / float64(len(arr))), nil
}

func builtinCount(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Int(int64(args[0].Len())), nil
}

// parseAggPredArgs validates all/any/none/one arity and types.
// Form 1: (arr) — truthiness of elements.
// Form 2: (arr, pred) — pred must be kindFunc; called as pred(elem) / pred(elem, index).
func parseAggPredArgs(args []Value) (arr []any, pred Value, hasPred bool, err error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, Nil, false, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return nil, Nil, false, ErrInvalidType
	}
	if len(args) == 2 {
		if args[1].kind != kindFunc {
			return nil, Nil, false, ErrInvalidType
		}
		return args[0].Array(), args[1], true, nil
	}
	return args[0].Array(), Nil, false, nil
}

// aggElemMatches returns whether an element "matches" for aggregation:
// truthiness of the element, or truthiness of pred(elem[, index]).
func aggElemMatches(vm *execVM, elem any, index int, pred Value, hasPred bool) (bool, error) {
	if !hasPred {
		return NewValue(elem).Bool(), nil
	}
	val, err := vm.call(pred, []Value{NewValue(elem), Int(int64(index))})
	if err != nil {
		return false, err
	}
	return val.Bool(), nil
}

func builtinAll(vm *execVM, args []Value) (Value, error) {
	arr, pred, hasPred, err := parseAggPredArgs(args)
	if err != nil {
		return Nil, err
	}
	for i, v := range arr {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		ok, err := aggElemMatches(vm, v, i, pred, hasPred)
		if err != nil {
			return Nil, err
		}
		if !ok {
			return Bool(false), nil
		}
	}
	return Bool(true), nil
}

func builtinAny(vm *execVM, args []Value) (Value, error) {
	arr, pred, hasPred, err := parseAggPredArgs(args)
	if err != nil {
		return Nil, err
	}
	for i, v := range arr {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		ok, err := aggElemMatches(vm, v, i, pred, hasPred)
		if err != nil {
			return Nil, err
		}
		if ok {
			return Bool(true), nil
		}
	}
	return Bool(false), nil
}

func builtinNone(vm *execVM, args []Value) (Value, error) {
	arr, pred, hasPred, err := parseAggPredArgs(args)
	if err != nil {
		return Nil, err
	}
	for i, v := range arr {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		ok, err := aggElemMatches(vm, v, i, pred, hasPred)
		if err != nil {
			return Nil, err
		}
		if ok {
			return Bool(false), nil
		}
	}
	return Bool(true), nil
}

func builtinOne(vm *execVM, args []Value) (Value, error) {
	arr, pred, hasPred, err := parseAggPredArgs(args)
	if err != nil {
		return Nil, err
	}
	count := 0
	for i, v := range arr {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		ok, err := aggElemMatches(vm, v, i, pred, hasPred)
		if err != nil {
			return Nil, err
		}
		if ok {
			count++
			if count > 1 {
				return Bool(false), nil
			}
		}
	}
	return Bool(count == 1), nil
}

// Higher-order functions (simple implementations)
func builtinMap(vm *execVM, args []Value) (Value, error) {
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
	result := make([]any, len(arr))

	for i, elem := range arr {
		if err := vm.sandboxLoopCheck(i); err != nil {
			return Nil, err
		}
		val, err := vm.call(fn, []Value{NewValue(elem), Int(int64(i))})
		if err != nil {
			return Nil, err
		}
		result[i] = val.Interface()
	}
	return Array(result), nil
}

func builtinFilter(vm *execVM, args []Value) (Value, error) {
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
	result := make([]any, 0)

	for i, elem := range arr {
		if err := vm.sandboxLoopCheck(i); err != nil {
			return Nil, err
		}
		val, err := vm.call(fn, []Value{NewValue(elem), Int(int64(i))})
		if err != nil {
			return Nil, err
		}
		if val.Bool() {
			result = append(result, elem)
		}
	}
	return Array(result), nil
}

func builtinReduce(vm *execVM, args []Value) (Value, error) {
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

	if len(arr) == 0 {
		if len(args) == 3 {
			return args[2], nil
		}
		return Nil, nil
	}

	var acc Value
	startIdx := 0
	if len(args) == 3 {
		acc = args[2]
	} else {
		acc = NewValue(arr[0])
		startIdx = 1
	}

	for i := startIdx; i < len(arr); i++ {
		if err := vm.sandboxLoopCheck(i); err != nil {
			return Nil, err
		}
		val, err := vm.call(fn, []Value{acc, NewValue(arr[i]), Int(int64(i))})
		if err != nil {
			return Nil, err
		}
		acc = val
	}
	return acc, nil
}

func builtinFind(vm *execVM, args []Value) (Value, error) {
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

	for i, elem := range arr {
		if err := vm.sandboxLoopCheck(i); err != nil {
			return Nil, err
		}
		val, err := vm.call(fn, []Value{NewValue(elem), Int(int64(i))})
		if err != nil {
			return Nil, err
		}
		if val.Bool() {
			return NewValue(elem), nil
		}
	}
	return Nil, nil
}

func builtinFindIndex(vm *execVM, args []Value) (Value, error) {
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

	for i, elem := range arr {
		if err := vm.sandboxLoopCheck(i); err != nil {
			return Nil, err
		}
		val, err := vm.call(fn, []Value{NewValue(elem), Int(int64(i))})
		if err != nil {
			return Nil, err
		}
		if val.Bool() {
			return Int(int64(i)), nil
		}
	}
	return Int(-1), nil
}
