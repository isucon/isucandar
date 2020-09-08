package failure

import (
	"context"
	"fmt"
	"sync"
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
}

func TestErrorsClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	set := NewErrors(ctx)

	set.Add(fmt.Errorf("test"))
	set.Add(fmt.Errorf("test"))
	set.Add(fmt.Errorf("test"))

	cancel()
	set.Wait()

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

	cnt := 0
	mu := sync.Mutex{}
	set.Hook(func(err error) {
		mu.Lock()
		defer mu.Unlock()

		if GetErrorCode(err) == "unknown" {
			cnt++
		}
	})

	for i := 0; i < 10; i++ {
		set.Add(fmt.Errorf("unknown error"))
	}

	set.Done()

	table := set.Count()
	if table["unknown"] != 10 {
		t.Errorf("missmatch unknown count: %d", table["unknown"])
	}

	mu.Lock()
	if cnt != 10 {
		t.Errorf("missmatch unknown hook count: %d", cnt)
	}
	mu.Unlock()
}
