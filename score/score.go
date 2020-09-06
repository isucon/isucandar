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
	Table ScoreTable

	total  sumTable
	queue  chan ScoreTag
	wg     sync.WaitGroup
	closer sync.Once
}

func NewScore(ctx context.Context) *Score {
	score := &Score{
		Table:  make(ScoreTable),
		total:  make(sumTable),
		queue:  make(chan ScoreTag),
		wg:     sync.WaitGroup{},
		closer: sync.Once{},
	}

	go score.collect(ctx)

	return score
}

func (s *Score) add(tag ScoreTag) {
	if ptr, ok := s.total[tag]; ok {
		atomic.AddInt64(ptr, 1)
	} else {
		n := int64(1)
		s.total[tag] = &n
	}
}

func (s *Score) collect(ctx context.Context) {
	defer s.wg.Done()
	s.wg.Add(1)

	for {
		select {
		case tag, ok := <-s.queue:
			if !ok {
				return
			}
			s.add(tag)
		case <-ctx.Done():
			// Context による終了ではキューを閉じない
			s.closer.Do(func() {
				close(s.queue)
			})
			return // コンテキストが終了したので離脱
		}
	}
}

func (s *Score) Set(tag ScoreTag, mag int64) {
	s.Table[tag] = mag
}

func (s *Score) Add(tag ScoreTag) {
	defer func() {
		// catch error of "send on closed channel"
		recover()
	}()
	s.queue <- tag
}

func (s *Score) Wait() {
	s.wg.Wait()
}

func (s *Score) Done() {
	s.closer.Do(func() {
		close(s.queue)
	})
	s.Wait()
}

func (s *Score) Breakdown() ScoreTable {
	table := make(ScoreTable)
	for tag, ptr := range s.total {
		table[tag] = atomic.LoadInt64(ptr)
	}
	return table
}

func (s *Score) Sum() int64 {
	sum := int64(0)
	for tag, ptr := range s.total {
		if mag, found := s.Table[tag]; found {
			sum += atomic.LoadInt64(ptr) * mag
		} else {
			sum += atomic.LoadInt64(ptr)
		}
	}
	return sum
}

func (s *Score) Total() int64 {
	s.Done()
	return s.Sum()
}
