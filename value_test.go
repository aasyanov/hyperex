package hyperex

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type panicStringerKey struct{}

func (panicStringerKey) String() string { panic("boom") }

// =============================================================================
// VALUE TESTS
// =============================================================================

func TestValue_Types(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		isNil    bool
		isBool   bool
		isInt    bool
		isFloat  bool
		isNumber bool
		isString bool
		isArray  bool
		isMap    bool
		isFunc   bool
	}{
		{"nil", Nil, true, false, false, false, false, false, false, false, false},
		{"bool", Bool(true), false, true, false, false, false, false, false, false, false},
		{"int", Int(42), false, false, true, false, true, false, false, false, false},
		{"float", Float(3.14), false, false, false, true, true, false, false, false, false},
		{"string", String("hello"), false, false, false, false, false, true, false, false, false},
		{"array", Array([]any{1, 2}), false, false, false, false, false, false, true, false, false},
		{"map", Map(map[string]any{"a": 1}), false, false, false, false, false, false, false, true, false},
		{"func", Func(func() {}), false, false, false, false, false, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.IsNil() != tt.isNil {
				t.Errorf("IsNil: got %v, want %v", tt.value.IsNil(), tt.isNil)
			}
			if tt.value.IsBool() != tt.isBool {
				t.Errorf("IsBool: got %v, want %v", tt.value.IsBool(), tt.isBool)
			}
			if tt.value.IsInt() != tt.isInt {
				t.Errorf("IsInt: got %v, want %v", tt.value.IsInt(), tt.isInt)
			}
			if tt.value.IsFloat() != tt.isFloat {
				t.Errorf("IsFloat: got %v, want %v", tt.value.IsFloat(), tt.isFloat)
			}
			if tt.value.IsNumber() != tt.isNumber {
				t.Errorf("IsNumber: got %v, want %v", tt.value.IsNumber(), tt.isNumber)
			}
			if tt.value.IsString() != tt.isString {
				t.Errorf("IsString: got %v, want %v", tt.value.IsString(), tt.isString)
			}
			if tt.value.IsArray() != tt.isArray {
				t.Errorf("IsArray: got %v, want %v", tt.value.IsArray(), tt.isArray)
			}
			if tt.value.IsMap() != tt.isMap {
				t.Errorf("IsMap: got %v, want %v", tt.value.IsMap(), tt.isMap)
			}
			if tt.value.IsFunc() != tt.isFunc {
				t.Errorf("IsFunc: got %v, want %v", tt.value.IsFunc(), tt.isFunc)
			}
		})
	}
}

func TestValue_Type(t *testing.T) {
	tests := []struct {
		value    Value
		typeName string
	}{
		{Nil, "nil"},
		{Bool(true), "bool"},
		{Int(42), "int"},
		{Float(3.14), "float"},
		{String("hello"), "string"},
		{Array([]any{1}), "array"},
		{Map(map[string]any{}), "map"},
		{Func(func() {}), "function"},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			if tt.value.Type() != tt.typeName {
				t.Errorf("got %v, want %v", tt.value.Type(), tt.typeName)
			}
		})
	}
}

func TestValue_Bool(t *testing.T) {
	tests := []struct {
		value  Value
		result bool
	}{
		{Bool(true), true},
		{Bool(false), false},
		{Int(1), true},
		{Int(0), false},
		{Float(1.0), true},
		{Float(0.0), false},
		{String("hello"), true},
		{String(""), false},
		{Array([]any{1}), true},
		{Array([]any{}), false},
		{Map(map[string]any{"a": 1}), true},
		{Map(map[string]any{}), false},
		{Nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.value.Type(), func(t *testing.T) {
			if tt.value.Bool() != tt.result {
				t.Errorf("got %v, want %v", tt.value.Bool(), tt.result)
			}
		})
	}
}

func TestValue_Int(t *testing.T) {
	tests := []struct {
		value  Value
		result int64
	}{
		{Int(42), 42},
		{Float(3.7), 3},
		{Bool(true), 1},
		{Bool(false), 0},
		{String("123"), 123},
		{String("invalid"), 0},
		{Nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.value.Type(), func(t *testing.T) {
			if tt.value.Int() != tt.result {
				t.Errorf("got %v, want %v", tt.value.Int(), tt.result)
			}
		})
	}
}

func TestValue_Float(t *testing.T) {
	tests := []struct {
		value  Value
		result float64
	}{
		{Float(3.14), 3.14},
		{Int(42), 42.0},
		{Bool(true), 1.0},
		{String("3.14"), 3.14},
		{Nil, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.value.Type(), func(t *testing.T) {
			if tt.value.Float() != tt.result {
				t.Errorf("got %v, want %v", tt.value.Float(), tt.result)
			}
		})
	}
}

func TestValue_String(t *testing.T) {
	tests := []struct {
		value  Value
		result string
	}{
		{String("hello"), "hello"},
		{Int(42), "42"},
		{Float(3.14), "3.14"},
		{Bool(true), "true"},
		{Bool(false), "false"},
		{Nil, "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.value.Type(), func(t *testing.T) {
			if tt.value.String() != tt.result {
				t.Errorf("got %v, want %v", tt.value.String(), tt.result)
			}
		})
	}
}

