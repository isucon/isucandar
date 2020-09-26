package parallel

import (
	"context"
	"errors"
	"sync"
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
	mu     sync.Mutex
	ctx    context.Context
	limit  int32
	count  int32
	closed uint32
	closer chan struct{}
	doner  chan struct{}
}

func NewParallel(ctx context.Context, limit int32) *Parallel {
	var doner chan struct{} = nil
	if limit > 0 {
		doner = make(chan struct{}, limit)
	}

	p := &Parallel{
		mu:     sync.Mutex{},
		ctx:    ctx,
		limit:  limit,
		count:  0,
		closed: closedFalse,
		closer: make(chan struct{}),
		doner:  doner,
	}

	return p
}

func (l *Parallel) CurrentLimit() int32 {
	return atomic.LoadInt32(&l.limit)
}

func (l *Parallel) Do(f func(context.Context)) error {
	atomic.AddInt32(&l.count, 1)

	err := l.start()
	if err != nil {
		atomic.AddInt32(&l.count, -1)
		return err
	}

	l.mu.Lock()
	doner := l.doner
	l.mu.Unlock()

	go func(doner chan struct{}) {
		defer l.done(doner)
		f(l.ctx)
	}(doner)

	return nil
}

func (l *Parallel) Wait() {
	if atomic.LoadUint32(&l.closed) != closedTrue {
		for {
			select {
			case <-l.ctx.Done():
				l.Close()
			case <-l.closer:
				return
			}
		}
	}
}

func (l *Parallel) Close() {
	if atomic.CompareAndSwapUint32(&l.closed, closedFalse, closedTrue) {
		close(l.closer)
	}
}

func (l *Parallel) SetParallelism(limit int32) {
	l.mu.Lock()
	defer l.mu.Unlock()
	atomic.StoreInt32(&l.limit, limit)
	if l.doner != nil {
		close(l.doner)
	}

	if limit > 0 {
		l.doner = make(chan struct{}, limit)
	} else {
		l.doner = nil
	}
}

func (l *Parallel) AddParallelism(limit int32) {
	l.SetParallelism(atomic.LoadInt32(&l.limit) + limit)
}

func (l *Parallel) start() error {
	for l.isRunning() {
		if count, limit, kept := l.isLimitKept(); kept {
			if atomic.CompareAndSwapInt32(&l.count, count, count+1) {
				return nil
			}
		} else if limit > 0 {
			l.mu.Lock()
			l.doner <- struct{}{}
			l.mu.Unlock()
		}
	}

	return ErrLimiterClosed
}

func (l *Parallel) done(doner chan struct{}) {
	select {
	case <-doner:
	default:
	}

	count := atomic.AddInt32(&l.count, -2)
	if count < 0 {
		panic(ErrNegativeCount)
	}
	if count == 0 {
		l.Close()
	}
}

func (l *Parallel) isRunning() bool {
	return atomic.LoadUint32(&l.closed) == closedFalse && l.ctx.Err() == nil
}

func (l *Parallel) isLimitKept() (int32, int32, bool) {
	limit := atomic.LoadInt32(&l.limit)
	count := atomic.LoadInt32(&l.count)
	return count, limit, limit < 1 || count < (limit*2)
}
