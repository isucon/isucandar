package parallel

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestParallel(t *testing.T) {
	ctx := context.TODO()

	parallel := NewParallel(ctx, 2)
	defer parallel.Close()

	pcount := int32(0)
	pmcount := int32(0)
	exited := uint32(0)
	f := func(_ context.Context) {
		atomic.AddInt32(&pcount, 1)
		defer atomic.AddInt32(&pcount, -1)
		time.Sleep(10 * time.Millisecond)
	}

	parallel.Do(f)
	go func() {
		parallel.Do(f)
		parallel.Do(f)
		parallel.Do(f)
	}()

	go func() {
		for atomic.LoadUint32(&exited) == 0 {
			m := atomic.LoadInt32(&pcount)
			if atomic.LoadInt32(&pmcount) < m {
				atomic.StoreInt32(&pmcount, m)
			}
		}
	}()

	parallel.Wait()
	atomic.StoreUint32(&exited, 1)

	maxCount := atomic.LoadInt32(&pmcount)
	if maxCount != 2 {
		t.Fatalf("Invalid parallel count: %d / %d", maxCount, 2)
	}
}

func TestParallelClosed(t *testing.T) {
	ctx := context.TODO()

	parallel := NewParallel(ctx, 2)
	parallel.Close()

	called := false
	err := parallel.Do(func(_ context.Context) {
		called = true
	})

	parallel.Wait()

	if err == nil || err != ErrLimiterClosed {
		t.Fatalf("missmatch error: %+v", err)
	}

	if called {
		t.Fatalf("Do not process on closed")
	}
}

func TestParallelCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	parallel := NewParallel(ctx, 0)

	parallel.Do(func(_ context.Context) {
		t.Fatal("Do not call")
	})

	parallel.Wait()
}

func TestParallelPanicOnNegative(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	parallel := NewParallel(ctx, 0)

	var err interface{}
	func() {
		defer func() { err = recover() }()
		parallel.done(nil)
	}()

	if err != ErrNegativeCount {
		t.Fatal(err)
	}
}

func TestParallelSetParallelism(t *testing.T) {
	check := func(paralellism int32) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		parallel := NewParallel(ctx, -1)
		parallel.SetParallelism(paralellism)

		pcount := int32(0)
		pmcount := int32(0)
		exited := uint32(0)
		f := func(c context.Context) {
			atomic.AddInt32(&pcount, 1)
			defer atomic.AddInt32(&pcount, -1)

			time.Sleep(10 * time.Millisecond)
		}

		parallel.Do(f)
		go func() {
			parallel.Do(f)
			parallel.Do(f)
			parallel.Do(f)
		}()

		go func() {
			for atomic.LoadUint32(&exited) == 0 {
				m := atomic.LoadInt32(&pcount)
				if atomic.LoadInt32(&pmcount) < m {
					atomic.StoreInt32(&pmcount, m)
				}
			}
		}()
		parallel.Wait()
		atomic.StoreUint32(&exited, 1)

		maxCount := atomic.LoadInt32(&pmcount)
		if maxCount != parallel.CurrentLimit() && parallel.CurrentLimit() > 0 {
			t.Fatalf("Invalid parallel count: %d / %d", maxCount, parallel.CurrentLimit())
		}

		parallel.Wait()
	}

	check(2)
	check(1)
	check(-1)
}

func TestParallelAddParallelism(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	para := NewParallel(ctx, 1)
	para.AddParallelism(1)

	pcount := int32(0)
	pmcount := int32(0)
	exited := uint32(0)
	f := func(c context.Context) {
		atomic.AddInt32(&pcount, 1)
		defer atomic.AddInt32(&pcount, -1)

		time.Sleep(10 * time.Millisecond)
	}

	para.Do(f)
	go func() {
		para.Do(f)
		para.Do(f)
		para.Do(f)
	}()

	go func() {
		for atomic.LoadUint32(&exited) == 0 {
			m := atomic.LoadInt32(&pcount)
			if atomic.LoadInt32(&pmcount) < m {
				atomic.StoreInt32(&pmcount, m)
			}
		}
	}()
	para.Wait()
	atomic.StoreUint32(&exited, 1)

	maxCount := atomic.LoadInt32(&pmcount)
	if maxCount != 2 {
		t.Fatalf("Invalid parallel count: %d / %d", maxCount, para.CurrentLimit())
	}
}

func BenchmarkParallel(b *testing.B) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	parallel := NewParallel(ctx, -1)
	nop := func(_ context.Context) {}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parallel.Do(nop)
	}
	parallel.Wait()
	b.StopTimer()
}
