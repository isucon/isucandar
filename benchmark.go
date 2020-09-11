package isucandar

import (
	"context"
	"fmt"
	"github.com/rosylilly/isucandar/failure"
	"github.com/rosylilly/isucandar/parallel"
	"sync"
	"time"
)

var (
	ErrPanic      failure.StringCode = "panic"
	ErrPrepare    failure.StringCode = "prepare"
	ErrLoad       failure.StringCode = "load"
	ErrValidation failure.StringCode = "validation"
)

type BenchmarkStep func(context.Context, *Result) error
type BenchmarkErrorHook func(error, *Result)

type Benchmark struct {
	mu sync.Mutex

	prepareSteps    []BenchmarkStep
	loadSteps       []BenchmarkStep
	validationSteps []BenchmarkStep

	prepateTimeout time.Duration
	loadTimeout    time.Duration
	ignoreCodes    []failure.Code
	errorHooks     []BenchmarkErrorHook
}

func NewBenchmark(opts ...BenchmarkOption) (*Benchmark, error) {
	benchmark := &Benchmark{
		mu:              sync.Mutex{},
		prepareSteps:    []BenchmarkStep{},
		loadSteps:       []BenchmarkStep{},
		validationSteps: []BenchmarkStep{},
		prepateTimeout:  time.Duration(0),
		loadTimeout:     time.Duration(0),
		ignoreCodes:     []failure.Code{},
		errorHooks:      []BenchmarkErrorHook{},
	}

	for _, opt := range opts {
		if err := opt(benchmark); err != nil {
			return nil, err
		}
	}

	return benchmark, nil
}

func (b *Benchmark) Start(parent context.Context) *Result {
	ctx, cancel := context.WithCancel(parent)
	result := newResult(ctx, cancel)
	defer result.Cancel()

	for _, hook := range b.errorHooks {
		func(hook BenchmarkErrorHook) {
			result.Errors.Hook(func(err error) {
				hook(err, result)
			})
		}(hook)
	}

	loadParallel := parallel.NewParallel(-1)
	var (
		loadCtx    context.Context
		loadCancel context.CancelFunc
	)

	for _, prepare := range b.prepareSteps {
		var (
			prepareCtx    context.Context
			prepareCancel context.CancelFunc
		)

		if b.prepateTimeout > 0 {
			prepareCtx, prepareCancel = context.WithTimeout(ctx, b.prepateTimeout)
		} else {
			prepareCtx, prepareCancel = context.WithCancel(ctx)
		}
		defer prepareCancel()

		if err := panicWrapper(func() error { return prepare(prepareCtx, result) }); err != nil {
			for _, ignore := range b.ignoreCodes {
				if failure.IsCode(err, ignore) {
					goto Result
				}
			}
			result.Errors.Add(failure.NewError(ErrPrepare, err))
			goto Result
		}
	}

	result.Errors.Wait()

	if ctx.Err() != nil {
		goto Result
	}

	if b.loadTimeout > 0 {
		loadCtx, loadCancel = context.WithTimeout(ctx, b.loadTimeout)
	} else {
		loadCtx, loadCancel = context.WithCancel(ctx)
	}
	defer loadCancel()

	for _, load := range b.loadSteps {
		func(f BenchmarkStep) {
			loadParallel.Do(loadCtx, func(c context.Context) {
				if err := panicWrapper(func() error { return f(c, result) }); err != nil {
					for _, ignore := range b.ignoreCodes {
						if failure.IsCode(err, ignore) {
							return
						}
					}
					result.Errors.Add(failure.NewError(ErrLoad, err))
				}
			})
		}(load)
	}
	<-loadParallel.Wait()

	result.Errors.Wait()

	if ctx.Err() != nil {
		goto Result
	}

	for _, validation := range b.validationSteps {
		if err := panicWrapper(func() error { return validation(ctx, result) }); err != nil {
			for _, ignore := range b.ignoreCodes {
				if failure.IsCode(err, ignore) {
					goto Result
				}
			}
			result.Errors.Add(failure.NewError(ErrValidation, err))
			goto Result
		}
	}

Result:
	cancel()
	result.wait()

	return result
}

func (b *Benchmark) OnError(f BenchmarkErrorHook) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.errorHooks = append(b.errorHooks, f)
}

func (b *Benchmark) Prepare(f BenchmarkStep) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.prepareSteps = append(b.prepareSteps, f)
}

func (b *Benchmark) Load(f BenchmarkStep) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.loadSteps = append(b.loadSteps, f)
}

func (b *Benchmark) Validation(f BenchmarkStep) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.validationSteps = append(b.validationSteps, f)
}

func (b *Benchmark) IgnoreErrorCode(code failure.Code) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.ignoreCodes = append(b.ignoreCodes, code)
}

func panicWrapper(f func() error) (err error) {
	defer func() {
		re := recover()
		if re == nil {
			return
		}

		if rerr, ok := re.(error); !ok {
			err = failure.NewError(ErrPanic, fmt.Errorf("%v", re))
		} else {
			err = failure.NewError(ErrPanic, rerr)
		}
	}()

	return f()
}
