package failure

import (
	"fmt"
	"strings"
	"testing"
)

const (
	errApplication StringCode = "application"
	errTemporary   StringCode = "temporary"
)

func TestError(t *testing.T) {
	berr := fmt.Errorf("Test")
	aerr := NewError(errApplication, berr)

	if m := fmt.Sprint(aerr); m != "application: Test" {
		t.Fatalf("missmatch: %s", m)
	}

	if m := fmt.Sprintf("%+v", aerr); strings.HasPrefix(m, "application: Test") {
		t.Fatalf("missmatch: %s", m)
	}

	if !Is(aerr, berr) {
		t.Fatalf("check invalid")
	}

	terr := NewError(errTemporary, aerr)

	if m := fmt.Sprint(terr); m != "temporary: application: Test" {
		t.Fatalf("missmatch: %s", m)
	}

	if !Is(terr, berr) {
		t.Fatalf("check invalid")
	}

	t.Logf("\n%+v", aerr)
	t.Logf("\n%+v", terr)
}

func TestErrorFrames(t *testing.T) {
	berr := fmt.Errorf("frames")

	var f func(int) error
	f = func(n int) error {
		if n > 0 {
			return f(n - 1)
		} else {
			return NewError(errApplication, berr)
		}
	}
	aerr := f(3)

	details := fmt.Sprintf("%+v", aerr)
	dLines := strings.Split(details, "\n")

	// callstack * 2 + 2 messages
	eLineCount := 2 + CaptureCallstackSize*2
	if len(dLines) != eLineCount {
		t.Fatalf("expected %d but got %d", eLineCount, len(dLines))
	}

	t.Logf("%+v", aerr)
}
