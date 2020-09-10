package pubsub

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPubSub(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pubsub := NewPubSub()

	wg := sync.WaitGroup{}

	result1 := int32(0)
	pubsub.Subscribe(ctx, func(payload interface{}) {
		atomic.AddInt32(&result1, int32(payload.(int)))
		wg.Done()
	})
	result2 := int32(0)
	pubsub.Subscribe(ctx, func(payload interface{}) {
		atomic.AddInt32(&result2, int32(payload.(int)))
		wg.Done()
	})

	wg.Add(4)
	pubsub.Publish(1)
	pubsub.Publish(2)

	wg.Wait()

	if result1 != 3 {
		t.Fatalf("invalid 1: %v", result1)
	}
	if result2 != 3 {
		t.Fatalf("invalid 2: %v", result2)
	}
}

func TestPubSubUnsubscribe(t *testing.T) {
	pubsub := NewPubSub()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	<-pubsub.Subscribe(ctx, func(payload interface{}) {})
}