func TestValue_ArrayString(t *testing.T) {
	v := Array([]any{int64(1), int64(2), int64(3)})
	s := v.String()
	if !strings.Contains(s, "1") || !strings.Contains(s, "2") || !strings.Contains(s, "3") {
		t.Errorf("unexpected string: %v", s)
	}
}

func TestValue_MapString(t *testing.T) {
	v := Map(map[string]any{"a": int64(1)})
	s := v.String()
	if !strings.Contains(s, "a") || !strings.Contains(s, "1") {
		t.Errorf("unexpected string: %v", s)
	}
}

func TestValue_Len(t *testing.T) {
	tests := []struct {
		value  Value
		result int
	}{
		{String("hello"), 5},
		{Array([]any{1, 2, 3}), 3},
		{Map(map[string]any{"a": 1, "b": 2}), 2},
		{Int(42), 0},
	}

	for _, tt := range tests {
		t.Run(tt.value.Type(), func(t *testing.T) {
			if tt.value.Len() != tt.result {
				t.Errorf("got %v, want %v", tt.value.Len(), tt.result)
			}
		})
	}
}

func TestValue_Index(t *testing.T) {
	arr := Array([]any{int64(10), int64(20), int64(30)})

	// Valid indices
	v, err := arr.Index(0)
	if err != nil || v.Int() != 10 {
		t.Errorf("Index(0): got %v, err %v", v, err)
	}

	v, err = arr.Index(-1)
	if err != nil || v.Int() != 30 {
		t.Errorf("Index(-1): got %v, err %v", v, err)
	}

	// Out of bounds
	_, err = arr.Index(10)
	if !errors.Is(err, ErrIndexOutOfBounds) {
		t.Errorf("expected out of bounds error")
	}

	// String index
	str := String("hello")
	v, err = str.Index(0)
	if err != nil || v.String() != "h" {
		t.Errorf("String.Index(0): got %v, err %v", v, err)
	}

	// Not indexable
	i := Int(42)
	_, err = i.Index(0)
	if !errors.Is(err, ErrNotIndexable) {
		t.Errorf("expected not indexable error")
	}
}

func TestValue_Get(t *testing.T) {
	m := Map(map[string]any{"foo": int64(42)})

	v, ok := m.Get("foo")
	if !ok || v.Int() != 42 {
		t.Errorf("Get(foo): got %v, ok %v", v, ok)
	}

	_, ok = m.Get("missing")
	if ok {
		t.Error("expected missing key")
	}
}

func TestValue_GetStruct(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	v := NewValue(&Person{Name: "Alice", Age: 30})

	name, ok := v.Get("Name")
	if !ok || name.String() != "Alice" {
		t.Errorf("Get(Name): got %v, ok %v", name, ok)
	}

	// Case-insensitive
	age, ok := v.Get("age")
	if !ok {
		t.Error("expected to find age (case-insensitive)")
	}
	_ = age
}

func TestValue_Equal(t *testing.T) {
	tests := []struct {
		a, b   Value
		result bool
	}{
		{Nil, Nil, true},
		{Bool(true), Bool(true), true},
		{Bool(true), Bool(false), false},
		{Int(42), Int(42), true},
		{Int(42), Int(43), false},
		{Int(42), Float(42.0), true}, // cross-type numeric
		{Float(3.14), Float(3.14), true},
		{String("hello"), String("hello"), true},
		{String("hello"), String("world"), false},
		{Array([]any{int64(1)}), Array([]any{int64(1)}), true},
		{Array([]any{int64(1)}), Array([]any{int64(2)}), false},
	}

	for _, tt := range tests {
		if tt.a.Equal(tt.b) != tt.result {
			t.Errorf("%v == %v: got %v, want %v", tt.a, tt.b, tt.a.Equal(tt.b), tt.result)
		}
	}
}

func TestValue_Compare(t *testing.T) {
	tests := []struct {
		a, b   Value
		result int
	}{
		{Int(1), Int(2), -1},
		{Int(2), Int(2), 0},
		{Int(3), Int(2), 1},
		{Float(1.0), Float(2.0), -1},
		{String("a"), String("b"), -1},
		{String("b"), String("a"), 1},
		{String("a"), String("a"), 0},
	}

	for _, tt := range tests {
		if tt.a.Compare(tt.b) != tt.result {
			t.Errorf("%v <=> %v: got %v, want %v", tt.a, tt.b, tt.a.Compare(tt.b), tt.result)
		}
	}
}

func TestNewValue_ComplexTypes(t *testing.T) {
	// Slice of int
	v := NewValue([]int{1, 2, 3})
	if !v.IsArray() || v.Len() != 3 {
		t.Errorf("[]int: got %v", v)
	}

	// Map with int values
	v = NewValue(map[string]int{"a": 1})
	if !v.IsMap() {
		t.Errorf("map[string]int: got %v", v)
	}

	// Pointer
	x := 42
	v = NewValue(&x)
	if v.Int() != 42 {
		t.Errorf("*int: got %v", v)
	}

	// Nil pointer
	var p *int
	v = NewValue(p)
	if !v.IsNil() {
		t.Errorf("nil *int: got %v", v)
	}

	// Struct
	type S struct{ X int }
	v = NewValue(S{X: 5})
	if v.Type() != "hyperex.S" {
		t.Logf("struct type: %v", v.Type())
	}
}

