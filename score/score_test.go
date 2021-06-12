package score

import (
	"context"
	"testing"
)

func TestScoreWithDone(t *testing.T) {
	score := NewScore(context.Background())
	score.Set("foo", 2)
	score.Set("bar", 1)

	for i := 0; i < 1000; i++ {
		score.Add("foo")
		score.Add("bar")
		score.Add("baz")
	}

	score.Done()

	if score.Total() != 3000 {
		t.Fatalf("Expected 3000 but got %d", score.Total())
	}

	score.Reset()
	if score.Total() != 0 {
		t.Fatalf("Expected 0 but got %d", score.Total())
	}
}

func TestScoreWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	score := NewScore(ctx)
	score.Set("foo", 2)

	for i := 0; i < 1000; i++ {
		score.Add("foo")
		score.Add("bar")
	}

	cancel()

	score.Done()

	score.Add("d")
}

func TestScoreBreakdown(t *testing.T) {
	score := NewScore(context.Background())

	score.Add("a")
	score.Add("b")
	score.Add("c")

	score.Done()

	breakdown := score.Breakdown()
	if c, ok := breakdown["a"]; !ok || c != int64(1) {
		t.Fatalf("Add failed of a: %d", c)
	}
	if c, ok := breakdown["b"]; !ok || c != int64(1) {
		t.Fatalf("Add failed of b: %d", c)
	}
	if c, ok := breakdown["c"]; !ok || c != int64(1) {
		t.Fatalf("Add failed of c: %d", c)
	}
}

func BenchmarkScoreCollection(b *testing.B) {
	score := NewScore(context.TODO())
	for i := 0; i < b.N; i++ {
		score.Add("test")
		score.Sum()
	}
	score.Done()
}
