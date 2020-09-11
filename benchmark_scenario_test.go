package isucandar

import (
	"context"
	"sync/atomic"
	"testing"
)

type exampleScenario struct {
	prepare    uint32
	load       uint32
	validation uint32
}

func (e *exampleScenario) Prepare(_ context.Context, _ *BenchmarkStep) error {
	atomic.StoreUint32(&e.prepare, 1)
	return nil
}

func (e *exampleScenario) Load(_ context.Context, _ *BenchmarkStep) error {
	atomic.StoreUint32(&e.load, 1)
	return nil
}

func (e *exampleScenario) Validation(_ context.Context, _ *BenchmarkStep) error {
	atomic.StoreUint32(&e.validation, 1)
	return nil
}

func TestBenchmarkAddScenario(t *testing.T) {
	benchmark, err := NewBenchmark()
	if err != nil {
		t.Fatal(err)
	}

	e := &exampleScenario{
		prepare:    0,
		load:       0,
		validation: 0,
	}

	benchmark.AddScenario(e)

	result := benchmark.Start(context.Background())

	if len(result.Errors.All()) > 0 {
		t.Fatal(result.Errors.All())
	}

	if e.prepare != 1 || e.load != 1 || e.validation != 1 {
		t.Fatal(e)
	}
}

func TestBenchmarkAddScenarioPanic(t *testing.T) {
	benchmark, err := NewBenchmark()
	if err != nil {
		t.Fatal(err)
	}

	var rerr interface{}
	func() {
		defer func() {
			rerr = recover()
		}()
		benchmark.AddScenario(nil)
	}()

	if rerr == nil {
		t.Fatal("Do not register invalid scenario")
	}
}