// =============================================================================
// GO INTEROP TESTS (kindAny support)
// =============================================================================

func TestKindAny_Contains_GoSlice(t *testing.T) {
	// Test 'in' operator with native Go slice (not converted to Array)
	goSlice := []int{10, 20, 30, 40, 50}
	env := map[string]any{"arr": goSlice}

	// Should find element
	result, err := Eval(context.Background(), "20 in arr", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("Expected true for 20 in [10,20,30,40,50], got %v", result)
	}

	// Should not find element
	result, err = Eval(context.Background(), "99 in arr", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("Expected false for 99 in [10,20,30,40,50], got %v", result)
	}
}

func TestKindAny_Contains_GoMap(t *testing.T) {
	// Test 'in' operator with native Go map
	goMap := map[string]int{"foo": 1, "bar": 2, "baz": 3}
	env := map[string]any{"m": goMap}

	// Should find key
	result, err := Eval(context.Background(), `"foo" in m`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("Expected true for 'foo' in map, got %v", result)
	}

	// Should not find key
	result, err = Eval(context.Background(), `"xyz" in m`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("Expected false for 'xyz' in map, got %v", result)
	}
}

func TestKindAny_Index_GoSlice(t *testing.T) {
	// Test indexing native Go slice
	goSlice := []string{"a", "b", "c", "d", "e"}
	env := map[string]any{"arr": goSlice}

	// Positive index
	result, err := Eval(context.Background(), "arr[0]", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "a" {
		t.Errorf("Expected 'a', got %v", result)
	}

	// Negative index
	result, err = Eval(context.Background(), "arr[-1]", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "e" {
		t.Errorf("Expected 'e', got %v", result)
	}
}

func TestKindAny_Index_GoMap(t *testing.T) {
	// Test indexing native Go map
	goMap := map[string]int{"x": 100, "y": 200}
	env := map[string]any{"m": goMap}

	result, err := Eval(context.Background(), `m["x"]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 100 {
		t.Errorf("Expected 100, got %v", result)
	}
}

func TestKindAny_Slice_GoSlice(t *testing.T) {
	// Test slicing native Go slice
	goSlice := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	env := map[string]any{"arr": goSlice}

	// Basic slice
	result, err := Eval(context.Background(), "arr[2:5]", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 3 || NewValue(arr[0]).Int() != 2 || NewValue(arr[2]).Int() != 4 {
		t.Errorf("Expected [2,3,4], got %v", arr)
	}

	// Slice with step
	result, err = Eval(context.Background(), "arr[0:10:2]", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 5 {
		t.Errorf("Expected 5 elements with step 2, got %d", len(arr))
	}
}
func TestModeB_ValueEqual_TimeDurationFunc(t *testing.T) {
	t1 := Time(time.Unix(100, 0).UTC())
	t2 := Time(time.Unix(100, 0).UTC())
	t3 := Time(time.Unix(101, 0).UTC())
	if !t1.Equal(t2) || t1.Equal(t3) {
		t.Fatal("time Equal")
	}
	d1 := Duration(time.Second)
	d2 := Duration(time.Second)
	d3 := Duration(2 * time.Second)
	if !d1.Equal(d2) || d1.Equal(d3) {
		t.Fatal("duration Equal")
	}

	// Compare ordering for time/duration
	if c := t1.Compare(t3); c >= 0 {
		t.Fatalf("t1 Compare t3: %v", c)
	}
	if c := d3.Compare(d1); c <= 0 {
		t.Fatalf("d3 Compare d1: %v", c)
	}

	// func / any equality helpers via Equal on constructed values
	lenBF := &builtinFunc{name: "len", fn: wrapBuiltin(builtinLen)}
	bf := Func(lenBF)
	bfSame := Func(lenBF) // same *builtinFunc -> Equal
	if !bf.Equal(bfSame) {
		t.Fatal("same builtin should Equal")
	}
	bf2 := Func(&builtinFunc{name: "abs", fn: wrapBuiltin(builtinAbs)})
	if bf.Equal(bf2) {
		t.Fatal("different builtins must not Equal")
	}
	// kindAny comparable scalars go through anyValuesEqual
	any1 := Value{kind: kindAny, ptr: 7}
	any2 := Value{kind: kindAny, ptr: 7}
	any3 := Value{kind: kindAny, ptr: 8}
	if !any1.Equal(any2) {
		t.Fatal("any Equal")
	}
	if any1.Equal(any3) {
		t.Fatal("any unequal")
	}
	// slice headers are never equal via anyValuesEqual
	s1 := Value{kind: kindAny, ptr: []int{1}}
	s2 := Value{kind: kindAny, ptr: []int{1}}
	if s1.Equal(s2) {
		t.Fatal("kindAny slices should not Equal")
	}
}
