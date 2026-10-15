package hyperex

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// builtinFunc represents a built-in function.
type builtinFunc struct {
	name string
	fn   func(*execVM, []Value) (Value, error)
}

// wrapBuiltin adapts a VM-free builtin to the vm-aware builtinFunc signature.
func wrapBuiltin(fn func([]Value) (Value, error)) func(*execVM, []Value) (Value, error) {
	return func(_ *execVM, args []Value) (Value, error) {
		return fn(args)
	}
}

// defaultBuiltinNameSet holds the keys of defaultBuiltins for O(1) name checks
// (pipe vs bitwise OR heuristic, analysis warnings). Built once at init.
var defaultBuiltinNameSet = func() map[string]struct{} {
	m := make(map[string]struct{}, 100)
	for name := range defaultBuiltins() {
		m[name] = struct{}{}
	}
	return m
}()

var builtinNamesSorted = func() []string {
	names := make([]string, 0, len(defaultBuiltinNameSet))
	for name := range defaultBuiltinNameSet {
		names = append(names, name)
	}
	sortStrings(names)
	return names
}()

// isDefaultBuiltinName reports whether name is a key in the default builtin set.
func isDefaultBuiltinName(name string) bool {
	_, ok := defaultBuiltinNameSet[name]
	return ok
}

// BuiltinNames returns a sorted copy of all default built-in function names.
func BuiltinNames() []string {
	out := make([]string, len(builtinNamesSorted))
	copy(out, builtinNamesSorted)
	return out
}

// IsBuiltin reports whether name is a default built-in function.
func IsBuiltin(name string) bool {
	return isDefaultBuiltinName(name)
}

