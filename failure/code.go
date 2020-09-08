package failure

import (
	"golang.org/x/xerrors"
)

type Code interface {
	Error() string
	ErrorCode() string
}

type StringCode string

func (s StringCode) Error() string {
	return string(s)
}

func (s StringCode) ErrorCode() string {
	return string(s)
}

func GetErrorCode(err error) string {
	var code Code
	if ok := As(err, &code); ok {
		return code.ErrorCode()
	} else {
		return UnknownErrorCode.ErrorCode()
	}
}

func GetErrorCodes(err error) []string {
	var code Code
	var wrap xerrors.Wrapper

	unwrapped := false
	codes := []string{}

	for err != nil {
		if ok := As(err, &code); ok {
			codes = append(codes, code.ErrorCode())
		} else if !unwrapped {
			codes = append(codes, UnknownErrorCode.ErrorCode())
		}

		if ok := As(err, &wrap); ok {
			err = wrap.Unwrap()
			unwrapped = true
		} else {
			err = nil
		}
	}

	return codes
}

const (
	UnknownErrorCode   StringCode = "unknown"
	CanceledErrorCode  StringCode = "canceled"
	TimeoutErrorCode   StringCode = "timeout"
	TemporaryErrorCode StringCode = "temporary"
)
