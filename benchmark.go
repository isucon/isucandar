package isucandar

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/parallel"
)

var (
	ErrPanic      failure.StringCode = "panic"
	ErrPrepare    failure.StringCode = "prepare"
	ErrLoad       failure.StringCode = "load"
	ErrValidation failure.StringCode = "validation"
)

type BenchmarkStepFunc func(context.Context, *BenchmarkStep) error
type BenchmarkErrorHook func(error, *BenchmarkStep)

type Benchmark struct {
	mu sync.Mutex

	prepareSteps    []BenchmarkStepFunc
	loadSteps       []BenchmarkStepFunc
	validationSteps []BenchmarkStepFunc

	panicRecover   bool
	prepareTimeout time.Duration
	loadTimeout    time.Duration
	ignoreCodes    []failure.Code
	errorHooks     []BenchmarkErrorHook
}

func NewBenchmark(opts ...BenchmarkOption) (*Benchmark, error) {
	benchmark := &Benchmark{
		mu:              sync.Mutex{},
		prepareSteps:    []BenchmarkStepFunc{},
		loadSteps:       []BenchmarkStepFunc{},
		validationSteps: []BenchmarkStepFunc{},
		panicRecover:    true,
		prepareTimeout:  time.Duration(0),
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

func (b *Benchmark) Start(parent context.Context) *BenchmarkResult {
	ctx, cancel := context.WithCancel(parent)
	result := newBenchmarkResult(ctx)
	defer cancel()

	step := &BenchmarkStep{
		mu:     sync.RWMutex{},
		result: result,
		cancel: cancel,
	}

	for _, hook := range b.errorHooks {
		func(hook BenchmarkErrorHook) {
			result.Errors.Hook(func(err error) {
				hook(err, step)
			})
		}(hook)
	}

	var (
		loadParallel *parallel.Parallel
		loadCtx      context.Context
		loadCancel   context.CancelFunc
	)

	step.setErrorCode(ErrPrepare)
	for _, prepare := range b.prepareSteps {
		var (
			prepareCtx    context.Context
			prepareCancel context.CancelFunc
		)

		if b.prepareTimeout > 0 {
			prepareCtx, prepareCancel = context.WithTimeout(ctx, b.prepareTimeout)
		} else {
			prepareCtx, prepareCancel = context.WithCancel(ctx)
		}
		defer prepareCancel()

		if err := panicWrapper(b.panicRecover, func() error { return prepare(prepareCtx, step) }); err != nil {
			for _, ignore := range b.ignoreCodes {
				if failure.IsCode(err, ignore) {
					goto Result
				}
			}
			step.AddError(err)
			goto Result
		}
	}

	result.Errors.Wait()

	if ctx.Err() != nil {
		goto Result
	}

	step.setErrorCode(ErrLoad)
	if b.loadTimeout > 0 {
		loadCtx, loadCancel = context.WithTimeout(ctx, b.loadTimeout)
	} else {
		loadCtx, loadCancel = context.WithCancel(ctx)
	}
	loadParallel = parallel.NewParallel(loadCtx, -1)

	for _, load := range b.loadSteps {
		func(f BenchmarkStepFunc) {
			loadParallel.Do(func(c context.Context) {
				if err := panicWrapper(b.panicRecover, func() error { return f(c, step) }); err != nil {
					for _, ignore := range b.ignoreCodes {
						if failure.IsCode(err, ignore) {
							return
						}
					}
					step.AddError(err)
				}
			})
		}(load)
	}
	loadParallel.Wait()
	loadCancel()

	result.Errors.Wait()

	if ctx.Err() != nil {
		goto Result
	}

	step.setErrorCode(ErrValidation)
	for _, validation := range b.validationSteps {
		if err := panicWrapper(b.panicRecover, func() error { return validation(ctx, step) }); err != nil {
			for _, ignore := range b.ignoreCodes {
				if failure.IsCode(err, ignore) {
					goto Result
				}
			}
			step.AddError(err)
			goto Result
		}
	}

Result:
	cancel()
	step.wait()
	step.setErrorCode(nil)

	return result
}

func (b *Benchmark) OnError(f BenchmarkErrorHook) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.errorHooks = append(b.errorHooks, f)
}

func (b *Benchmark) Prepare(f BenchmarkStepFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.prepareSteps = append(b.prepareSteps, f)
}

func (b *Benchmark) Load(f BenchmarkStepFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.loadSteps = append(b.loadSteps, f)
}

func (b *Benchmark) Validation(f BenchmarkStepFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.validationSteps = append(b.validationSteps, f)
}

func (b *Benchmark) IgnoreErrorCode(code failure.Code) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.ignoreCodes = append(b.ignoreCodes, code)
}

func panicWrapper(on bool, f func() error) (err error) {
	if !on {
		return f()
	}

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