// defaultBuiltins returns the default set of built-in functions.
func defaultBuiltins() map[string]Value {
	m := map[string]Value{
		// Math functions
		nameAbs:   Func(&builtinFunc{name: nameAbs, fn: wrapBuiltin(builtinAbs)}),
		nameMin:   Func(&builtinFunc{name: nameMin, fn: wrapBuiltin(builtinMin)}),
		nameMax:   Func(&builtinFunc{name: nameMax, fn: wrapBuiltin(builtinMax)}),
		nameCeil:  Func(&builtinFunc{name: nameCeil, fn: wrapBuiltin(builtinCeil)}),
		nameFloor: Func(&builtinFunc{name: nameFloor, fn: wrapBuiltin(builtinFloor)}),
		nameRound: Func(&builtinFunc{name: nameRound, fn: wrapBuiltin(builtinRound)}),
		nameSqrt:  Func(&builtinFunc{name: nameSqrt, fn: wrapBuiltin(builtinSqrt)}),
		namePow:   Func(&builtinFunc{name: namePow, fn: wrapBuiltin(builtinPow)}),
		nameLog:   Func(&builtinFunc{name: nameLog, fn: wrapBuiltin(builtinLog)}),
		nameExp:   Func(&builtinFunc{name: nameExp, fn: wrapBuiltin(builtinExp)}),
		nameSin:   Func(&builtinFunc{name: nameSin, fn: wrapBuiltin(builtinSin)}),
		nameCos:   Func(&builtinFunc{name: nameCos, fn: wrapBuiltin(builtinCos)}),

		// String functions
		nameLen:        Func(&builtinFunc{name: nameLen, fn: wrapBuiltin(builtinLen)}),
		nameUpper:      Func(&builtinFunc{name: nameUpper, fn: wrapBuiltin(builtinUpper)}),
		nameLower:      Func(&builtinFunc{name: nameLower, fn: wrapBuiltin(builtinLower)}),
		nameTrim:       Func(&builtinFunc{name: nameTrim, fn: wrapBuiltin(builtinTrim)}),
		nameTrimLeft:   Func(&builtinFunc{name: nameTrimLeft, fn: wrapBuiltin(builtinTrimLeft)}),
		nameTrimRight:  Func(&builtinFunc{name: nameTrimRight, fn: wrapBuiltin(builtinTrimRight)}),
		nameSplit:      Func(&builtinFunc{name: nameSplit, fn: wrapBuiltin(builtinSplit)}),
		nameJoin:       Func(&builtinFunc{name: nameJoin, fn: wrapBuiltin(builtinJoin)}),
		nameContains:   Func(&builtinFunc{name: nameContains, fn: wrapBuiltin(builtinContains)}),
		nameStartsWith: Func(&builtinFunc{name: nameStartsWith, fn: wrapBuiltin(builtinStartsWith)}),
		nameEndsWith:   Func(&builtinFunc{name: nameEndsWith, fn: wrapBuiltin(builtinEndsWith)}),
		nameReplace:    Func(&builtinFunc{name: nameReplace, fn: wrapBuiltin(builtinReplace)}),
		nameSubstr:     Func(&builtinFunc{name: nameSubstr, fn: wrapBuiltin(builtinSubstr)}),
		nameIndexOf:    Func(&builtinFunc{name: nameIndexOf, fn: wrapBuiltin(builtinIndexOf)}),

		// Array functions
		nameFirst:   Func(&builtinFunc{name: nameFirst, fn: wrapBuiltin(builtinFirst)}),
		nameLast:    Func(&builtinFunc{name: nameLast, fn: wrapBuiltin(builtinLast)}),
		nameTake:    Func(&builtinFunc{name: nameTake, fn: wrapBuiltin(builtinTake)}),
		nameSkip:    Func(&builtinFunc{name: nameSkip, fn: wrapBuiltin(builtinSkip)}),
		nameReverse: Func(&builtinFunc{name: nameReverse, fn: wrapBuiltin(builtinReverse)}),
		nameSort:    Func(&builtinFunc{name: nameSort, fn: builtinSort}),
		nameUnique:  Func(&builtinFunc{name: nameUnique, fn: builtinUnique}),
		nameConcat:  Func(&builtinFunc{name: nameConcat, fn: builtinConcat}),
		nameFlatten: Func(&builtinFunc{name: nameFlatten, fn: builtinFlatten}),

		// Aggregation functions
		nameSum:   Func(&builtinFunc{name: nameSum, fn: builtinSum}),
		nameAvg:   Func(&builtinFunc{name: nameAvg, fn: builtinAvg}),
		nameCount: Func(&builtinFunc{name: nameCount, fn: wrapBuiltin(builtinCount)}),
		nameAll:   Func(&builtinFunc{name: nameAll, fn: builtinAll}),
		nameAny:   Func(&builtinFunc{name: nameAny, fn: builtinAny}),
		nameNone:  Func(&builtinFunc{name: nameNone, fn: builtinNone}),
		nameOne:   Func(&builtinFunc{name: nameOne, fn: builtinOne}),

		// Higher-order functions (simple versions)
		nameMap:       Func(&builtinFunc{name: nameMap, fn: builtinMap}),
		nameFilter:    Func(&builtinFunc{name: nameFilter, fn: builtinFilter}),
		nameReduce:    Func(&builtinFunc{name: nameReduce, fn: builtinReduce}),
		nameFind:      Func(&builtinFunc{name: nameFind, fn: builtinFind}),
		nameFindIndex: Func(&builtinFunc{name: nameFindIndex, fn: builtinFindIndex}),

		// Type functions
		nameInt:    Func(&builtinFunc{name: nameInt, fn: wrapBuiltin(builtinInt)}),
		nameFloat:  Func(&builtinFunc{name: nameFloat, fn: wrapBuiltin(builtinFloat)}),
		nameString: Func(&builtinFunc{name: nameString, fn: wrapBuiltin(builtinString)}),
		nameBool:   Func(&builtinFunc{name: nameBool, fn: wrapBuiltin(builtinBool)}),
		nameType:   Func(&builtinFunc{name: nameType, fn: wrapBuiltin(builtinType)}),
		nameIsNil:  Func(&builtinFunc{name: nameIsNil, fn: wrapBuiltin(builtinIsNil)}),

		// Utility / map / scalar helpers
		namePrint:   Func(&builtinFunc{name: namePrint, fn: wrapBuiltin(builtinPrint)}),
		nameSprintf: Func(&builtinFunc{name: nameSprintf, fn: wrapBuiltin(builtinSprintf)}),
		nameKeys:    Func(&builtinFunc{name: nameKeys, fn: builtinKeys}),
		nameValues:  Func(&builtinFunc{name: nameValues, fn: builtinValues}),

		nameClamp:    Func(&builtinFunc{name: nameClamp, fn: wrapBuiltin(builtinClamp)}),
		nameBetween:  Func(&builtinFunc{name: nameBetween, fn: wrapBuiltin(builtinBetween)}),
		nameHasKey:   Func(&builtinFunc{name: nameHasKey, fn: wrapBuiltin(builtinHasKey)}),
		nameEmpty:    Func(&builtinFunc{name: nameEmpty, fn: wrapBuiltin(builtinEmpty)}),
		nameNotEmpty: Func(&builtinFunc{name: nameNotEmpty, fn: wrapBuiltin(builtinNotEmpty)}),
		nameDefault:  Func(&builtinFunc{name: nameDefault, fn: wrapBuiltin(builtinDefault)}),
		nameCoalesce: Func(&builtinFunc{name: nameCoalesce, fn: wrapBuiltin(builtinCoalesce)}),
		nameRepeat:   Func(&builtinFunc{name: nameRepeat, fn: builtinRepeat}),
		nameRange:    Func(&builtinFunc{name: nameRange, fn: builtinRange}),
		nameZip:      Func(&builtinFunc{name: nameZip, fn: builtinZip}),
		nameChunk:    Func(&builtinFunc{name: nameChunk, fn: builtinChunk}),
		nameCompact:  Func(&builtinFunc{name: nameCompact, fn: builtinCompact}),
		nameTruthy:   Func(&builtinFunc{name: nameTruthy, fn: builtinTruthy}),
		namePluck:    Func(&builtinFunc{name: namePluck, fn: builtinPluck}),
		nameGroupBy:  Func(&builtinFunc{name: nameGroupBy, fn: builtinGroupBy}),
		nameSortBy:   Func(&builtinFunc{name: nameSortBy, fn: builtinSortBy}),
		nameToJSON:   Func(&builtinFunc{name: nameToJSON, fn: builtinToJSON}),
		nameFromJSON: Func(&builtinFunc{name: nameFromJSON, fn: builtinFromJSON}),
	}
	return mergeBuiltinMaps(m, timeDurationBuiltins(), qualityBuiltins())
}

