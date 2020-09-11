package failure

import (
	"context"
	"sync"
	"sync/atomic"
)

type ErrorsHook func(error)

type Errors struct {
	mu     sync.RWMutex
	count  int32
	closer sync.Once
	errors []error
	queue  chan error
	hook   ErrorsHook
}

func NewErrors(ctx context.Context) *Errors {
	set := &Errors{
		mu:     sync.RWMutex{},
		count:  int32(0),
		closer: sync.Once{},
		errors: make([]error, 0, 0),
		queue:  make(chan error),
	}

	set.hook = func(err error) {
		atomic.AddInt32(&set.count, -1)
	}

	go set.collect(ctx)

	return set
}

func (s *Errors) collect(ctx context.Context) {
	atomic.AddInt32(&s.count, 1)
	defer atomic.AddInt32(&s.count, -1)

	go func() {
		<-ctx.Done()
		s.Close()
	}()

	for err := range s.queue {
		s.mu.Lock()
		s.errors = append(s.errors, err)
		s.mu.Unlock()

		go s.hook(err)
	}
}

func (s *Errors) Add(err error) {
	defer func() { recover() }()
	s.queue <- err
	atomic.AddInt32(&s.count, 1)
}

func (s *Errors) Hook(hook ErrorsHook) {
	oldHook := s.hook
	s.hook = func(err error) {
		defer oldHook(err)

		hook(err)
	}
}

func (s *Errors) Wait() {
	for atomic.LoadInt32(&s.count) > 0 {
	}
}

func (s *Errors) Close() {
	s.closer.Do(func() { close(s.queue) })
}

func (s *Errors) Done() {
	s.Close()
	s.Wait()
}

func (s *Errors) Messages() map[string][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table := make(map[string][]string)
	for _, err := range s.errors {
		code := GetErrorCode(err)
		if _, ok := table[code]; ok {
			table[code] = append(table[code], err.Error())
		} else {
			table[code] = []string{err.Error()}
		}
	}

	return table
}

func (s *Errors) Count() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table := make(map[string]int64)
	for _, err := range s.errors {
		codes := GetErrorCodes(err)
		for _, code := range codes {
			if _, ok := table[code]; ok {
				table[code]++
			} else {
				table[code] = 1
			}
		}
	}

	return table
}

func (s *Errors) All() []error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	errors := make([]error, len(s.errors))
	copy(errors, s.errors)

	return errors
}
