package failure

import (
	"errors"
	"testing"
)

type errorWithCode struct {
	code    string
	message string
}

func (e *errorWithCode) Error() string {
	return e.message
}

func (e *errorWithCode) ErrorCode() string {
	return e.code
}

func TestErrorCode(t *testing.T) {
	err := errors.New("test")
	if code := GetErrorCode(err); code != "unknown" {
		t.Fatalf("expected unknown, got %s", code)
	}

	err = &errorWithCode{
		code:    "test",
		message: "Hello",
	}
	if code := GetErrorCode(err); code != "test" {
		t.Fatalf("expected test, got %s", code)
	}
}