func mergeBuiltinMaps(dst map[string]Value, extras ...map[string]Value) map[string]Value {
	for _, m := range extras {
		for k, v := range m {
			dst[k] = v
		}
	}
	return dst
}

// stableValueKey builds a deterministic, cycle-safe key for values (used by unique()).
// It is sandbox-aware: long traversals account ops via sandboxLoopCheck.
func stableValueKey(vm *execVM, v Value, depth int, seen map[uintptr]bool) (string, error) {
	if depth > 50 {
		return "<depth>", nil
	}
	switch v.kind {
	case kindNil:
		return litNil, nil
	case kindBool:
		if v.bits != 0 {
			return litTrue, nil
		}
		return litFalse, nil
	case kindInt:
		return strconv.FormatInt(int64(v.bits), 10), nil
	case kindFloat:
		return strconv.FormatFloat(math.Float64frombits(v.bits), 'g', -1, 64), nil
	case kindTime:
		return "t:" + strconv.FormatInt(int64(v.bits), 10), nil
	case kindDuration:
		return "d:" + strconv.FormatInt(int64(v.bits), 10), nil
	case kindString:
		s := v.ptr.(string)
		if vm != nil && vm.limits != nil {
			vm.opsCount += int64(len(s))
			if err := vm.checkLimits(); err != nil {
				return "", err
			}
		}
		return "s:" + s, nil
	case kindArray:
		arr := v.ptr.([]any)
		rv := reflect.ValueOf(arr)
		if rv.IsValid() {
			p := rv.Pointer()
			if p != 0 {
				if seen[p] {
					return litCycle, nil
				}
				seen[p] = true
				defer delete(seen, p)
			}
		}
		var b strings.Builder
		b.WriteByte('[')
		for i := range arr {
			if vm != nil {
				if err := vm.sandboxLoopCheck(i); err != nil {
					return "", err
				}
			}
			if i > 0 {
				b.WriteByte(',')
			}
			s, err := stableValueKey(vm, NewValue(arr[i]), depth+1, seen)
			if err != nil {
				return "", err
			}
			b.WriteString(s)
		}
		b.WriteByte(']')
		return b.String(), nil
	case kindMap:
		m := v.ptr.(map[string]any)
		rv := reflect.ValueOf(m)
		if rv.IsValid() {
			p := rv.Pointer()
			if p != 0 {
				if seen[p] {
					return litCycle, nil
				}
				seen[p] = true
				defer delete(seen, p)
			}
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sortStrings(keys)
		var b strings.Builder
		b.WriteByte('{')
		for i, k := range keys {
			if vm != nil {
				if err := vm.sandboxLoopCheck(i); err != nil {
					return "", err
				}
			}
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(k)
			b.WriteByte(':')
			s, err := stableValueKey(vm, NewValue(m[k]), depth+1, seen)
			if err != nil {
				return "", err
			}
			b.WriteString(s)
		}
		b.WriteByte('}')
		return b.String(), nil
	default:
		// Best-effort for external Go values.
		s := safeSprint(v.ptr)
		if vm != nil && vm.limits != nil {
			vm.opsCount += int64(len(s))
			if err := vm.checkLimits(); err != nil {
				return "", err
			}
		}
		return "x:" + s, nil
	}
}

func callBuiltinFunc(vm *execVM, fn Value, args []Value) (Value, error) {
	if fn.kind != kindFunc {
		return Nil, ErrNotCallable
	}

	// Check if it's a closure (lambda)
	if closure, ok := fn.ptr.(*closure); ok {
		return callClosureFromBuiltin(closure, args)
	}

	// Check if it's a builtinFunc
	if bf, ok := fn.ptr.(*builtinFunc); ok {
		return bf.fn(vm, args)
	}

	// Native Go function - use reflection
	return callNativeFromBuiltin(fn.ptr, args)
}

// callClosureFromBuiltin executes a closure without requiring a full vm context
func callClosureFromBuiltin(closure *closure, args []Value) (Value, error) {
	// Use closure's builtins (includes custom functions) if available, otherwise default
	builtins := closure.builtins
	if builtins == nil {
		builtins = defaultBuiltins()
	}

	// Create vm with sandbox limits if closure has them
	var vm *execVM
	if closure.limits != nil {
		vm = newVMWithLimits(closure.code, builtins, closure.limits)
	} else {
		vm = newVM(closure.code, builtins)
	}

	// Inherit strict mode and numeric/non-finite policy from the closure's originating vm
	vm.strictMode = closure.strictMode
	vm.numericMode = closure.numericMode
	vm.nonFiniteMode = closure.nonFiniteMode

	// Inherit maxDepth from the closure's originating vm when available.
	// If not set, fall back to a safe default to prevent infinite recursion.
	if closure.maxDepth > 0 {
		vm.maxDepth = closure.maxDepth
	} else if vm.maxDepth == 0 {
		vm.maxDepth = defaultMaxCallDepth
	}
	defer releaseVM(vm)

	// Copy closure environment
	for k, v := range closure.env {
		vm.env[k] = v
	}

	// Bind arguments by position and by name
	for i, arg := range args {
		vm.env[fmt.Sprintf("$%d", i)] = arg
		if i < len(closure.params) {
			vm.env[closure.params[i]] = arg
		}
	}

	// Special: first argument is also available as "#"
	if len(args) > 0 {
		vm.env["#"] = args[0]
	}

	return vm.execute()
}

// callNativeFromBuiltin calls a native Go function
func callNativeFromBuiltin(fn any, args []Value) (result Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = Nil
			err = &RuntimeError{Message: fmt.Sprintf("panic in native function call: %v", r)}
		}
	}()

	rv := reflect.ValueOf(fn)
	if rv.Kind() != reflect.Func {
		return Nil, ErrNotCallable
	}

	rt := rv.Type()
	numIn := rt.NumIn()
	isVariadic := rt.IsVariadic()

	// Check argument count
	if isVariadic {
		if len(args) < numIn-1 {
			return Nil, ErrArgumentCount
		}
	} else {
		if len(args) != numIn {
			return Nil, ErrArgumentCount
		}
	}

	// Convert arguments with proper type conversion
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		var targetType reflect.Type
		switch {
		case isVariadic && i >= numIn-1:
			targetType = rt.In(numIn - 1).Elem()
		case i < numIn:
			targetType = rt.In(i)
		default:
			in[i] = reflect.ValueOf(arg.Interface())
			continue
		}
		in[i] = convertArgForBuiltin(arg, targetType)
	}

	// Call function
	out := rv.Call(in)

	if len(out) == 0 {
		return Nil, nil
	}

	// Check for error return (handle 2-value returns where second is error)
	if len(out) == 2 {
		// Check if second return value is an error (only call IsNil on nillable types)
		errVal := out[1]
		errKind := errVal.Kind()
		isNil := false
		switch errKind {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
			isNil = errVal.IsNil()
		}
		if !isNil {
			if err, ok := errVal.Interface().(error); ok && err != nil {
				return Nil, err
			}
		}
		return NewValue(out[0].Interface()), nil
	}

	return NewValue(out[0].Interface()), nil
}

