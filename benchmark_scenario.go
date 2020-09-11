package isucandar

import (
	"context"
	"errors"
)

var (
	ErrInvalidScenario = errors.New("Invalid scenario interface")
)

type PrepareScenario interface {
	Prepare(context.Context, *BenchmarkStep) error
}

type LoadScenario interface {
	Load(context.Context, *BenchmarkStep) error
}

type ValidationScenario interface {
	Validation(context.Context, *BenchmarkStep) error
}

func (b *Benchmark) AddScenario(scenario interface{}) {
	match := false
	if p, ok := scenario.(PrepareScenario); ok {
		b.Prepare(p.Prepare)
		match = true
	}

	if l, ok := scenario.(LoadScenario); ok {
		b.Load(l.Load)
		match = true
	}

	if v, ok := scenario.(ValidationScenario); ok {
		b.Validation(v.Validation)
		match = true
	}

	if !match {
		panic(ErrInvalidScenario)
	}
}
