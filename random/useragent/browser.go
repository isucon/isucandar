package useragent

import (
	"fmt"
	"math/rand"
)

var (
	chromeVersions = []string{
		"60.0.3112.113",
		"63.0.3239.132",
		"67.0.3396.99",
		"69.0.3497.100",
		"72.0.3626.121",
		"74.0.3729.169",
		"79.0.3945.88",
		"80.0.3987.163",
		"81.0.4044.138",
		"83.0.4103.116",
		"84.0.4147.135",
		"85.0.4183.102",
	}
)

func Chrome() string {
	return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", Platform(), chromeVersions[rand.Intn(len(chromeVersions))])
}

var (
	edgeVersions = []string{
		"79.0.522.52",
		"80.0.522.52",
		"81.0.522.52",
		"82.0.522.52",
		"83.0.522.52",
		"85.0.564.44",
	}
)

func Edge() string {
	return fmt.Sprintf("%s Edg/%s", Chrome(), edgeVersions[rand.Intn(len(edgeVersions))])
}

func Firefox() string {
	return fmt.Sprintf("Mozilla/5.0 (%s) Gecko/20100101 Firefox/%d.0", Platform(), 70+rand.Intn(10))
}
