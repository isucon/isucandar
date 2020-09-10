package test

import (
	"net/http"
)

func IsSuccessfulResponse(r *http.Response) bool {
	return (r.StatusCode >= 200 && r.StatusCode <= 299) || r.StatusCode == 304
}

func HasExpectedHeader(r *http.Response, header http.Header) bool {
	for key, values := range header {
		actual := r.Header.Values(key)
		if len(actual) != len(values) {
			return false
		}

		for i, v := range values {
			if v != actual[i] {
				return false
			}
		}
	}

	return true
}
