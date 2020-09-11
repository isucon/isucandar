package isucandar

import (
	"time"
)

type BenchmarkOption func(*Benchmark) error

func WithPrepareTimeout(d time.Duration) BenchmarkOption {
	return func(b *Benchmark) error {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.prepateTimeout = d
		return nil
	}
}

func WithLoadTimeout(d time.Duration) BenchmarkOption {
	return func(b *Benchmark) error {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.loadTimeout = d
		return nil
	}
}
