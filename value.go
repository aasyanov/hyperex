package hyperex

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func safeSprint(v any) (s string) {
	defer func() {
		if r := recover(); r != nil {
			s = "<panic>"
		}
	}()
	return fmt.Sprint(v)
}

const (
	maxStringDepth    = 10
	maxStringElems    = 50
	maxStringTotalLen = 4096
)

func truncateString(s string) string {
	if len(s) <= maxStringTotalLen {
		return s
	}
	return s[:maxStringTotalLen] + "…"
}

func (v Value) stringify(depth int, seen map[uintptr]bool) string {
	if depth > maxStringDepth {
		return "…"
	}
	switch v.kind {
	case kindString:
		return v.ptr.(string)
	case kindInt:
		return strconv.FormatInt(int64(v.bits), 10)
	case kindFloat:
		return strconv.FormatFloat(math.Float64frombits(v.bits), 'g', -1, 64)
	case kindBool:
		if v.bits != 0 {
			return litTrue
		}
		return litFalse
	case kindNil:
		return litNil
	case kindTime:
		return timeFromBits(v.bits).Format(time.RFC3339Nano)
	case kindDuration:
		return time.Duration(int64(v.bits)).String()
	case kindArray:
		arr := v.ptr.([]any)
		rv := reflect.ValueOf(arr)
		if rv.IsValid() {
			p := rv.Pointer()
			if p != 0 {
				if seen[p] {
					return litCycle
				}
				seen[p] = true
				defer delete(seen, p)
			}
		}
		n := len(arr)
		if n > maxStringElems {
			n = maxStringElems
		}
		var b strings.Builder
		b.WriteByte('[')
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(NewValue(arr[i]).stringify(depth+1, seen))
		}
		if len(arr) > n {
			if n > 0 {
				b.WriteString(", ")
			}
			b.WriteString("…")
		}
		b.WriteByte(']')
		return truncateString(b.String())
	case kindMap:
		m := v.ptr.(map[string]any)
		rv := reflect.ValueOf(m)
		if rv.IsValid() {
			p := rv.Pointer()
			if p != 0 {
				if seen[p] {
					return litCycle
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
		n := len(keys)
		if n > maxStringElems {
			n = maxStringElems
		}
		var b strings.Builder
		b.WriteByte('{')
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			k := keys[i]
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(NewValue(m[k]).stringify(depth+1, seen))
		}
		if len(m) > n {
			if n > 0 {
				b.WriteString(", ")
			}
			b.WriteString("…")
		}
		b.WriteByte('}')
		return truncateString(b.String())
	default:
		// Avoid panics from user-defined String() methods on external Go objects.
		return truncateString(safeSprint(v.ptr))
	}
}

// Value represents a runtime value in HyperEx.
// It uses a tagged union approach for efficient storage of common types.
// Size: 32 bytes (kind:8 + bits:8 + ptr:16) instead of 40 bytes.
//
// Value is not safe for concurrent mutation; treat as immutable after construction
// for concurrent reads of the same Value.
//
// IMPORTANT: Do not construct Value manually; use NewValue / Int / Float / String / Bool / Array / Map / Func.
// The (kind, bits, ptr) fields form an internal invariant. Violating it may cause panics or incorrect behavior.
type Value struct {
	kind valueKind
	// bits stores numeric values: int64 directly, float64 via math.Float64bits
	// This saves 8 bytes compared to having separate num and fnum fields
	bits uint64
	// Pointer to complex values (string, slice, map, etc.)
	ptr any
}

type valueKind uint8

const (
	kindNil valueKind = iota
	kindBool
	kindInt
	kindFloat
	kindString
	kindArray
	kindMap
	kindFunc
	kindAny // wrapped Go value
	kindTime
	kindDuration
)

// Nil is the nil value constant.
var Nil = Value{kind: kindNil}

// NewValue creates a Value from a Go value.
func NewValue(v any) Value {
	if v == nil {
		return Nil
	}

	switch x := v.(type) {
	case Value:
		return x
	case bool:
		return Bool(x)
	case int:
		return Int(int64(x))
	case int8:
		return Int(int64(x))
	case int16:
		return Int(int64(x))
	case int32:
		return Int(int64(x))
	case int64:
		return Int(x)
	case uint:
		if x > math.MaxInt64 {
			return Float(float64(x))
		}
		return Int(int64(x))
	case uint8:
		return Int(int64(x))
	case uint16:
		return Int(int64(x))
	case uint32:
		return Int(int64(x))
	case uint64:
		if x > math.MaxInt64 {
			return Float(float64(x))
		}
		return Int(int64(x))
	case float32:
		return Float(float64(x))
	case float64:
		return Float(x)
	case string:
		return String(x)
	case []any:
		return Array(x)
	case []Value:
		arr := make([]any, len(x))
		for i, v := range x {
			arr[i] = v
		}
		return Array(arr)
	case map[string]any:
		return Map(x)
	case *builtinFunc:
		return Func(x)
	case builtinFunc:
		// Rare: keep a stable pointer-shaped Func Value.
		cp := x
		return Func(&cp)
	case *closure:
		return Func(x)
	case time.Time:
		return Time(x)
	case time.Duration:
		return Duration(x)
	default:
		// Use reflection for complex types
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			arr := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				arr[i] = rv.Index(i).Interface()
			}
			return Array(arr)
		case reflect.Map:
			// Industrial hardening:
			// - Only convert maps with string keys to Map(map[string]any) without invoking user code.
			// - For non-string key maps, keep as kindAny so vm can handle them via reflection.
			if rv.Type().Key().Kind() != reflect.String {
				return Value{kind: kindAny, ptr: v}
			}
			m := make(map[string]any, rv.Len())
			iter := rv.MapRange()
			for iter.Next() {
				// Safe: key is string-kind.
				key := iter.Key().String()
				m[key] = iter.Value().Interface()
			}
			return Map(m)
		case reflect.Func:
			return Value{kind: kindFunc, ptr: v}
		case reflect.Pointer:
			if rv.IsNil() {
				return Nil
			}
			return NewValue(rv.Elem().Interface())
		case reflect.Struct:
			return Value{kind: kindAny, ptr: v}
		default:
			return Value{kind: kindAny, ptr: v}
		}
	}
}

