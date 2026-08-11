// Package main demonstrates context cancellation during evaluation.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aasyanov/hyperex"
)

func main() {
	program, err := hyperex.Compile(`map(range(0, 1000000), # + 1)`)
	must(err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Eval starts
	_, err = program.Eval(ctx, nil)
	fmt.Printf("pre-cancelled:     err=%v\n", err)
	fmt.Printf("                   is ErrOperationCancelled=%v\n", errors.Is(err, hyperex.ErrOperationCancelled))

	eng := hyperex.New(hyperex.WithMaxOps(1_000_000_000))
	heavy, err := eng.Compile(`map(range(0, 100000), # + 1)`)
	must(err)

	ctx2, cancel2 := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Millisecond)
		cancel2()
	}()
	_, err = heavy.Eval(ctx2, nil)
	fmt.Printf("cancel mid-eval:   err=%v\n", err)
	fmt.Printf("                   is ErrOperationCancelled=%v\n", errors.Is(err, hyperex.ErrOperationCancelled))

	deadline, cancel3 := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel3()
	time.Sleep(time.Millisecond) // ensure deadline already expired
	_, err = heavy.Eval(deadline, nil)
	fmt.Printf("expired deadline:  err=%v\n", err)
	fmt.Printf("                   is ErrOperationCancelled=%v\n", errors.Is(err, hyperex.ErrOperationCancelled))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
