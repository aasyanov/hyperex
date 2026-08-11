// Library-style Value API demos (no expression eval required).
package main

import (
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	fmt.Println("=== Constructors ===")
	vNew := hyperex.NewValue(42)
	vInt := hyperex.Int(7)
	vFloat := hyperex.Float(3.14)
	vStr := hyperex.String("hello")
	vBool := hyperex.Bool(true)
	vArr := hyperex.Array([]any{1, "two", true})
	vMap := hyperex.Map(map[string]any{"a": 1, "b": "x"})
	vNil := hyperex.Nil
	vFunc := hyperex.Func(func(x int) int { return x * 2 })

	fmt.Printf("  NewValue(42): Type=%s Interface=%v\n", vNew.Type(), vNew.Interface())
	fmt.Printf("  Int(7):       Type=%s\n", vInt.Type())
	fmt.Printf("  Float(3.14):  Type=%s\n", vFloat.Type())
	fmt.Printf("  String:       Type=%s Len=%d\n", vStr.Type(), vStr.Len())
	fmt.Printf("  Bool(true):   Type=%s IsBool=%v\n", vBool.Type(), vBool.IsBool())
	fmt.Printf("  Array:        Type=%s Len=%d IsArray=%v\n", vArr.Type(), vArr.Len(), vArr.IsArray())
	fmt.Printf("  Map:          Type=%s Len=%d IsMap=%v\n", vMap.Type(), vMap.Len(), vMap.IsMap())
	fmt.Printf("  Nil:          Type=%s IsNil=%v\n", vNil.Type(), vNil.IsNil())
	fmt.Printf("  Func:         Type=%s IsFunc=%v\n", vFunc.Type(), vFunc.IsFunc())

	fmt.Println("\n=== Try* conversions ===")
	if b, err := vBool.TryBool(); err != nil {
		fmt.Printf("  TryBool err: %v\n", err)
	} else {
		fmt.Printf("  TryBool: %v\n", b)
	}
	if n, err := vInt.TryInt(); err != nil {
		fmt.Printf("  TryInt err: %v\n", err)
	} else {
		fmt.Printf("  TryInt: %d\n", n)
	}
	if f, err := vFloat.TryFloat(); err != nil {
		fmt.Printf("  TryFloat err: %v\n", err)
	} else {
		fmt.Printf("  TryFloat: %v\n", f)
	}
	if s, err := vStr.TryString(); err != nil {
		fmt.Printf("  TryString err: %v\n", err)
	} else {
		fmt.Printf("  TryString: %q\n", s)
	}
	if _, err := vInt.TryString(); err != nil {
		fmt.Printf("  TryString on Int: %v\n", err)
	}

	fmt.Println("\n=== Equal / Compare ===")
	fmt.Printf("  Int(7).Equal(NewValue(7)): %v\n", vInt.Equal(hyperex.NewValue(7)))
	fmt.Printf("  Int(7).Equal(Float(7)):    %v\n", vInt.Equal(hyperex.Float(7)))
	fmt.Printf("  Int(3).Compare(Int(5)):    %d\n", hyperex.Int(3).Compare(hyperex.Int(5)))
	fmt.Printf("  String(a).Compare(b):      %d\n", hyperex.String("a").Compare(hyperex.String("b")))

	fmt.Println("\n=== Index / Get ===")
	elem, err := vArr.Index(1)
	if err != nil {
		fmt.Printf("  Index(1) err: %v\n", err)
	} else {
		fmt.Printf("  Array.Index(1): %v (%s)\n", elem.Interface(), elem.Type())
	}
	if _, err := vArr.Index(99); err != nil {
		fmt.Printf("  Array.Index(99): %v\n", err)
	}
	if got, ok := vMap.Get("b"); ok {
		fmt.Printf("  Map.Get(\"b\"): %v\n", got.Interface())
	}
	if _, ok := vMap.Get("missing"); !ok {
		fmt.Println("  Map.Get(\"missing\"): not found")
	}
}
