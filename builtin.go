package hyperex

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// builtinFunc represents a built-in function.
type builtinFunc struct {
	name string
	fn   func(*execVM, []Value) (Value, error)
}

// wrapBuiltin adapts legacy builtins that don't need vm context to the vm-aware builtinFunc signature.
func wrapBuiltin(fn func([]Value) (Value, error)) func(*execVM, []Value) (Value, error) {
	return func(_ *execVM, args []Value) (Value, error) {
		return fn(args)
	}
}

// defaultBuiltinNameSet holds the keys of defaultBuiltins for O(1) name checks
// (pipe vs bitwise OR heuristic, analysis warnings). Built once at init.
var defaultBuiltinNameSet = func() map[string]struct{} {
	m := make(map[string]struct{}, 80)
	for name := range defaultBuiltins() {
		m[name] = struct{}{}
	}
	return m
}()

// isDefaultBuiltinName reports whether name is a key in the default builtin set.
func isDefaultBuiltinName(name string) bool {
	_, ok := defaultBuiltinNameSet[name]
	return ok
}

// defaultBuiltins returns the default set of built-in functions.
func defaultBuiltins() map[string]Value {
	return map[string]Value{
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

		// Utility functions
		namePrint:   Func(&builtinFunc{name: namePrint, fn: wrapBuiltin(builtinPrint)}),
		nameSprintf: Func(&builtinFunc{name: nameSprintf, fn: wrapBuiltin(builtinSprintf)}),
		nameKeys:    Func(&builtinFunc{name: nameKeys, fn: builtinKeys}),
		nameValues:  Func(&builtinFunc{name: nameValues, fn: builtinValues}),

		// New utility functions
		nameClamp:    Func(&builtinFunc{name: nameClamp, fn: wrapBuiltin(builtinClamp)}),
		nameBetween:  Func(&builtinFunc{name: nameBetween, fn: wrapBuiltin(builtinBetween)}),
		nameHasKey:   Func(&builtinFunc{name: nameHasKey, fn: wrapBuiltin(builtinHasKey)}),
		nameEmpty:    Func(&builtinFunc{name: nameEmpty, fn: wrapBuiltin(builtinEmpty)}),
		nameNotEmpty: Func(&builtinFunc{name: nameNotEmpty, fn: wrapBuiltin(builtinNotEmpty)}),
		nameSize:     Func(&builtinFunc{name: nameSize, fn: wrapBuiltin(builtinLen)}), // alias for len
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

// String builtins
func builtinLen(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Int(int64(args[0].Len())), nil
}

func builtinUpper(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return String(strings.ToUpper(args[0].String())), nil
}

func builtinLower(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return String(strings.ToLower(args[0].String())), nil
}

func builtinTrim(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return String(strings.TrimSpace(args[0].String())), nil
}

func builtinTrimLeft(args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return Nil, ErrArgumentCount
	}
	s := args[0].String()
	if len(args) == 2 {
		return String(strings.TrimLeft(s, args[1].String())), nil
	}
	return String(strings.TrimLeft(s, " \t\n\r")), nil
}

func builtinTrimRight(args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return Nil, ErrArgumentCount
	}
	s := args[0].String()
	if len(args) == 2 {
		return String(strings.TrimRight(s, args[1].String())), nil
	}
	return String(strings.TrimRight(s, " \t\n\r")), nil
}

func builtinSplit(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	parts := strings.Split(args[0].String(), args[1].String())
	arr := make([]any, len(parts))
	for i, p := range parts {
		arr[i] = p
	}
	return Array(arr), nil
}

func builtinJoin(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsArray() {
		return Nil, ErrInvalidType
	}
	arr := args[0].Array()
	parts := make([]string, len(arr))
	for i, v := range arr {
		parts[i] = NewValue(v).String()
	}
	return String(strings.Join(parts, args[1].String())), nil
}

func builtinContains(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	return Bool(strings.Contains(args[0].String(), args[1].String())), nil
}

func builtinStartsWith(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	return Bool(strings.HasPrefix(args[0].String(), args[1].String())), nil
}

func builtinEndsWith(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	return Bool(strings.HasSuffix(args[0].String(), args[1].String())), nil
}

func builtinReplace(args []Value) (Value, error) {
	if len(args) != 3 {
		return Nil, ErrArgumentCount
	}
	return String(strings.ReplaceAll(args[0].String(), args[1].String(), args[2].String())), nil
}

func builtinSubstr(args []Value) (Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return Nil, ErrArgumentCount
	}
	s := args[0].String()
	runes := []rune(s)
	runeLen := len(runes)

	start64 := args[1].Int()
	// Clamp extremes without relying on int width (important on 32-bit platforms).
	if start64 > int64(runeLen) {
		return String(""), nil
	}
	if start64 < -int64(runeLen) {
		start64 = 0
	}
	start := int(start64)
	if start < 0 {
		start = runeLen + start
	}
	if start < 0 {
		start = 0
	}
	if start > runeLen {
		return String(""), nil
	}
	if len(args) == 3 {
		length64 := args[2].Int()
		// Guard against negative length
		if length64 < 0 {
			length64 = 0
		}
		// Compute end in int64 then clamp into [start, runeLen] to avoid int overflow.
		end64 := int64(start) + length64
		if end64 < int64(start) || end64 > int64(runeLen) {
			end64 = int64(runeLen)
		}
		return String(string(runes[start:int(end64)])), nil
	}
	return String(string(runes[start:])), nil
}