// convertArgForBuiltin converts a Value to the expected reflect.Value type
func convertArgForBuiltin(arg Value, targetType reflect.Type) reflect.Value {
	// Handle nil safely: reflect.ValueOf(nil) is invalid and can panic in Call().
	if arg.IsNil() {
		switch targetType.Kind() {
		case reflect.Interface, reflect.Pointer, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
			return reflect.Zero(targetType)
		default:
			return reflect.Zero(targetType)
		}
	}
	switch targetType.Kind() {
	case reflect.Int:
		return reflect.ValueOf(int(arg.Int()))
	case reflect.Int8:
		return reflect.ValueOf(int8(arg.Int()))
	case reflect.Int16:
		return reflect.ValueOf(int16(arg.Int()))
	case reflect.Int32:
		return reflect.ValueOf(int32(arg.Int()))
	case reflect.Int64:
		return reflect.ValueOf(arg.Int())
	case reflect.Uint:
		return reflect.ValueOf(uint(arg.Int()))
	case reflect.Uint8:
		return reflect.ValueOf(uint8(arg.Int()))
	case reflect.Uint16:
		return reflect.ValueOf(uint16(arg.Int()))
	case reflect.Uint32:
		return reflect.ValueOf(uint32(arg.Int()))
	case reflect.Uint64:
		return reflect.ValueOf(uint64(arg.Int()))
	case reflect.Float32:
		return reflect.ValueOf(float32(arg.Float()))
	case reflect.Float64:
		return reflect.ValueOf(arg.Float())
	case reflect.String:
		return reflect.ValueOf(arg.String())
	case reflect.Bool:
		return reflect.ValueOf(arg.Bool())
	case reflect.Slice:
		if arg.IsArray() {
			src := arg.Array()
			// Fast path: []any
			if targetType.Elem().Kind() == reflect.Interface && targetType.Elem().NumMethod() == 0 {
				return reflect.ValueOf(src)
			}
			dst := reflect.MakeSlice(targetType, len(src), len(src))
			elemType := targetType.Elem()
			for i := range src {
				ev := convertArgForBuiltin(NewValue(src[i]), elemType)
				if !ev.IsValid() {
					ev = reflect.Zero(elemType)
				}
				switch {
				case ev.Type().AssignableTo(elemType):
					dst.Index(i).Set(ev)
				case ev.Type().ConvertibleTo(elemType):
					dst.Index(i).Set(ev.Convert(elemType))
				default:
					dst.Index(i).Set(reflect.Zero(elemType))
				}
			}
			return dst
		}
	case reflect.Interface:
		return reflect.ValueOf(arg.Interface())
	}
	return reflect.ValueOf(arg.Interface())
}
