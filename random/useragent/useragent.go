package useragent

import (
	"math/rand"
)

func UserAgent() string {
	switch rand.Intn(3) {
	case 1:
		return Chrome()
	case 2:
		return Edge()
	default:
		return Firefox()
	}
}