func builtinIndexOf(args []Value) (Value, error) {
	if len(args) != 2 {
		return Nil, ErrArgumentCount
	}
	s := args[0].String()
	substr := args[1].String()
	// Get byte index
	byteIdx := strings.Index(s, substr)
	if byteIdx == -1 {
		return Int(-1), nil
	}
	// Convert byte index to rune index
	runeIdx := utf8.RuneCountInString(s[:byteIdx])
	return Int(int64(runeIdx)), nil
}

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

// NOTE: stableMapString / stableValueString were superseded by stableValueKey(),
// which is cycle-safe and sandbox-aware.

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
	result, err := flattenArrayWithDepth(vm, args[0].Array(), 0, 100)
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
	val, err := callBuiltinFunc(vm, pred, []Value{NewValue(elem), Int(int64(index))})
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
		val, err := callBuiltinFunc(vm, fn, []Value{NewValue(elem), Int(int64(i))})
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
		val, err := callBuiltinFunc(vm, fn, []Value{NewValue(elem), Int(int64(i))})
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
		val, err := callBuiltinFunc(vm, fn, []Value{acc, NewValue(arr[i]), Int(int64(i))})
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
		val, err := callBuiltinFunc(vm, fn, []Value{NewValue(elem), Int(int64(i))})
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
		val, err := callBuiltinFunc(vm, fn, []Value{NewValue(elem), Int(int64(i))})
		if err != nil {
			return Nil, err
		}
		if val.Bool() {
			return Int(int64(i)), nil
		}
	}
	return Int(-1), nil
}

// callBuiltinFunc calls a function value with arguments
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

	// Inherit strict mode from the closure's originating vm
	vm.strictMode = closure.strictMode

	// Inherit maxDepth from the closure's originating vm when available.
	// If not set, fall back to a safe default to prevent infinite recursion.
	if closure.maxDepth > 0 {
		vm.maxDepth = closure.maxDepth
	} else if vm.maxDepth == 0 {
		vm.maxDepth = 1000
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

// Type conversion builtins
func builtinInt(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	// Use TryInt for strict conversion - returns error on invalid strings
	n, err := args[0].TryInt()
	if err != nil {
		return Nil, &RuntimeError{Message: err.Error()}
	}
	return Int(n), nil
}

func builtinFloat(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	// Use TryFloat for strict conversion - returns error on invalid strings
	f, err := args[0].TryFloat()
	if err != nil {
		return Nil, &RuntimeError{Message: err.Error()}
	}
	return Float(f), nil
}

func builtinString(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return String(args[0].String()), nil
}

func builtinBool(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Bool(args[0].Bool()), nil
}

func builtinType(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return String(args[0].Type()), nil
}

func builtinIsNil(args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	return Bool(args[0].IsNil()), nil
}

// Utility builtins
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

func builtinKeys(vm *execVM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsMap() {
		return Nil, ErrInvalidType
	}
	m := args[0].Map()
	keys := make([]any, 0, len(m))
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
	return Array(keys), nil
}

func builtinValues(vm *execVM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Nil, ErrArgumentCount
	}
	if !args[0].IsMap() {
		return Nil, ErrInvalidType
	}
	m := args[0].Map()
	values := make([]any, 0, len(m))
	i := 0
	for _, v := range m {
		if vm != nil {
			if err := vm.sandboxLoopCheck(i); err != nil {
				return Nil, err
			}
		}
		i++
		values = append(values, v)
	}
	return Array(values), nil
}

// =============================================================================
// NEW UTILITY FUNCTIONS
// =============================================================================

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

	// Calculate number of chunks and limit to prevent excessive allocation
	numChunks := (len(arr) + size - 1) / size
	const maxChunks = 1_000_000
	if numChunks > maxChunks {
		return Nil, &RuntimeError{Message: fmt.Sprintf("chunk would create %d chunks, exceeds maximum %d", numChunks, maxChunks)}
	}

	result := make([]any, 0, numChunks)
	iter := 0
	for i := 0; i < len(arr); i += size {
		if vm != nil {
			if err := vm.sandboxLoopCheck(iter); err != nil {
				return Nil, err
			}
		}
		iter++
		end := i + size
		if end > len(arr) {
			end = len(arr)
		}
		chunk := make([]any, end-i)
		copy(chunk, arr[i:end])
		result = append(result, chunk)
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
		keyResult, err := callBuiltinFunc(vm, fn, []Value{NewValue(item)})
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
			fieldKey:   key,
			"items": groups[key],
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
		keyResult, err := callBuiltinFunc(vm, fn, []Value{NewValue(item)})
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
	// Sort using computed keys
	sortSlice(keyed, func(i, j int) bool {
		ki, kj := keyed[i].key, keyed[j].key

		var less bool
		if ki.IsNumber() && kj.IsNumber() {
			less = ki.Float() < kj.Float()
		} else {
			less = ki.String() < kj.String()
		}

		if desc {
			return !less
		}
		return less
	})

	// Extract sorted items
	result := make([]any, len(keyed))
	for i, k := range keyed {
		result[i] = k.item
	}

	return Array(result), nil
}

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

// convertJSONNumbers recursively converts float64 values that are whole numbers to int64
// This provides better compatibility with HyperEx's type system
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
