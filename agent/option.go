package agent

import (
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
		a.BaseURL = base
		return nil
	}
}

func WithTimeout(d time.Duration) AgentOption {
	return func(a *Agent) error {
		a.RequestTimeout = d
		return nil
	}
}
