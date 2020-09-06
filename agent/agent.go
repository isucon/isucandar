package agent

import (
	"net/http/cookiejar"
)

type Agent struct {
	Cookies *cookiejar.Jar
}

func NewAgent() (*Agent, error) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		return nil, err
	}

	agent := &Agent{
		Cookies: jar,
	}

	return agent, nil
}
