package worker

import (
	"context"
	"sync/atomic"
)

var (
	nopWorkFunc = func(_ context.Context, _ int) {}
)

type WorkerFunc func(context.Context, int)
type WorkerOption func(*Worker) error

type Worker struct {
	workFunc    WorkerFunc
	count       *int32
	parallelism *int32
}

func NewWorker(f WorkerFunc, opts ...WorkerOption) (*Worker, error) {
	count := int32(-1)
	parallelism := int32(-1)

	if f == nil {
		f = nopWorkFunc
	}

	worker := &Worker{
		workFunc:    f,
		count:       &count,
		parallelism: &parallelism,
	}

	for _, opt := range opts {
		err := opt(worker)
		if err != nil {
			return nil, err
		}
	}

	return worker, nil
}

func (w *Worker) Process(ctx context.Context) {
	count := atomic.LoadInt32(w.count)
	if count < 1 {
		w.processInfinity(ctx)
	} else {
		w.processLimited(ctx, int(count))
	}
}

func (w *Worker) processInfinity(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	limiter := w.limiter()
	defer limiter.Close()

	work := func(ctx context.Context) {
		w.workFunc(ctx, -1)
	}

	go func() {
		<-ctx.Done()
		limiter.Close()
	}()

	for {
		if err := limiter.Do(ctx, work); err != nil {
			return
		}
	}
}

func (w *Worker) processLimited(ctx context.Context, limit int) {
	if ctx.Err() != nil {
		return
	}

	limiter := w.limiter()
	defer limiter.Close()

	work := func(i int) func(context.Context) {
		return func(ctx context.Context) {
			w.workFunc(ctx, i)
		}
	}

	go func() {
		<-ctx.Done()
		limiter.Close()
	}()

	for i := 0; i < limit; i++ {
		if err := limiter.Do(ctx, work(i)); err != nil {
			return
		}
	}

	<-limiter.Wait()
}

func (w *Worker) limiter() *WorkerLimiter {
	p := atomic.LoadInt32(w.parallelism)
	limiter := NewWorkerLimiter(p)

	return limiter
}

func WithLoopCount(count int) WorkerOption {
	return func(w *Worker) error {
		atomic.StoreInt32(w.count, int32(count))
		return nil
	}
}

func WithInfinityLoop() WorkerOption {
	return func(w *Worker) error {
		atomic.StoreInt32(w.count, int32(-1))
		return nil
	}
}

func WithMaxParallelism(parallelism int) WorkerOption {
	return func(w *Worker) error {
		atomic.StoreInt32(w.parallelism, int32(parallelism))
		return nil
	}
}

func WithUnlimitedParallelism() WorkerOption {
	return func(w *Worker) error {
		atomic.StoreInt32(w.parallelism, int32(-1))
		return nil
	}
}
