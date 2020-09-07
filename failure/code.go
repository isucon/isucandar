package failure

type Code interface {
	ErrorCode() string
}

type StringCode string

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

const (
	UnknownErrorCode   StringCode = "unknown"
	CanceledErrorCode  StringCode = "canceled"
	TimeoutErrorCode   StringCode = "timeout"
	TemporaryErrorCode StringCode = "temporary"
)
