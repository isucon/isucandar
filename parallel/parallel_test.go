package parallel

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestParallel(t *testing.T) {
	parallel := NewParallel(2)
	defer parallel.Close()

	mu := sync.Mutex{}
	var latestExecutionTime time.Time
	f := func(_ context.Context) {
		time.Sleep(1 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		latestExecutionTime = time.Now()
	}

	ctx := context.TODO()

	parallel.Do(ctx, f)
	parallel.Do(ctx, f)
	parallel.Do(ctx, f)
	parallel.Do(ctx, f)

	now := time.Now()
	<-parallel.Wait()

	mu.Lock()
	diff := latestExecutionTime.Sub(now)
	mu.Unlock()

	if diff >= (3*time.Millisecond + 500*time.Microsecond) {
		t.Fatalf("process time: %s", diff)
	}
	t.Logf("Execution time: %s", diff)
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

	check := func(expectTime time.Duration) {
		parallel.Reset()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mu := sync.Mutex{}
		var latestExecutionTime time.Time
		f := func(c context.Context) {
			time.Sleep(1 * time.Millisecond)
			mu.Lock()
			latestExecutionTime = time.Now()
			mu.Unlock()
		}

		parallel.Do(ctx, f)
		parallel.Do(ctx, f)
		parallel.Do(ctx, f)
		parallel.Do(ctx, f)
		now := time.Now()

		<-parallel.Wait()

		mu.Lock()
		diff := latestExecutionTime.Sub(now)
		mu.Unlock()

		if diff > expectTime {
			t.Fatalf("longer execution time: %s / %s", diff, expectTime)
		}
		t.Logf("Pass with execution time: %s / %s", diff, expectTime)

		<-parallel.Wait()
	}

	parallel.SetParallelism(2)
	check(3 * time.Millisecond)

	parallel.AddParallelism(-1)
	check(6 * time.Millisecond)

	parallel.AddParallelism(-1)
	unlimitedCount := 4 / runtime.GOMAXPROCS(0)
	if unlimitedCount <= 0 {
		unlimitedCount = 1
	}
	check(time.Duration(unlimitedCount)*time.Millisecond + (time.Duration(unlimitedCount) * 500 * time.Microsecond))
}

func BenchmarkParallel(b *testing.B) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	parallel := NewParallel(-1)
	nop := func(_ context.Context) {}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parallel.Do(ctx, nop)
	}
	<-parallel.Wait()
	b.StopTimer()
}
