package useragent

import (
	"fmt"
	"math/rand"
)

func Platform() string {
	switch rand.Intn(3) {
	case 1:
		return MacOS()
	case 2:
		return Linux()
	default:
		return Windows()
	}
}

func Windows() string {
	return "Windows NT 10.0; Win64; x64"
}

func MacOS() string {
	return fmt.Sprintf("Macintosh; Intel Mac OS X 10.%d", 11+rand.Intn(3))
}

var (
	linuxDistributions = []string{
		"Ubuntu",
		"U",
		"Arch Linux",
	}
)

func Linux() string {
	return fmt.Sprintf("X11; %s; Linux x86_64", linuxDistributions[rand.Intn(len(linuxDistributions))])
}
