package parallel

import (
	"context"
	"errors"
	"sync/atomic"
)

var (
	ErrLimiterClosed = errors.New("limiter closed")
	ErrNegativeCount = errors.New("negative count")
)

const (
	closedFalse uint32 = iota
	closedTrue
)

type Parallel struct {
	limit  int32
	count  int32
	state  uint32
	closed uint32
}

func NewParallel(limit int32) *Parallel {
	return &Parallel{
		limit:  limit,
		count:  0,
		state:  0,
		closed: closedFalse,
	}
}

func (l *Parallel) Do(ctx context.Context, f func(context.Context)) error {
	err := l.start(ctx)
	if err != nil {
		return err
	}

	atomic.AddInt32(&l.count, 1)
	go func(state uint32) {
		defer l.done(state)
		defer func() {
			if atomic.LoadUint32(&l.state) == state {
				atomic.AddInt32(&l.count, -1)
			}
		}()
		f(ctx)
	}(atomic.LoadUint32(&l.state))

	return nil
}

func (l *Parallel) Wait() <-chan bool {
	ch := make(chan bool)

	go func(state uint32) {
		for atomic.LoadInt32(&l.count) > 0 && atomic.LoadUint32(&l.state) == state {
			// nop
		}
		close(ch)
	}(atomic.LoadUint32(&l.state))

	return ch
}

func (l *Parallel) Close() {
	atomic.StoreUint32(&l.closed, closedTrue)
}

func (l *Parallel) Reset() {
	atomic.StoreUint32(&l.closed, closedFalse)
	atomic.AddUint32(&l.state, 1)
	atomic.StoreInt32(&l.count, 0)
}

func (l *Parallel) SetParallelism(limit int32) {
	atomic.StoreInt32(&l.limit, limit)
}

func (l *Parallel) AddParallelism(limit int32) {
	atomic.AddInt32(&l.limit, limit)
}

func (l *Parallel) start(ctx context.Context) error {
	for l.isRunning(ctx) {
		if count, kept := l.isLimitKept(); kept {
			if atomic.CompareAndSwapInt32(&l.count, count, count+1) {
				return nil
			}
		}
	}

	return ErrLimiterClosed
}

func (l *Parallel) done(state uint32) {
	if atomic.LoadUint32(&l.state) == state {
		if atomic.AddInt32(&l.count, -1) < 0 {
			panic(ErrNegativeCount)
		}
	}
}

func (l *Parallel) isRunning(ctx context.Context) bool {
	return atomic.LoadUint32(&l.closed) == closedFalse && ctx.Err() == nil
}

func (l *Parallel) isLimitKept() (int32, bool) {
	limit := atomic.LoadInt32(&l.limit)
	count := atomic.LoadInt32(&l.count)
	return count, limit < 1 || count < (limit*2)
}
