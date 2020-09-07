package agent

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/andybalholm/brotli"
	"github.com/julienschmidt/httprouter"
	"io"
	"io/ioutil"
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
	r.GET("/br", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "br")
		w.WriteHeader(200)
		bw := brotli.NewWriter(w)
		defer bw.Close()
		io.WriteString(bw, "test it")
	})
	r.GET("/gzip", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		gw := gzip.NewWriter(w)
		defer gw.Close()

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(200)
		io.WriteString(gw, "test it")
	})
	r.GET("/deflate", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		fw, err := flate.NewWriter(w, 9)
		if err != nil {
			io.WriteString(w, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer fw.Close()

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "deflate")
		w.WriteHeader(200)
		io.WriteString(fw, "test it")
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

	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	agent.BaseURL = srv.URL

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

	r, _ := agent.Get("/dump")
	agent.Do(ctx, r)
}

func TestBrotliResponse(t *testing.T) {
	srv := newHTTPServer()
	defer srv.Close()

	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	agent.BaseURL = srv.URL

	req, err := agent.Get("/br")
	if err != nil {
		t.Fatalf("%+v", err)
	}

	res, err := agent.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("%#v", res)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if bytes.Compare(body, []byte("test it")) != 0 {
		t.Fatalf("%s missmatch %s", body, "test it")
	}
}

func TestGzipResponse(t *testing.T) {
	srv := newHTTPServer()
	defer srv.Close()

	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	agent.BaseURL = srv.URL

	req, err := agent.Get("/gzip")
	if err != nil {
		t.Fatalf("%+v", err)
	}

	res, err := agent.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("%#v", res)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if bytes.Compare(body, []byte("test it")) != 0 {
		t.Fatalf("%s missmatch %s", body, "test it")
	}
}

func TestDeflateResponse(t *testing.T) {
	srv := newHTTPServer()
	defer srv.Close()

	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	agent.BaseURL = srv.URL

	req, err := agent.Get("/deflate")
	if err != nil {
		t.Fatalf("%+v", err)
	}

	res, err := agent.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("%#v", res)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if bytes.Compare(body, []byte("test it")) != 0 {
		t.Fatalf("%s missmatch %s", body, "test it")
	}
}

func TestCacheControl(t *testing.T) {
	srv := newHTTPServer()
	defer srv.Close()

	agent, err := NewAgent()
	if err != nil {
		t.Fatal(err)
	}
	agent.BaseURL = srv.URL

	req, err := agent.Get("/dot.gif")
	if err != nil {
		t.Fatal(err)
	}
	res, err := agent.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	body, err := httputil.DumpResponse(res, true)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", body)

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

	body, err = httputil.DumpResponse(res, true)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", body)
}
