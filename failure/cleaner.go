package failure

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/xerrors"
)

var (
	BacktraceCleaner = &backtraceCleaner{}
)

type Backtrace struct {
	Function string
	File     string
	LineNo   int
}

func (b *Backtrace) String() string {
	return fmt.Sprintf("%s\n  %s:%d", b.Function, b.File, b.LineNo)
}

type backtraceCleaner struct {
	matcher func(Backtrace) bool
}

func (bc *backtraceCleaner) match(frame xerrors.Frame) bool {
	if bc.matcher == nil {
		return false
	}

	c := &frameConvertor{
		Function: "",
		File:     "",
		LineNo:   -1,
	}
	frame.Format(c)

	b := c.Backtrace()
	return bc.matcher(b)
}

func (bc *backtraceCleaner) Add(matcher func(Backtrace) bool) {
	oldMatcher := bc.matcher
	m := func(b Backtrace) bool {
		if oldMatcher == nil {
			return matcher(b)
		}
		return matcher(b) || oldMatcher(b)
	}
	bc.matcher = m
}

type frameConvertor struct {
	Function string
	File     string
	LineNo   int
}

func (f *frameConvertor) Detail() bool {
	// frame の内容を取りたいので常に true
	return true
}

func (f *frameConvertor) Print(args ...interface{}) {}

func (f *frameConvertor) Printf(format string, args ...interface{}) {
	switch format {
	case "%s\n    ": // function name formatter
		f.Function = fmt.Sprintf("%s", args[0])
	case "%s:%d\n": // file name formatter
		f.File = fmt.Sprintf("%s", args[0])
		f.LineNo, _ = strconv.Atoi(fmt.Sprintf("%d", args[1]))
	}
}

func (f *frameConvertor) Backtrace() Backtrace {
	return Backtrace{
		Function: f.Function,
		File:     f.File,
		LineNo:   f.LineNo,
	}
}

func SkipGOROOT(b Backtrace) bool {
	return strings.HasPrefix(b.File, runtime.GOROOT())
}
