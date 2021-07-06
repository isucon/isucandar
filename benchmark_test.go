package isucandar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/isucon/isucandar/failure"
)

var (
	ErrIgnore          failure.StringCode = "ignore"
	ErrBenchmarkCancel failure.StringCode = "banchmark-cancel"
)

func newBenchmark(opts ...BenchmarkOption) *Benchmark {
	benchmark, err := NewBenchmark(opts...)
	if err != nil {
		panic(err)
	}

	benchmark.IgnoreErrorCode(ErrIgnore)

	benchmark.Prepare(func(ctx context.Context, s *BenchmarkStep) error {
		time.Sleep(1 * time.Microsecond)
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})

	benchmark.Load(func(ctx context.Context, s *BenchmarkStep) error {
		time.Sleep(1 * time.Microsecond)
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})

	benchmark.Validation(func(ctx context.Context, s *BenchmarkStep) error {
		time.Sleep(1 * time.Microsecond)
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})

	return benchmark
}

func TestBenchmark(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	result := b.Start(ctx)

	if len(result.Errors.All()) != 0 {
		t.Fatal(result.Errors.All())
	}
}

func TestBenchmarkScore(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	b.Load(func(_ context.Context, s *BenchmarkStep) error {
		s.AddScore("dummy")
		return nil
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) != 0 {
		t.Fatal(result.Errors.All())
	}

	result.Score.Set("dummy", 1)

	if result.Score.Sum() != 1 {
		t.Fatalf("%d", result.Score.Sum())
	}
}

func TestBenchmarkCreation(t *testing.T) {
	raise := errors.New("error")
	_, err := NewBenchmark(func(b *Benchmark) error {
		return raise
	})

	if err != raise {
		t.Fatal(err)
	}
}

func TestBenchmarkErrorHook(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()
	b.OnError(func(err error, s *BenchmarkStep) {
		if failure.IsCode(err, ErrBenchmarkCancel) {
			s.Cancel()
		}
	})

	b.Prepare(func(_ context.Context, s *BenchmarkStep) error {
		s.AddError(failure.NewError(ErrBenchmarkCancel, errors.New("cancel")))
		return nil
	})

	loaded := false
	b.Load(func(_ context.Context, _ *BenchmarkStep) error {
		loaded = true
		return nil
	})

	b.Start(ctx)

	if loaded {
		t.Fatal("error hook error")
	}
}

func TestBenchmarkPrepareTimeout(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark(WithPrepareTimeout(1))

	result := b.Start(ctx)

	if len(result.Errors.All()) != 1 || !failure.Is(result.Errors.All()[0], context.DeadlineExceeded) {
		t.Fatal(result.Errors.All())
	}
}

func TestBenchmarkPreparePanic(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	b.Prepare(func(_ context.Context, _ *BenchmarkStep) error {
		panic("Prepare panic")
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) != 1 || !failure.IsCode(result.Errors.All()[0], ErrPanic) {
		t.Fatal(result.Errors.All())
	}
}

func TestBenchmarkPreparePanicError(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	err := errors.New("Prepare panic")
	b.Prepare(func(_ context.Context, _ *BenchmarkStep) error {
		panic(err)
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) != 1 || !failure.Is(result.Errors.All()[0], err) {
		t.Fatal(result.Errors.All())
	}
}

func TestBenchmarkPrepareIgnoredError(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	err := failure.NewError(ErrIgnore, errors.New("Prepare panic"))
	b.Prepare(func(_ context.Context, _ *BenchmarkStep) error {
		return err
	})

	loaded := false
	b.Load(func(_ context.Context, _ *BenchmarkStep) error {
		loaded = true
		return nil
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) != 0 {
		t.Fatal(result.Errors.All())
	}

	if loaded {
		t.Fatal("ignore error")
	}
}

func TestBenchmarkPrepareCancel(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	b.Prepare(func(_ context.Context, s *BenchmarkStep) error {
		s.Cancel()
		return nil
	})

	loaded := false
	b.Load(func(_ context.Context, _ *BenchmarkStep) error {
		loaded = true
		return nil
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) > 1 {
		t.Fatal(result.Errors.All())
	}

	if loaded {
		t.Fatal("cancel error")
	}
}

func TestBenchmarkLoadTimeout(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark(WithLoadTimeout(5 * time.Millisecond))

	runAll := false
	b.Load(func(ctx context.Context, _ *BenchmarkStep) error {
		time.Sleep(100 * time.Millisecond)
		runAll = true
		return nil
	})
	b.Start(ctx)

	if runAll {
		t.Fatal("Not timeout")
	}
}

func TestBenchmarkLoadPanic(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	b.Load(func(_ context.Context, _ *BenchmarkStep) error {
		panic("Load panic")
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) != 1 || !failure.IsCode(result.Errors.All()[0], ErrPanic) {
		t.Fatal(result.Errors.All())
	}
	t.Log(result.Errors.All())
}

func TestBenchmarkLoadPanicError(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	err := errors.New("Load panic")
	b.Load(func(_ context.Context, _ *BenchmarkStep) error {
		panic(err)
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) != 1 || !failure.Is(result.Errors.All()[0], err) {
		t.Fatal(result.Errors.All())
	}
}

func TestBenchmarkLoadIgnoredError(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	err := failure.NewError(ErrIgnore, errors.New("Prepare panic"))
	b.Load(func(_ context.Context, _ *BenchmarkStep) error {
		return err
	})

	loaded := false
	b.Validation(func(_ context.Context, _ *BenchmarkStep) error {
		loaded = true
		return nil
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) != 0 {
		t.Fatal(result.Errors.All())
	}

	if !loaded {
		t.Fatal("ignore error")
	}
}

func TestBenchmarkLoadCancel(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	b.Load(func(_ context.Context, s *BenchmarkStep) error {
		s.Cancel()
		return nil
	})

	loaded := false
	b.Validation(func(_ context.Context, _ *BenchmarkStep) error {
		loaded = true
		return nil
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) > 1 {
		t.Fatal(result.Errors.All())
	}

	if loaded {
		t.Fatal("cancel error")
	}
}

func TestBenchmarkValidationPanic(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	b.Validation(func(_ context.Context, _ *BenchmarkStep) error {
		panic("Validation panic")
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) != 1 || !failure.IsCode(result.Errors.All()[0], ErrPanic) {
		t.Fatal(result.Errors.All())
	}
}

func TestBenchmarkValidationPanicError(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	err := errors.New("Validation panic")
	b.Validation(func(_ context.Context, _ *BenchmarkStep) error {
		panic(err)
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) != 1 || !failure.Is(result.Errors.All()[0], err) {
		t.Fatal(result.Errors.All())
	}
}

func TestBenchmarkValidationIgnoredError(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark()

	err := failure.NewError(ErrIgnore, errors.New("Prepare panic"))
	b.Validation(func(_ context.Context, _ *BenchmarkStep) error {
		return err
	})

	result := b.Start(ctx)

	if len(result.Errors.All()) != 0 {
		t.Fatal(result.Errors.All())
	}
}

func TestBenchmarWithoutPanicRecover(t *testing.T) {
	ctx := context.TODO()
	b := newBenchmark(WithoutPanicRecover())

	panicErr := errors.New("panic")
	b.Validation(func(_ context.Context, _ *BenchmarkStep) error {
		panic(panicErr)
	})

	func() {
		defer func() {
			err := recover()
			if err == nil {
				t.Fatalf("not thrown panic")
			}

			if err != panicErr {
				t.Fatalf("invalid panic: %+v", err)
			}
		}()

		b.Start(ctx)
	}()
}
