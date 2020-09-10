package test

import (
	"net/http"
	"testing"
)

func TestIsSuccessfulResponse(t *testing.T) {
	res := &http.Response{}

	expects := map[int]bool{
		200: true,
		201: true,
		204: true,
		299: true,
		300: false,
		303: false,
		304: true,
		305: false,
		404: false,
	}

	for statusCode, ok := range expects {
		res.StatusCode = statusCode
		if IsSuccessfulResponse(res) != ok {
			t.Fatalf("%d: %v / %v", statusCode, IsSuccessfulResponse(res), ok)
		}
	}
}

func TestHasExpectedHeader(t *testing.T) {
	res := &http.Response{
		Header: make(http.Header),
	}

	res.Header.Set("X-Drive", "1")
	res.Header.Add("X-Drive", "2")

	expected := http.Header{
		"X-Drive": []string{"1", "2"},
	}

	if !HasExpectedHeader(res, expected) {
		t.Fatal("header check failed")
	}

	notFound := http.Header{
		"X-Not-Found": []string{"value"},
	}
	if HasExpectedHeader(res, notFound) {
		t.Fatal("header check failed")
	}

	invalidLength := http.Header{
		"X-Drive": []string{"1"},
	}
	if HasExpectedHeader(res, invalidLength) {
		t.Fatal("header check failed")
	}

	invalidValue := http.Header{
		"X-Drive": []string{"1", "3"},
	}
	if HasExpectedHeader(res, invalidValue) {
		t.Fatal("header check failed")
	}
}
