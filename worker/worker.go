package worker

import (
	"context"
	"sync/atomic"

	"github.com/isucon/isucandar/parallel"
)

var (
	nopWorkFunc = func(_ context.Context, _ int) {}
)

type WorkerFunc func(context.Context, int)
type WorkerOption func(*Worker) error

type Worker struct {
	workFunc WorkerFunc
	count    int32
	parallel *parallel.Parallel
}

func NewWorker(f WorkerFunc, opts ...WorkerOption) (*Worker, error) {
	count := int32(-1)
	parallelism := int32(-1)

	if f == nil {
		f = nopWorkFunc
	}

	worker := &Worker{
		workFunc: f,
		count:    count,
		parallel: parallel.NewParallel(parallelism),
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

	parallel := w.parallel
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

	parallel := w.parallel
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
		w.parallel.Wait()
	}
}

func (w *Worker) SetLoopCount(count int32) {
	atomic.StoreInt32(&w.count, count)
}

func (w *Worker) SetParallelism(paralellism int32) {
	w.parallel.SetParallelism(paralellism)
}

func (w *Worker) AddParallelism(paralellism int32) {
	w.parallel.AddParallelism(paralellism)
}

func WithLoopCount(count int32) WorkerOption {
	return func(w *Worker) error {
		w.SetLoopCount(count)
		return nil
	}
}

func WithInfinityLoop() WorkerOption {
	return func(w *Worker) error {
		w.SetLoopCount(-1)
		return nil
	}
}

func WithMaxParallelism(parallelism int32) WorkerOption {
	return func(w *Worker) error {
		w.SetParallelism(parallelism)
		return nil
	}
}

func WithUnlimitedParallelism() WorkerOption {
	return func(w *Worker) error {
		w.SetParallelism(-1)
		return nil
	}
}
