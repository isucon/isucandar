package failure

import (
	"context"
	"fmt"
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
		t.Errorf("missmatch unknown count: %d", table["unknown"])
	}

	messages := set.Messages()
	t.Logf("%+v", messages)
}