// Bool constructs a boolean Value.
func Bool(b bool) Value {
	v := Value{kind: kindBool}
	if b {
		v.bits = 1
	}
	return v
}

// Int constructs an integer Value.
func Int(n int64) Value {
	return Value{kind: kindInt, bits: uint64(n)}
}

// Float constructs a floating-point Value.
func Float(f float64) Value {
	return Value{kind: kindFloat, bits: math.Float64bits(f)}
}

// String constructs a string Value.
func String(s string) Value {
	return Value{kind: kindString, ptr: s}
}

// Array constructs an array Value.
func Array(arr []any) Value {
	return Value{kind: kindArray, ptr: arr}
}

// Map constructs a map Value.
func Map(m map[string]any) Value {
	return Value{kind: kindMap, ptr: m}
}

// Func constructs a function Value.
func Func(fn any) Value {
	return Value{kind: kindFunc, ptr: fn}
}

// Time constructs a kindTime Value from t.
// The instant is normalized to UTC Absolute (monotonic clock stripped via UnixNano).
func Time(t time.Time) Value {
	return Value{kind: kindTime, bits: uint64(t.UTC().UnixNano())}
}

// Duration constructs a kindDuration Value from d (nanoseconds in bits).
func Duration(d time.Duration) Value {
	return Value{kind: kindDuration, bits: uint64(d)}
}

func timeFromBits(bits uint64) time.Time {
	return time.Unix(0, int64(bits)).UTC()
}

func durationFromBits(bits uint64) time.Duration {
	return time.Duration(int64(bits))
}

// IsNil reports whether v is nil.
func (v Value) IsNil() bool { return v.kind == kindNil }

// IsBool reports whether v is a boolean.
func (v Value) IsBool() bool { return v.kind == kindBool }

// IsInt reports whether v is an integer.
func (v Value) IsInt() bool { return v.kind == kindInt }

// IsFloat reports whether v is a float.
func (v Value) IsFloat() bool { return v.kind == kindFloat }

// IsNumber reports whether v is an int or float.
func (v Value) IsNumber() bool { return v.kind == kindInt || v.kind == kindFloat }

// IsString reports whether v is a string.
func (v Value) IsString() bool { return v.kind == kindString }

