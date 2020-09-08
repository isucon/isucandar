package agent

import (
	"context"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"path/filepath"
	"runtime"
	"testing"
)

func newHTTPServer() *httptest.Server {
	_, file, _, _ := runtime.Caller(1)
	fs := http.FileServer(http.Dir(filepath.Join(filepath.Dir(file), "..", "example")))

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

	r.NotFound = fs

	return httptest.NewServer(r)
}

func TestAgentRequest(t *testing.T) {
	srv := newHTTPServer()
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("%+v", err)
	}

	req, err := agent.Get("/302redirect")
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

	r, _ := agent.Get("/")
	if r.Method != http.MethodGet {
		t.Fatalf("Method missmatch: %s", r.Method)
	}
	r, _ = agent.Post("/", nil)
	if r.Method != http.MethodPost {
		t.Fatalf("Method missmatch: %s", r.Method)
	}
	r, _ = agent.Put("/", nil)
	if r.Method != http.MethodPut {
		t.Fatalf("Method missmatch: %s", r.Method)
	}
	r, _ = agent.Patch("/", nil)
	if r.Method != http.MethodPatch {
		t.Fatalf("Method missmatch: %s", r.Method)
	}
	r, _ = agent.Delete("/", nil)
	if r.Method != http.MethodDelete {
		t.Fatalf("Method missmatch: %s", r.Method)
	}
}

func TestCacheControl(t *testing.T) {
	srv := newHTTPServer()
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	req, err := agent.Get("/dot.gif")
	if err != nil {
		t.Fatal(err)
	}
	res, err := agent.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// Second request

	req, err = agent.Get("/dot.gif")
	if err != nil {
		t.Fatal(err)
	}
	res, err = agent.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 304 {
		t.Fatalf("missmatch status code: %d", res.StatusCode)
	}
}
