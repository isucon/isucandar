package agent

import (
	"github.com/rosylilly/isucandar/failure"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-time.After(2 * time.Second)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		io.WriteString(w, "Hello, World")
	}))
	defer srv.Close()

	agent, err := NewAgent(WithTimeout(1*time.Second), WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = get(agent, "/")
	if err == nil || failure.GetErrorCode(err) != failure.TimeoutErrorCode.ErrorCode() {
		t.Fatalf("expected timeout error: %+v", err)
	}
}