// IsArray reports whether v is an array.
func (v Value) IsArray() bool { return v.kind == kindArray }

// IsMap reports whether v is a map.
func (v Value) IsMap() bool { return v.kind == kindMap }

// IsFunc reports whether v is a function.
func (v Value) IsFunc() bool { return v.kind == kindFunc }

// IsTime reports whether v is a UTC instant (kindTime).
func (v Value) IsTime() bool { return v.kind == kindTime }

// IsDuration reports whether v is a duration (kindDuration).
func (v Value) IsDuration() bool { return v.kind == kindDuration }

// Type returns a string describing the value's type.
func (v Value) Type() string {
	switch v.kind {
	case kindNil:
		return litNil
	case kindBool:
		return nameBool
	case kindInt:
		return nameInt
	case kindFloat:
		return nameFloat
	case kindString:
		return nameString
	case kindArray:
		return typeNameArray
	case kindMap:
		return nameMap
	case kindFunc:
		return kindFunction
	case kindTime:
		return nameTime
	case kindDuration:
		return nameDuration
	case kindAny:
		if v.ptr == nil {
			return litNil
		}
		return reflect.TypeOf(v.ptr).String()
	default:
		return typeNameUnknown
	}
}

// Bool converts v to a boolean using HyperEx truthiness rules.
func (v Value) Bool() bool {
	switch v.kind {
	case kindBool:
		return v.bits != 0
	case kindInt:
		return v.bits != 0
	case kindFloat:
		return math.Float64frombits(v.bits) != 0
	case kindString:
		return v.ptr.(string) != ""
	case kindArray:
		return len(v.ptr.([]any)) > 0
	case kindMap:
		return len(v.ptr.(map[string]any)) > 0
	case kindNil:
		return false
	case kindTime, kindDuration:
		return v.bits != 0
	default:
		return v.ptr != nil
	}
}

// Int converts v to int64.
func (v Value) Int() int64 {
	switch v.kind {
	case kindInt:
		return int64(v.bits)
	case kindFloat:
		return int64(math.Float64frombits(v.bits))
	case kindBool:
		return int64(v.bits)
	case kindString:
		n, _ := strconv.ParseInt(v.ptr.(string), 0, 64)
		return n
	default:
		return 0
	}
}

// Float converts v to float64.
func (v Value) Float() float64 {
	switch v.kind {
	case kindFloat:
		return math.Float64frombits(v.bits)
	case kindInt:
		return float64(int64(v.bits))
	case kindBool:
		return float64(v.bits)
	case kindString:
		f, _ := strconv.ParseFloat(v.ptr.(string), 64)
		return f
	default:
		return 0
	}
}

// TryBool attempts to convert the value to bool without truthiness coercion.
// Only kindBool succeeds; other kinds return [ErrInvalidConversion].
func (v Value) TryBool() (bool, error) {
	if v.kind == kindBool {
		return v.bits != 0, nil
	}
	return false, ErrInvalidConversion
}

// TryInt attempts to convert the value to int64.
// Returns [ErrInvalidConversion] if the value cannot be converted.
func (v Value) TryInt() (int64, error) {
	switch v.kind {
	case kindInt:
		return int64(v.bits), nil
	case kindFloat:
		return int64(math.Float64frombits(v.bits)), nil
	case kindBool:
		return int64(v.bits), nil
	case kindString:
		s := v.ptr.(string)
		n, err := strconv.ParseInt(s, 0, 64)
		if err != nil {
			f, ferr := strconv.ParseFloat(s, 64)
			if ferr != nil {
				return 0, ErrInvalidConversion
			}
			return int64(f), nil
		}
		return n, nil
	default:
		return 0, ErrInvalidConversion
	}
}

// TryFloat attempts to convert the value to float64.
// Returns [ErrInvalidConversion] if the value cannot be converted.
func (v Value) TryFloat() (float64, error) {
	switch v.kind {
	case kindFloat:
		return math.Float64frombits(v.bits), nil
	case kindInt:
		return float64(int64(v.bits)), nil
	case kindBool:
		return float64(v.bits), nil
	case kindString:
		s := v.ptr.(string)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, ErrInvalidConversion
		}
		return f, nil
	default:
		return 0, ErrInvalidConversion
	}
}

