package useragent

import (
	"testing"
)

func TestUserAgent(t *testing.T) {
	for i := 0; i < 100; i++ {
		ua := UserAgent()
		if ua == "" {
			t.Fatal("Empty platform")
		}
	}
}
