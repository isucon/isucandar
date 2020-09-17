package main

import (
	"context"
	"fmt"
	"time"

	"github.com/isucon/isucandar/worker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeWorker, err := worker.NewWorker(func(ctx context.Context, _ int) {
		fmt.Println(time.Now().Format(time.RFC3339))
		time.Sleep(1 * time.Second)
	}, worker.WithInfinityLoop(), worker.WithMaxParallelism(1))
	if err != nil {
		panic(err)
	}

	increaseWorker, err := worker.NewWorker(func(ctx context.Context, _ int) {
		time.Sleep(3 * time.Second)
		fmt.Println("Increase time worker!")
		timeWorker.AddParallelism(1)
	}, worker.WithLoopCount(3), worker.WithMaxParallelism(1))
	if err != nil {
		panic(err)
	}

	go func() {
		time.Sleep(10 * time.Second)
		cancel()
	}()

	go func() {
		increaseWorker.Process(ctx)
		fmt.Println("Increase worker executed")
	}()
	timeWorker.Process(ctx)
	fmt.Println("Time worker executed")
}