// TryString attempts to convert the value to string without formatting composites.
// Only kindString succeeds; other kinds return [ErrInvalidConversion].
func (v Value) TryString() (string, error) {
	if v.kind == kindString {
		return v.ptr.(string), nil
	}
	return "", ErrInvalidConversion
}

// String returns a human-readable representation of v.
// Arrays and maps are formatted with depth/element/length limits; cycles are marked.
func (v Value) String() string {
	return v.stringify(0, make(map[uintptr]bool, 8))
}

// Array returns the underlying slice, or nil if v is not an array.
func (v Value) Array() []any {
	if v.kind == kindArray {
		return v.ptr.([]any)
	}
	return nil
}

// Map returns the underlying map, or nil if v is not a map.
func (v Value) Map() map[string]any {
	if v.kind == kindMap {
		return v.ptr.(map[string]any)
	}
	return nil
}

// Interface returns the underlying Go value.
func (v Value) Interface() any {
	switch v.kind {
	case kindNil:
		return nil
	case kindBool:
		return v.bits != 0
	case kindInt:
		return int64(v.bits)
	case kindFloat:
		return math.Float64frombits(v.bits)
	case kindString:
		return v.ptr.(string)
	case kindArray:
		return v.ptr.([]any)
	case kindMap:
		return v.ptr.(map[string]any)
	case kindTime:
		return timeFromBits(v.bits)
	case kindDuration:
		return durationFromBits(v.bits)
	default:
		return v.ptr
	}
}

// Len returns the length of the value (for strings, arrays, maps).
// For strings, returns the number of Unicode code points (runes), not bytes.
func (v Value) Len() int {
	switch v.kind {
	case kindString:
		// Use utf8.RuneCountInString instead of len([]rune()) to avoid allocation
		return utf8.RuneCountInString(v.ptr.(string))
	case kindArray:
		return len(v.ptr.([]any))
	case kindMap:
		return len(v.ptr.(map[string]any))
	default:
		return 0
	}
}

// Index returns the element at the given index (for arrays and strings).
func (v Value) Index(i int) (Value, error) {
	switch v.kind {
	case kindArray:
		arr := v.ptr.([]any)
		if i < 0 {
			i = len(arr) + i
		}
		if i < 0 || i >= len(arr) {
			return Nil, ErrIndexOutOfBounds
		}
		return NewValue(arr[i]), nil
	case kindString:
		s := v.ptr.(string)
		runes := []rune(s)
		if i < 0 {
			i = len(runes) + i
		}
		if i < 0 || i >= len(runes) {
			return Nil, ErrIndexOutOfBounds
		}
		return String(string(runes[i])), nil
	default:
		return Nil, ErrNotIndexable
	}
}

