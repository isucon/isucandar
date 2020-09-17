package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/isucon/isucandar/failure"
)

var (
	ErrDeepCall failure.StringCode = "DEEP"
)

func deepError(n int) error {
	if n > 0 {
		return deepError(n - 1)
	} else {
		return failure.NewError(ErrDeepCall, fmt.Errorf("error"))
	}
}

func main() {
	failure.BacktraceCleaner.Add(failure.SkipGOROOT)

	ctx, cancel := context.WithCancel(context.Background())
	errors := failure.NewErrors(ctx)

	errors.Add(deepError(rand.Intn(5)))
	errors.Add(deepError(rand.Intn(5)))
	errors.Add(deepError(rand.Intn(5)))
	cancel()

	errors.Wait()

	for _, err := range errors.All() {
		fmt.Printf("%+v\n", err)
	}
}
