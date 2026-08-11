// Common hyperex error sentinels and typed errors.
package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/aasyanov/hyperex"
)

func main() {
	ctx := context.Background()
	eng := hyperex.New()

	fmt.Println("=== ErrDivisionByZero ===")
	_, err := eng.Eval(ctx, `1/0`)
	fmt.Printf("  err: %v\n", err)
	fmt.Printf("  errors.Is(ErrDivisionByZero): %v\n", errors.Is(err, hyperex.ErrDivisionByZero))

	fmt.Println("\n=== ParseError (Line/Column) ===")
	_, err = eng.Compile(`((((`)
	fmt.Printf("  err: %v\n", err)
	var pe *hyperex.ParseError
	if errors.As(err, &pe) {
		fmt.Printf("  *ParseError Line=%d Column=%d\n", pe.Line(), pe.Column())
	}
	// Direct type assertion also works when the error is exactly *ParseError
	if pe2, ok := err.(*hyperex.ParseError); ok {
		fmt.Printf("  type assert Line=%d Column=%d\n", pe2.Line(), pe2.Column())
	}

	fmt.Println("\n=== UndefinedError ===")
	_, err = eng.Eval(ctx, `missing_var + 1`)
	fmt.Printf("  err: %v\n", err)
	var ue *hyperex.UndefinedError
	if errors.As(err, &ue) {
		fmt.Printf("  *UndefinedError Kind=%q Name=%q\n", ue.Kind, ue.Name)
	} else {
		fmt.Println("  (wrapped; errors.As may still find UndefinedError via Unwrap)")
		var re *hyperex.RuntimeError
		if errors.As(err, &re) {
			fmt.Printf("  *RuntimeError: %v\n", re)
		}
	}

	fmt.Println("\n=== ErrIndexOutOfBounds ===")
	_, err = eng.Eval(ctx, `[1,2,3][10]`)
	fmt.Printf("  err: %v\n", err)
	fmt.Printf("  errors.Is(ErrIndexOutOfBounds): %v\n", errors.Is(err, hyperex.ErrIndexOutOfBounds))

	fmt.Println("\n=== ErrNotIndexable ===")
	_, err = eng.Eval(ctx, `42[0]`)
	fmt.Printf("  errors.Is(ErrNotIndexable): %v  err=%v\n", errors.Is(err, hyperex.ErrNotIndexable), err)

	fmt.Println("\n=== LexerError ===")
	// Compile maps token errors into ParseError; LexerError is the typed form
	// returned by the tokenizer when used directly (also constructible).
	_, err = eng.Compile(`/* never closed`)
	var peLex *hyperex.ParseError
	if errors.As(err, &peLex) {
		fmt.Printf("  Compile of bad lexeme -> *ParseError Line=%d Column=%d\n", peLex.Line(), peLex.Column())
	}
	leDemo := &hyperex.LexerError{Message: "unterminated block comment", Line: 1, Column: 1, Source: "/*"}
	fmt.Printf("  *LexerError demo: %v\n", leDemo)

	fmt.Println("\n=== Other sentinels (errors.Is-ready) ===")
	fmt.Println("  Also defined: ErrInvalidOperation, ErrNilDereference, ErrNotCallable,")
	fmt.Println("  ErrArgumentCount, ErrInvalidType, ErrNotIterable, ErrKeyNotFound,")
	fmt.Println("  ErrInvalidSliceIndex, ErrIntegerOverflow, ErrInvalidRegex, ErrMaxCallDepth")
	fmt.Println("  (plus sandbox sentinels — see sandbox example)")

	fmt.Println("\n=== Constructed errors (optional) ===")
	re := hyperex.NewRuntimeErrorf("custom failure: %s", "demo")
	fmt.Printf("  NewRuntimeErrorf: %v\n", re)
	te := hyperex.NewTypeError("mismatch", "int", "string")
	fmt.Printf("  NewTypeError: %v\n", te)
}
