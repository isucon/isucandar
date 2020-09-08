package parallel

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

var (
	ErrLimiterClosed = errors.New("limiter closed")
)

const (
	MaxParallelism = 5000
)

type Parallel struct {
	limit  int32
	count  int32
	closed uint32
}

func NewParallel(limit int32) *Parallel {
	if limit <= 0 {
		limit = MaxParallelism
	}

	return &Parallel{
		limit:  limit,
		count:  0,
		closed: 0,
	}
}

func (l *Parallel) Do(ctx context.Context, f func(context.Context)) error {
	err := l.start()
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		l.done()
		return nil
	}

	go func() {
		defer func() {
			l.done()
		}()
		f(ctx)
	}()

	return nil
}

func (l *Parallel) Wait() <-chan bool {
	ch := make(chan bool)

	go func() {
		for atomic.LoadInt32(&l.count) > 0 {
			// nop
		}
		close(ch)
	}()

	return ch
}

func (l *Parallel) Close() {
	atomic.StoreUint32(&l.closed, 1)
}

func (l *Parallel) Reset() {
	atomic.StoreUint32(&l.closed, 0)
	atomic.StoreInt32(&l.count, 0)
}

func (l *Parallel) SetParallelism(parallel int32) {
	atomic.StoreInt32(&l.limit, parallel)
}

func (l *Parallel) start() error {
	for atomic.LoadUint32(&l.closed) == 0 {
		if count := atomic.LoadInt32(&l.count); count < atomic.LoadInt32(&l.limit) {
			if atomic.CompareAndSwapInt32(&l.count, count, count+1) {
				return nil
			}
		}

		time.Sleep(-1)
	}

	return ErrLimiterClosed
}

func (l *Parallel) done() {
	atomic.AddInt32(&l.count, -1)
}
