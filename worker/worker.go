package worker

import (
	"context"
	"github.com/rosylilly/isucandar/parallel"
	"sync/atomic"
)

var (
	nopWorkFunc = func(_ context.Context, _ int) {}
)

type WorkerFunc func(context.Context, int)
type WorkerOption func(*Worker) error

type Worker struct {
	workFunc    WorkerFunc
	count       int32
	parallelism int32
	parallel    *parallel.Parallel
}

func NewWorker(f WorkerFunc, opts ...WorkerOption) (*Worker, error) {
	count := int32(-1)
	parallelism := int32(-1)

	if f == nil {
		f = nopWorkFunc
	}

	worker := &Worker{
		workFunc:    f,
		count:       count,
		parallelism: parallelism,
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
	count := atomic.LoadInt32(&w.count)
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

	parallel := w.getParallel()
	parallel.Reset()
	defer parallel.Close()

	work := func(ctx context.Context) {
		w.workFunc(ctx, -1)
	}

L:
	for {
		select {
		case <-ctx.Done():
			break L
		default:
			parallel.Do(ctx, work)
		}
	}

	w.Wait()
}

func (w *Worker) processLimited(ctx context.Context, limit int) {
	if ctx.Err() != nil {
		return
	}

	parallel := w.getParallel()
	parallel.Reset()
	defer parallel.Close()

	work := func(i int) func(context.Context) {
		return func(ctx context.Context) {
			w.workFunc(ctx, i)
		}
	}

L:
	for i := 0; i < limit; i++ {
		select {
		case <-ctx.Done():
			break L
		default:
			parallel.Do(ctx, work(i))
		}
	}

	w.Wait()
}

func (w *Worker) Wait() {
	if w.parallel != nil {
		<-w.parallel.Wait()
	}
}

func (w *Worker) SetLoopCount(count int32) {
	atomic.StoreInt32(&w.count, count)
}

func (w *Worker) SetParallelism(paralellism int32) {
	atomic.StoreInt32(&w.parallelism, paralellism)
	if w.parallel != nil {
		w.parallel.SetParallelism(paralellism)
	}
}

func (w *Worker) AddParallelism(paralellism int32) {
	atomic.AddInt32(&w.parallelism, paralellism)
	if w.parallel != nil {
		w.parallel.AddParallelism(paralellism)
	}
}

func (w *Worker) getParallel() *parallel.Parallel {
	if w.parallel == nil {
		p := atomic.LoadInt32(&w.parallelism)
		parallel := parallel.NewParallel(p)
		w.parallel = parallel
	}

	return w.parallel
}

func WithLoopCount(count int32) WorkerOption {
	return func(w *Worker) error {
		atomic.StoreInt32(&w.count, count)
		return nil
	}
}

func WithInfinityLoop() WorkerOption {
	return func(w *Worker) error {
		atomic.StoreInt32(&w.count, int32(-1))
		return nil
	}
}

func WithMaxParallelism(parallelism int32) WorkerOption {
	return func(w *Worker) error {
		atomic.StoreInt32(&w.parallelism, parallelism)
		return nil
	}
}

func WithUnlimitedParallelism() WorkerOption {
	return func(w *Worker) error {
		atomic.StoreInt32(&w.parallelism, int32(-1))
		return nil
	}
}
