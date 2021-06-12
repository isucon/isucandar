package failure

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/xerrors"
)

func TestBacktraceCleaner(t *testing.T) {
	cleaner := &backtraceCleaner{}
	defaultCleaner := BacktraceCleaner
	defaultCaptureBacktraceSize := CaptureBacktraceSize
	BacktraceCleaner = cleaner
	CaptureBacktraceSize = 100
	defer func() {
		BacktraceCleaner = defaultCleaner
		CaptureBacktraceSize = defaultCaptureBacktraceSize
	}()

	cleaner.Add(SkipGOROOT)
	cleaner.Add(func(b Backtrace) bool {
		return strings.HasSuffix(b.Function, "TestBacktraceCleaner")
	})

	var code StringCode = "cleaner"
	var f func(int) error
	f = func(n int) error {
		if n > 0 {
			return f(n - 1)
		}
		return NewError(code, fmt.Errorf("invalid"))
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

func TestFrameConvertor(t *testing.T) {
	convertor := &frameConvertor{}

	frame := xerrors.Caller(0)

	// No op
	convertor.Print(frame)

	frame.Format(convertor)
	backtrace := convertor.Backtrace()

	if !strings.HasSuffix(backtrace.Function, "failure.TestFrameConvertor") {
		t.Fatalf("Not match function: %s", backtrace.String())
	}
	t.Logf("%s", backtrace.String())
}
