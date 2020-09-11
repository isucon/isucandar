package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	"github.com/julienschmidt/httprouter"
)

func newHTTPServer() *httptest.Server {
	r := httprouter.New()
	r.GET("/dump", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Printf("%s", dump)
		w.WriteHeader(http.StatusNoContent)
	})

	r.GET("/not_found", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(404)
	})
	r.GET("/301redirect", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.Redirect(w, r, "/301", http.StatusMovedPermanently)
	})
	r.GET("/302redirect", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.Redirect(w, r, "/302", http.StatusFound)
	})
	r.GET("/304redirect", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.Redirect(w, r, "/304", http.StatusNotModified)
	})
	r.GET("/307redirect", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.Redirect(w, r, "/307", http.StatusTemporaryRedirect)
	})
	r.GET("/308redirect", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.Redirect(w, r, "/308", http.StatusPermanentRedirect)
	})

	return httptest.NewServer(r)
}

func TestAgent(t *testing.T) {
	errOpt := func(_ *Agent) error {
		return errors.New("invalid")
	}

	agent, err := NewAgent(errOpt)
	if err == nil || agent != nil {
		t.Fatal("error not occured")
	}
}

func TestAgentClearCookie(t *testing.T) {
	agent, err := NewAgent(WithBaseURL("http://example.com/"))
	if err != nil {
		t.Fatal(err)
	}

	agent.HttpClient.Jar.SetCookies(agent.BaseURL, []*http.Cookie{
		&http.Cookie{},
	})
	if len(agent.HttpClient.Jar.Cookies(agent.BaseURL)) != 1 {
		t.Fatal("Set cookie failed")
	}
	agent.ClearCookie()
	if len(agent.HttpClient.Jar.Cookies(agent.BaseURL)) != 0 {
		t.Fatal("Clear  cookie failed")
	}
}

func TestAgentNewRequest(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	_, err = agent.NewRequest(http.MethodGet, "://invalid-uri", nil)
	if err == nil {
		t.Fatal("Not reached url parse error")
	}

	_, err = agent.NewRequest("bad method", "/", nil)
	if err == nil {
		t.Fatalf("Not reached method name error")
	}
}

func TestAgentRequest(t *testing.T) {
	srv := newHTTPServer()
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("%+v", err)
	}

	req, err := agent.GET("/302redirect")
	if err != nil {
		t.Fatalf("%+v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res, err := agent.Do(ctx, req)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if res.StatusCode != 302 {
		t.Fatalf("%#v", res)
	}
}

func TestAgentMethods(t *testing.T) {
	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	r, _ := agent.GET("/")
	if r.Method != http.MethodGet {
		t.Fatalf("Method missmatch: %s", r.Method)
	}
	r, _ = agent.POST("/", nil)
	if r.Method != http.MethodPost {
		t.Fatalf("Method missmatch: %s", r.Method)
	}
	r, _ = agent.PUT("/", nil)
	if r.Method != http.MethodPut {
		t.Fatalf("Method missmatch: %s", r.Method)
	}
	r, _ = agent.PATCH("/", nil)
	if r.Method != http.MethodPatch {
		t.Fatalf("Method missmatch: %s", r.Method)
	}
	r, _ = agent.DELETE("/", nil)
	if r.Method != http.MethodDelete {
		t.Fatalf("Method missmatch: %s", r.Method)
	}
}
