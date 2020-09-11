package isucandar

import (
	"context"
	"github.com/rosylilly/isucandar/failure"
	"github.com/rosylilly/isucandar/score"
	"sync"
)

type Result struct {
	Score  *score.Score
	Errors *failure.Errors
	cancel context.CancelFunc
}

func newResult(ctx context.Context, cancel context.CancelFunc) *Result {
	return &Result{
		Score:  score.NewScore(ctx),
		Errors: failure.NewErrors(ctx),
		cancel: cancel,
	}
}

func (r *Result) Cancel() {
	r.cancel()
}

func (r *Result) wait() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		r.Score.Wait()
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		r.Errors.Wait()
		wg.Done()
	}()
	wg.Wait()
}
