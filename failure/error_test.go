package failure

import (
	"fmt"
	"strings"
	"testing"
)

const (
	errApplication StringCode = "application"
	errTemporary   StringCode = "temporary"
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

	terr := NewError(errTemporary, aerr)

	if m := fmt.Sprint(terr); m != "temporary: Test" {
		t.Fatalf("missmatch: %s", m)
	}

	if !Is(terr, berr) {
		t.Fatalf("check invalid")
	}
}
