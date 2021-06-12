package failure

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	errApplication StringCode = "application"
	errTemporary   StringCode = "temporary"
	errTest        StringCode = "test"
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

	if GetErrorCode(aerr) != "application" {
		t.Fatalf("Error code is invalid: %s", GetErrorCode(aerr))
	}

	terr := NewError(errTemporary, aerr)

	if m := fmt.Sprint(terr); m != "temporary: application: Test" {
		t.Fatalf("missmatch: %s", m)
	}

	if !Is(terr, berr) {
		t.Fatalf("check invalid")
	}

	if GetErrorCode(terr) != "temporary" {
		t.Fatalf("Error code is invalid: %s", GetErrorCode(terr))
	}

	gotCodes := GetErrorCodes(terr)
	expectCodes := []string{"temporary", "application"}
	if !reflect.DeepEqual(gotCodes, expectCodes) {
		t.Fatalf("Error codes is invalid:\n  %v\n  %v", gotCodes, expectCodes)
	}
}

type fakeNetError struct {
	timeout   bool
	temporary bool
}

func (f fakeNetError) Error() string {
	return "fake"
}

func (f fakeNetError) Timeout() bool {
	return f.timeout
}

func (f fakeNetError) Temporary() bool {
	return f.temporary
}

func TestErrorWrap(t *testing.T) {
	rctx := context.TODO()

	ctx, cancel := context.WithCancel(rctx)
	cancel()

	canceledError := NewError(errApplication, ctx.Err())
	if GetErrorCode(canceledError) != "application" {
		t.Fatalf("%s", GetErrorCode(canceledError))
	}
	codes := GetErrorCodes(canceledError)
	expectCodes := []string{"application", CanceledErrorCode.ErrorCode()}
	if !reflect.DeepEqual(codes, expectCodes) {
		t.Fatalf("Error codes is invalid:\n  %v\n  %v", codes, expectCodes)
	}

	ctx, cancel = context.WithTimeout(rctx, -1*time.Second)
	defer cancel()

	timeoutError := NewError(errApplication, ctx.Err())
	if GetErrorCode(timeoutError) != "application" {
		t.Fatalf("%s", GetErrorCode(timeoutError))
	}
	codes = GetErrorCodes(timeoutError)
	expectCodes = []string{"application", TimeoutErrorCode.ErrorCode()}
	if !reflect.DeepEqual(codes, expectCodes) {
		t.Fatalf("Error codes is invalid:\n  %v\n  %v", codes, expectCodes)
	}

	ferr := fakeNetError{timeout: false, temporary: true}
	temporaryError := NewError(errApplication, ferr)
	if GetErrorCode(temporaryError) != "application" {
		t.Fatalf("%s", GetErrorCode(temporaryError))
	}
	codes = GetErrorCodes(temporaryError)
	expectCodes = []string{"application", TemporaryErrorCode.ErrorCode()}
	if !reflect.DeepEqual(codes, expectCodes) {
		t.Fatalf("Error codes is invalid:\n  %v\n  %v", codes, expectCodes)
	}

	err := NewError(errTest, fmt.Errorf("error"))
	nilError := NewError(errTest, err)
	codes = GetErrorCodes(nilError)
	expectCodes = []string{"test"}
	if !reflect.DeepEqual(codes, expectCodes) {
		t.Fatalf("Error codes is invalid:\n  %v\n  %v", codes, expectCodes)
	}
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
	eLineCount := 2 + CaptureBacktraceSize*2
	if len(dLines) != eLineCount {
		t.Fatalf("expected %d but got %d", eLineCount, len(dLines))
	}
}

func TestIsCode(t *testing.T) {
	err := NewError(errApplication, NewError(errTemporary, fmt.Errorf("foo")))

	if !IsCode(err, errApplication) {
		t.Fatal(err)
	}

	if !IsCode(err, errTemporary) {
		t.Fatal(err)
	}

	if IsCode(err, UnknownErrorCode) {
		t.Fatal(err)
	}
}
