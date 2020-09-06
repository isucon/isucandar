package failure

import (
	"fmt"
	"golang.org/x/xerrors"
)

type Error struct {
	Code
	err   error
	frame xerrors.Frame
}

func NewError(code Code, err error) *Error {
	var wrapped *Error
	if ok := As(err, &wrapped); ok {
		return NewError(code, wrapped.err)
	}

	return &Error{
		Code:  code,
		err:   err,
		frame: xerrors.Caller(1),
	}
}

func (e *Error) Error() string { // implements error
	return e.ErrorCode()
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
		e.frame.Format(p)
	}
	return e.err
}

func Is(err, target error) bool {
	return xerrors.Is(err, target)
}

func As(err error, target interface{}) bool {
	return xerrors.As(err, target)
}