// Get returns the value at the given key (for maps and structs).
func (v Value) Get(key string) (Value, bool) {
	switch v.kind {
	case kindMap:
		m := v.ptr.(map[string]any)
		if val, ok := m[key]; ok {
			return NewValue(val), true
		}
		return Nil, false
	case kindAny:
		if v.ptr == nil {
			return Nil, false
		}
		rv := reflect.ValueOf(v.ptr)
		if rv.Kind() == reflect.Pointer {
			if rv.IsNil() {
				return Nil, false
			}
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Struct {
			field := rv.FieldByName(key)
			if field.IsValid() && field.CanInterface() {
				return NewValue(field.Interface()), true
			}
			// Try lowercase
			field = rv.FieldByNameFunc(func(name string) bool {
				return strings.EqualFold(name, key)
			})
			if field.IsValid() && field.CanInterface() {
				return NewValue(field.Interface()), true
			}
		}
		return Nil, false
	default:
		return Nil, false
	}
}

// Equal checks if two values are equal.
func (v Value) Equal(other Value) bool {
	return valuesEqual(v, other, make(map[valuePair]bool), 0)
}

type valuePair struct {
	left  uintptr
	right uintptr
	kind  valueKind
}

const maxValueEqualityDepth = 1000

func valuesEqual(v, other Value, seen map[valuePair]bool, depth int) bool {
	if v.kind != other.kind {
		if v.IsNumber() && other.IsNumber() {
			return v.Float() == other.Float()
		}
		return false
	}

	switch v.kind {
	case kindNil:
		return true
	case kindBool, kindInt, kindTime, kindDuration, kindFloat:
		return v.bits == other.bits
	case kindString:
		return v.ptr.(string) == other.ptr.(string)
	case kindArray:
		if depth >= maxValueEqualityDepth {
			return false
		}
		a1, a2 := v.ptr.([]any), other.ptr.([]any)
		if len(a1) != len(a2) {
			return false
		}
		p1, p2 := reflect.ValueOf(a1).Pointer(), reflect.ValueOf(a2).Pointer()
		if p1 == p2 {
			return true
		}
		pair := valuePair{left: p1, right: p2, kind: kindArray}
		if seen[pair] {
			return true
		}
		seen[pair] = true
		defer delete(seen, pair)
		for i := range a1 {
			if !valuesEqual(NewValue(a1[i]), NewValue(a2[i]), seen, depth+1) {
				return false
			}
		}
		return true
	case kindMap:
		if depth >= maxValueEqualityDepth {
			return false
		}
		m1, m2 := v.ptr.(map[string]any), other.ptr.(map[string]any)
		if len(m1) != len(m2) {
			return false
		}
		p1, p2 := reflect.ValueOf(m1).Pointer(), reflect.ValueOf(m2).Pointer()
		if p1 == p2 {
			return true
		}
		pair := valuePair{left: p1, right: p2, kind: kindMap}
		if seen[pair] {
			return true
		}
		seen[pair] = true
		defer delete(seen, pair)
		for k, v1 := range m1 {
			v2, ok := m2[k]
			if !ok || !valuesEqual(NewValue(v1), NewValue(v2), seen, depth+1) {
				return false
			}
		}
		return true
	case kindFunc:
		return funcValuesEqual(v.ptr, other.ptr)
	case kindAny:
		return anyValuesEqual(v.ptr, other.ptr)
	default:
		return false
	}
}

// funcValuesEqual compares function Values without using == on uncomparable
// concrete types (structs that embed func fields panic under interface ==).
func funcValuesEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch x := a.(type) {
	case *builtinFunc:
		y, ok := b.(*builtinFunc)
		return ok && x != nil && y != nil && x.name == y.name && reflect.ValueOf(x.fn).Pointer() == reflect.ValueOf(y.fn).Pointer()
	default:
		ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
		if !ra.IsValid() || !rb.IsValid() || ra.Type() != rb.Type() {
			return false
		}
		switch ra.Kind() {
		case reflect.Func:
			return ra.Pointer() == rb.Pointer()
		case reflect.Pointer:
			if ra.IsNil() || rb.IsNil() {
				return ra.IsNil() == rb.IsNil()
			}
			return ra.Pointer() == rb.Pointer()
		default:
			return false
		}
	}
}

func anyValuesEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
	if !ra.IsValid() || !rb.IsValid() || ra.Type() != rb.Type() {
		return false
	}
	switch ra.Kind() {
	case reflect.Func:
		return ra.Pointer() == rb.Pointer()
	case reflect.Pointer, reflect.Map, reflect.Chan, reflect.UnsafePointer:
		return ra.Pointer() == rb.Pointer()
	case reflect.Slice:
		return false // distinct slice headers; element-wise handled via kindArray
	default:
		// Avoid panic on structs containing funcs: only comparable kinds use ==.
		if ra.Comparable() {
			return a == b
		}
		return false
	}
}

// Compare compares two values. Returns -1, 0, or 1.
func (v Value) Compare(other Value) int {
	// Handle numeric comparisons
	if v.IsNumber() && other.IsNumber() {
		f1, f2 := v.Float(), other.Float()
		if f1 < f2 {
			return -1
		}
		if f1 > f2 {
			return 1
		}
		return 0
	}

	// Instant / duration ordering (same kind only)
	if v.kind == kindTime && other.kind == kindTime {
		a, b := int64(v.bits), int64(other.bits)
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
	if v.kind == kindDuration && other.kind == kindDuration {
		a, b := int64(v.bits), int64(other.bits)
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}

	// String comparison
	if v.IsString() && other.IsString() {
		s1, s2 := v.ptr.(string), other.ptr.(string)
		if s1 < s2 {
			return -1
		}
		if s1 > s2 {
			return 1
		}
		return 0
	}

	// Fall back to equality check
	if v.Equal(other) {
		return 0
	}
	return -1 // Arbitrary for incomparable types
}
