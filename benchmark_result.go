package isucandar

import (
	"context"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/score"
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
