package failure

import (
	"context"
	"errors"
	"fmt"
	"net"

	"golang.org/x/xerrors"
)

var (
	CaptureBacktraceSize = 5
)

type Error struct {
	Code
	err error
	// xerrors は1スタックしかとりあげてくれないので複数取るように
	frames []xerrors.Frame
}

func NewError(code Code, err error) error {
	// Skip already wrapped
	if IsCode(err, code) {
		return err
	}

	var nerr net.Error
	if As(err, &nerr) {
		switch true {
		case nerr.Timeout():
			err = newError(TimeoutErrorCode, err)
		case nerr.Temporary():
			err = newError(TemporaryErrorCode, err)
		}
	} else if Is(err, context.Canceled) {
		err = newError(CanceledErrorCode, err)
	}

	return newError(code, err)
}

func newError(code Code, err error) *Error {
	frames := make([]xerrors.Frame, 0, CaptureBacktraceSize)
	skip := 2
	for i := 0; i < CaptureBacktraceSize; i++ {
		frame := xerrors.Caller(i + skip)
		if BacktraceCleaner.match(frame) {
			i--
			skip++
		} else {
			frames = append(frames, frame)
		}
	}

	return &Error{
		Code:   code,
		err:    err,
		frames: frames,
	}
}

func (e *Error) Unwrap() error { // implments xerrors.Wrapper
	return e.err
}

func (e *Error) Format(f fmt.State, c rune) { // implements fmt.Formatter
	xerrors.FormatError(e, f, c)
}

func (e *Error) FormatError(p xerrors.Printer) error { // implements xerrors.Formatter
	p.Print(e.Error())
	if p.Detail() {
		for _, frame := range e.frames {
			frame.Format(p)
		}
	}
	return e.err
}

func Is(err, target error) bool {
	return err == target || xerrors.Is(err, target) || errors.Is(err, target)
}

func As(err error, target interface{}) bool {
	return xerrors.As(err, target)
}

func IsCode(err error, code Code) bool {
	for _, c := range GetErrorCodes(err) {
		if c == code.ErrorCode() {
			return true
		}
	}

	return false
}
