package failure

import (
	"fmt"
	"strings"
	"testing"
)

func TestBacktraceCleaner(t *testing.T) {
	cleaner := &backtraceCleaner{}
	defaultCleaner := BacktraceCleaner
	defaultCaptureCallstackSize := CaptureCallstackSize
	BacktraceCleaner = cleaner
	CaptureCallstackSize = 100
	defer func() {
		BacktraceCleaner = defaultCleaner
		CaptureCallstackSize = defaultCaptureCallstackSize
	}()

	cleaner.Add(SkipGOROOT)
	cleaner.Add(func(b Backtrace) bool {
		return strings.HasSuffix(b.Function, "TestBacktraceCleaner")
	})

	var f func(int) error
	f = func(n int) error {
		if n > 0 {
			return f(n - 1)
		}
		return NewError(UnknownErrorCode, fmt.Errorf("invalid"))
	}

	err := f(0)

	details := fmt.Sprintf("%+v", err)
	dLines := strings.Split(details, "\n")

	// TestBacktraceCleaner.func3: not match
	// TestBacktraceCleaner: match with Name
	// testing.tRunner: match with GOROOT
	expectLines := ((3 - 2) * 2) + 2
	if len(dLines) != expectLines {
		t.Logf("\n%+v", err)
		t.Fatalf("missmatch call stack size: %d / %d", len(dLines), expectLines)
	}
}
