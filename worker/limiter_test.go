package worker

import (
	"context"
	"testing"
	"time"
)

func TestWorkerLimiter(t *testing.T) {
	limiter := NewWorkerLimiter(2)
	defer limiter.Close()

	f := func(_ context.Context) {
		<-time.After(1 * time.Second)
	}

	now := time.Now()

	ctx := context.TODO()
	limiter.Do(ctx, f)
	limiter.Do(ctx, f)
	limiter.Do(ctx, f)
	limiter.Do(ctx, f)

	<-limiter.Wait()

	diff := time.Now().Sub(now)

	if diff >= 3*time.Second {
		t.Fatalf("process time: %s", diff)
	}
}

func TestWorkerLimiterClosed(t *testing.T) {
	limiter := NewWorkerLimiter(2)
	limiter.Close()

	ctx := context.TODO()

	called := false
	err := limiter.Do(ctx, func(_ context.Context) {
		called = true
	})

	<-limiter.Wait()

	if err == nil || err != ErrLimiterClosed {
		t.Fatalf("missmatch error: %+v", err)
	}

	if called {
		t.Fatalf("Do not process on closed")
	}
}

func TestWorkerLimiterDoneNotLock(t *testing.T) {
	limiter := NewWorkerLimiter(2)

	limiter.done()
	limiter.done()
	limiter.done()
	limiter.Close()
	limiter.done()
	limiter.done()

	<-limiter.Wait()
}
