package agent

import (
	"net/url"
	"time"
)

func WithNoCookie() AgentOption {
	return func(a *Agent) error {
		a.HttpClient.Jar = nil
		return nil
	}
}

func WithNoCache() AgentOption {
	return func(a *Agent) error {
		a.CacheStore = nil
		return nil
	}
}

func WithUserAgent(ua string) AgentOption {
	return func(a *Agent) error {
		a.Name = ua
		return nil
	}
}

func WithBaseURL(base string) AgentOption {
	return func(a *Agent) error {
		var err error
		a.BaseURL, err = url.Parse(base)
		return err
	}
}

func WithTimeout(d time.Duration) AgentOption {
	return func(a *Agent) error {
		a.HttpClient.Timeout = d
		return nil
	}
}
