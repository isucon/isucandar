package failure

import (
	"context"
	"fmt"
	"golang.org/x/xerrors"
	"net"
)

var (
	CaptureCallstackSize = 5
)

type Error struct {
	Code
	err error
	// xerrors は1スタックしかとりあげてくれないので複数取るように
	frames []xerrors.Frame
}

func NewError(code Code, err error) *Error {
	// var wrapped *Error
	// if ok := As(err, &wrapped); ok {
	// 	return NewError(code, wrapped.err)
	// }

	var nerr net.Error
	if ok := As(err, &nerr); ok {
		switch true {
		case nerr.Timeout():
			code = TimeoutErrorCode
		case nerr.Temporary():
			code = TemporaryErrorCode
		default:
		}
	}

	if ok := Is(err, context.Canceled); ok {
		code = CanceledErrorCode
	}

	if ok := Is(err, context.DeadlineExceeded); ok {
		code = TimeoutErrorCode
	}

	frames := make([]xerrors.Frame, 0, CaptureCallstackSize)
	for i := 0; i < CaptureCallstackSize; i++ {
		frame := xerrors.Caller(i + 1)
		frames = append(frames, frame)
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

func (e *Error) getOriginalMessage() string {
	var err *Error
	if As(e.err, &err) {
		return err.getOriginalMessage()
	} else {
		return e.err.Error()
	}
}

func Is(err, target error) bool {
	return xerrors.Is(err, target)
}

func As(err error, target interface{}) bool {
	return xerrors.As(err, target)
}
