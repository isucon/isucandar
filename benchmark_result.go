package isucandar

import (
	"context"
	"github.com/rosylilly/isucandar/failure"
	"github.com/rosylilly/isucandar/score"
)

type BenchmarkResult struct {
	Score  *score.Score
	Errors *failure.Errors
}

func newBenchmarkResult(ctx context.Context) *BenchmarkResult {
	return &BenchmarkResult{
		Score:  score.NewScore(ctx),
		Errors: failure.NewErrors(ctx),
	}
}
