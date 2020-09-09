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

	if diff >= 3*time.Millisecond {
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

func TestParallelDoneNotLock(t *testing.T) {
	parallel := NewParallel(2)

	parallel.done()
	parallel.done()
	parallel.done()
	parallel.Close()
	parallel.done()
	parallel.done()

	<-parallel.Wait()
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
	check(3 * time.Millisecond)

	parallel.SetParallelism(1)
	check(6 * time.Millisecond)

	parallel.SetParallelism(0)
	check(2 * time.Millisecond)
}
