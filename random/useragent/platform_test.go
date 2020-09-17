package useragent

import (
	"testing"
)

func TestPlatform(t *testing.T) {
	for i := 0; i < 100; i++ {
		platform := Platform()
		if platform == "" {
			t.Fatal("Empty platform")
		}
	}
}
