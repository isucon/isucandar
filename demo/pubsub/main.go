package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/isucon/isucandar/pubsub"
	"github.com/isucon/isucandar/worker"
)

func launchWorker(ctx context.Context, pubsub *pubsub.PubSub, format string) error {
	worker, err := worker.NewWorker(func(_ context.Context, _ int) {
		fmt.Println(time.Now().Format(format))
		time.Sleep(time.Second)
	}, worker.WithMaxParallelism(1))
	if err != nil {
		return err
	}

	go worker.Process(ctx)

	<-pubsub.Subscribe(ctx, func(limit interface{}) {
		l := limit.(int32)
		fmt.Printf("Worker increase: %d\n", l)
		worker.AddParallelism(l)
	})

	return nil
}

func main() {
	p := pubsub.NewPubSub()

	wg := sync.WaitGroup{}
	wg.Add(3)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		for i := 1; i < 3; i++ {
			time.Sleep(1 * time.Second)
			p.Publish(int32(i))
		}
	}()

	go func() {
		launchWorker(ctx, p, time.RFC822)
		wg.Done()
	}()
	go func() {
		launchWorker(ctx, p, time.RFC850)
		wg.Done()
	}()
	go func() {
		launchWorker(ctx, p, time.RFC3339)
		wg.Done()
	}()

	wg.Wait()
}
