package failure

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
)

func TestErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	set := NewErrors(ctx)
	defer cancel()

	for i := 0; i < 100; i++ {
		set.Add(fmt.Errorf("unknown error"))
	}

	set.Done()

	table := set.Count()
	if table["unknown"] != 100 {
		t.Errorf("missmatch unknown count: %d", table["unknown"])
	}

	errors := set.All()
	if len(errors) != 100 {
		t.Errorf("missmatch errors count: %d", len(errors))
	}

	set.Reset()
	moreErrors := set.All()
	if len(moreErrors) != 0 {
		t.Errorf("missmatch errors count: %d", len(moreErrors))
	}
}

func TestErrorsClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	set := NewErrors(ctx)

	set.Add(fmt.Errorf("test"))
	set.Add(fmt.Errorf("test"))
	set.Add(fmt.Errorf("test"))

	set.Done()

	set.Add(fmt.Errorf("test"))

	table := set.Count()
	if table["unknown"] != 3 {
		t.Fatalf("missmatch unknown count: %d", table["unknown"])
	}

	messages := set.Messages()
	if len(messages["unknown"]) != 3 {
		t.Fatalf("missmatch unknown message count: %d", len(messages["unknown"]))
	}
}

func TestErrorsHook(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	set := NewErrors(ctx)
	defer cancel()

	n := int32(0)
	cnt := &n
	set.Hook(func(err error) {
		atomic.AddInt32(cnt, 1)
	})

	for i := 0; i < 10; i++ {
		set.Add(fmt.Errorf("unknown error"))
	}

	set.Done()

	table := set.Count()
	if table["unknown"] != 10 {
		t.Errorf("missmatch unknown count: %d", table["unknown"])
	}

	set.Wait()

	if atomic.LoadInt32(cnt) != 10 {
		t.Errorf("missmatch unknown hook count: %d", atomic.LoadInt32(cnt))
	}
}

func BenchmarkErrorsAdd(b *testing.B) {
	err := fmt.Errorf("test")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	set := NewErrors(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Add(err)
	}
	set.Done()
	b.StopTimer()
}
