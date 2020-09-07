package agent

import (
	"testing"
)

func TestNoCookie(t *testing.T) {
	agent, err := NewAgent(WithNoCookie())
	if err != nil {
		t.Fatal(err)
	}

	if agent.HttpClient.Jar != nil {
		t.Fatal("Not removed cookie jar")
	}
}

func TestNoCache(t *testing.T) {
	agent, err := NewAgent(WithNoCache())
	if err != nil {
		t.Fatal(err)
	}

	if agent.CacheStore != nil {
		t.Fatal("Not removed cache store")
	}
}

func TestUserAgent(t *testing.T) {
	agent, err := NewAgent(WithUserAgent("Hello"))
	if err != nil {
		t.Fatal(err)
	}

	if agent.Name != "Hello" {
		t.Fatalf("missmatch ua: %s", agent.Name)
	}
}

func TestBaseURL(t *testing.T) {
	agent, err := NewAgent(WithBaseURL("http://base.example.com"))
	if err != nil {
		t.Fatal(err)
	}

	if agent.BaseURL != "http://base.example.com" {
		t.Fatalf("missmatch base URL: %s", agent.BaseURL)
	}
}
