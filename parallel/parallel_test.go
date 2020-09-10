package parallel

import (
	"context"
	"testing"
	"time"
)

func TestParallel(t *testing.T) {
	parallel := NewParallel(2)
	defer parallel.Close()

	f := func(_ context.Context) {
		time.Sleep(1 * time.Millisecond)
	}

	now := time.Now()

	ctx := context.TODO()
	parallel.Do(ctx, f)
	parallel.Do(ctx, f)
	parallel.Do(ctx, f)
	parallel.Do(ctx, f)

	<-parallel.Wait()

	diff := time.Now().Sub(now)

	if diff >= (3*time.Millisecond + 500*time.Microsecond) {
		t.Fatalf("process time: %s", diff)
	}
}

func TestParallelClosed(t *testing.T) {
	parallel := NewParallel(2)
	parallel.Close()

	ctx := context.TODO()

	called := false
	err := parallel.Do(ctx, func(_ context.Context) {
		called = true
	})

	<-parallel.Wait()

	if err == nil || err != ErrLimiterClosed {
		t.Fatalf("missmatch error: %+v", err)
	}

	if called {
		t.Fatalf("Do not process on closed")
	}
}

func TestParallelUnlimited(t *testing.T) {
	parallel := NewParallel(0)

	if parallel.limit != 0 {
		t.Fatalf("Invalid limit: %d", parallel.limit)
	}
}

func TestParallelCanceled(t *testing.T) {
	parallel := NewParallel(0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	parallel.Do(ctx, func(_ context.Context) {
		t.Fatal("Do not call")
	})

	<-parallel.Wait()
}

func TestParallelPanicOnNegative(t *testing.T) {
	parallel := NewParallel(0)

	var err interface{}
	func() {
		defer func() { err = recover() }()
		parallel.done(parallel.state)
	}()

	if err != ErrNegativeCount {
		t.Fatal(err)
	}
}

func TestParallelSetParallelism(t *testing.T) {
	parallel := NewParallel(0)

	f := func(c context.Context) {
		time.Sleep(1 * time.Millisecond)
	}

	check := func(expectTime time.Duration) {
		parallel.Reset()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := time.Now()
		parallel.Do(ctx, f)
		parallel.Do(ctx, f)
		parallel.Do(ctx, f)
		parallel.Do(ctx, f)
		<-parallel.Wait()

		parallel.Wait()
		diff := time.Now().Sub(now)

		if diff > expectTime {
			t.Fatalf("longer execution time: %s / %s", diff, expectTime)
		}
	}

	parallel.SetParallelism(2)
	check(3*time.Millisecond + 500*time.Microsecond)

	parallel.AddParallelism(-1)
	check(6 * time.Millisecond)

	parallel.AddParallelism(-1)
	check(1*time.Millisecond + 500*time.Microsecond)
}
