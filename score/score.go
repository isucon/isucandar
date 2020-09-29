package score

import (
	"context"
	"sync"
	"sync/atomic"
)

type ScoreTag string
type ScoreTable map[ScoreTag]int64

type sumTable map[ScoreTag]*int64

type Score struct {
	Table                     ScoreTable
	DefaultScoreMagnification int64

	mu     sync.RWMutex
	total  sumTable
	count  int32
	queue  chan ScoreTag
	closed uint32
}

func NewScore(ctx context.Context) *Score {
	score := &Score{
		Table:                     make(ScoreTable),
		DefaultScoreMagnification: 0,
		mu:                        sync.RWMutex{},
		total:                     make(sumTable),
		count:                     0,
		queue:                     make(chan ScoreTag),
		closed:                    0,
	}

	go score.collect(ctx)

	return score
}

func (s *Score) add(tag ScoreTag) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ptr, ok := s.total[tag]; ok {
		atomic.AddInt64(ptr, 1)
	} else {
		n := int64(1)
		s.total[tag] = &n
	}
}

func (s *Score) collect(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.Close()
	}()

	for tag := range s.queue {
		s.add(tag)
		atomic.AddInt32(&s.count, -1)
	}
	atomic.AddInt32(&s.count, -1)
}

func (s *Score) Set(tag ScoreTag, mag int64) {
	s.Table[tag] = mag
}

func (s *Score) Add(tag ScoreTag) {
	defer func() { recover() }()

	if atomic.CompareAndSwapUint32(&s.closed, 0, 0) {
		s.queue <- tag
		atomic.AddInt32(&s.count, 1)
	}
}

func (s *Score) Close() {
	if atomic.CompareAndSwapUint32(&s.closed, 0, 1) {
		atomic.AddInt32(&s.count, 1)
		close(s.queue)
	}
}

func (s *Score) Wait() {
	for atomic.LoadInt32(&s.count) > 0 {
	}
}

func (s *Score) Done() {
	s.Close()
	s.Wait()
}

func (s *Score) Breakdown() ScoreTable {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table := make(ScoreTable)
	for tag, ptr := range s.total {
		table[tag] = atomic.LoadInt64(ptr)
	}
	return table
}

func (s *Score) Sum() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sum := int64(0)
	for tag, ptr := range s.total {
		if mag, found := s.Table[tag]; found {
			sum += atomic.LoadInt64(ptr) * mag
		} else {
			sum += atomic.LoadInt64(ptr) * s.DefaultScoreMagnification
		}
	}
	return sum
}

func (s *Score) Total() int64 {
	s.Done()
	return s.Sum()
}

func (s *Score) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.total = make(sumTable)
}
