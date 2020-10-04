package agent

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/middleware"

	"github.com/andybalholm/brotli"
	"github.com/julienschmidt/httprouter"
	"github.com/labstack/echo"
)

func newCompressHTTPServer() *httptest.Server {
	r := httprouter.New()

	r.GET("/br", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "br")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(200)
		bw := brotli.NewWriter(w)
		defer bw.Close()
		io.WriteString(bw, "test it")
	})
	r.GET("/broken-br", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "br")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(200)
		io.WriteString(w, "test it")
	})
	r.GET("/gzip", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		gw := gzip.NewWriter(w)
		defer gw.Close()

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(200)
		io.WriteString(gw, "test it")
	})
	r.GET("/broken-gzip", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(200)
		io.WriteString(w, "test it")
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
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(200)
		io.WriteString(fw, "test it")
	})
	r.GET("/broken-deflate", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "deflate")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(200)
		io.WriteString(w, "test it")
	})

	return httptest.NewServer(r)
}

func TestBrotliResponse(t *testing.T) {
	srv := newCompressHTTPServer()
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("%+v", err)
	}

	req, err := agent.GET("/br")
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

	_, res, err = get(agent, "/broken-br")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ioutil.ReadAll(res.Body)
	if err == nil {
		t.Fatalf("Not raised error with broken encoding")
	}
}

func TestGzipResponse(t *testing.T) {
	srv := newCompressHTTPServer()
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("%+v", err)
	}

	req, err := agent.GET("/gzip")
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

	_, res, err = get(agent, "/broken-gzip")
	if err != nil {
		t.Fatal(err)
	}

	body, err = ioutil.ReadAll(res.Body)
	if err == nil {
		t.Fatalf("Not raised error with broken encoding")
	}
}

func TestDeflateResponse(t *testing.T) {
	srv := newCompressHTTPServer()
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("%+v", err)
	}

	req, err := agent.GET("/deflate")
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

	_, res, err = get(agent, "/broken-deflate")
	if err != nil {
		t.Fatal(err)
	}

	body, err = ioutil.ReadAll(res.Body)
	if err == nil {
		t.Fatalf("Not raised error with broken encoding")
	}
}

func TestWithEcho(t *testing.T) {
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "public, max-age=10000")
		return c.String(200, "test it")
	})
	e.Use(middleware.Gzip())
	srv := httptest.NewServer(e)
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("%+v", err)
	}

	for i := 0; i < 3; i++ {
		req, err := agent.GET("/")
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

		t.Logf("%+v", res)

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if bytes.Compare(body, []byte("test it")) != 0 {
			t.Fatalf("%s missmatch %s", body, "test it")
		}
		<-time.After(1 * time.Second)
	}
}
