package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrLimiterClosed = errors.New("limiter closed")
)

const (
	MaxParallelism = 5000
)

type WorkerLimiter struct {
	mu     sync.RWMutex
	count  *int32
	limit  int32
	closed bool
}

func NewWorkerLimiter(limit int32) *WorkerLimiter {
	if limit <= 0 {
		limit = MaxParallelism
	}

	count := int32(0)
	return &WorkerLimiter{
		count:  &count,
		limit:  limit,
		mu:     sync.RWMutex{},
		closed: false,
	}
}

func (l *WorkerLimiter) Do(ctx context.Context, f func(context.Context)) error {
	err := l.start()
	if err == ErrLimiterClosed {
		return err
	}

	go func() {
		defer l.done()
		f(ctx)
	}()

	return nil
}

func (l *WorkerLimiter) Wait() <-chan bool {
	ch := make(chan bool)

	go func() {
		for atomic.LoadInt32(l.count) > 0 {
			// nop
		}
		close(ch)
	}()

	return ch
}

func (l *WorkerLimiter) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	atomic.StoreInt32(l.count, 0)
}

func (l *WorkerLimiter) start() error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for {
		if l.closed {
			return ErrLimiterClosed
		}

		if count := atomic.LoadInt32(l.count); count < l.limit {
			if atomic.CompareAndSwapInt32(l.count, count, count+1) {
				return nil
			}
		}
	}
}

func (l *WorkerLimiter) done() {
	atomic.AddInt32(l.count, -1)
}
