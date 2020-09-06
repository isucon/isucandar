package failure

import (
	"context"
	"sync"
)

type Set struct {
	wg     sync.WaitGroup
	mu     sync.RWMutex
	closer sync.Once
	errors []error
	queue  chan error
}

func NewSet(ctx context.Context) *Set {
	set := &Set{
		wg:     sync.WaitGroup{},
		mu:     sync.RWMutex{},
		closer: sync.Once{},
		errors: make([]error, 0, 0),
		queue:  make(chan error),
	}

	go set.collect(ctx)

	return set
}

func (s *Set) collect(ctx context.Context) {
	s.wg.Add(1)
	defer s.wg.Done()

	go func() {
		<-ctx.Done()
		s.closer.Do(func() { close(s.queue) })
	}()

	for err := range s.queue {
		s.mu.Lock()
		s.errors = append(s.errors, err)
		s.mu.Unlock()
	}
}

func (s *Set) Add(err error) {
	defer func() { recover() }()
	s.queue <- err
}

func (s *Set) Wait() {
	s.wg.Wait()
}

func (s *Set) Done() {
	s.closer.Do(func() { close(s.queue) })
	s.Wait()
}

func (s *Set) Count() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table := make(map[string]int64)
	for _, err := range s.errors {
		code := GetErrorCode(err)
		if _, ok := table[code]; ok {
			table[code]++
		} else {
			table[code] = 1
		}
	}

	return table
}
