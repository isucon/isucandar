package isucandar

import (
	"context"
	"sync"

	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/score"
)

type BenchmarkStep struct {
	errorCode failure.Code
	mu        sync.RWMutex
	result    *BenchmarkResult
	cancel    context.CancelFunc
}

func (b *BenchmarkStep) setErrorCode(code failure.Code) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.errorCode = code
}

func (b *BenchmarkStep) AddError(err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.errorCode != nil {
		b.result.Errors.Add(failure.NewError(b.errorCode, err))
	} else {
		b.result.Errors.Add(err)
	}
}

func (b *BenchmarkStep) AddScore(tag score.ScoreTag) {
	b.result.Score.Add(tag)
}

func (b *BenchmarkStep) Cancel() {
	b.cancel()
}

func (b *BenchmarkStep) Result() *BenchmarkResult {
	return b.result
}

func (b *BenchmarkStep) wait() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.result.Score.Wait()
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		b.result.Errors.Wait()
		wg.Done()
	}()
	wg.Wait()
}
