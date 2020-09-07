package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"
	"time"
)

func get(a *Agent, path string) (*http.Request, *http.Response, error) {
	req, err := a.Get(path)
	if err != nil {
		return nil, nil, err
	}
	res, err := a.Do(context.Background(), req)
	if err != nil {
		return req, nil, err
	}

	return req, res, nil
}

func TestCacheWithLastModified(t *testing.T) {
	lm := time.Now().UTC()
	lm = lm.Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rb, _ := httputil.DumpRequest(r, false)
		t.Logf("%s", rb)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Last-Modified", lm.Format(http.TimeFormat))
		ims, _ := http.ParseTime(r.Header.Get("If-Modified-Since"))

		t.Logf("%v : %v", lm, ims)
		if !lm.Equal(ims) {
			w.WriteHeader(http.StatusOK)

			io.WriteString(w, "Hello, World")
		} else {
			w.WriteHeader(http.StatusNotModified)
		}
	}))
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	req, res, err := get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	c := agent.CacheStore.Get(req)
	t.Logf("%+v", c)

	req, res, err = get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 304 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	c = agent.CacheStore.Get(req)
	t.Logf("%+v", c)
}

func TestCacheWithETag(t *testing.T) {
	etag := "W/deadbeaf"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rb, _ := httputil.DumpRequest(r, false)
		t.Logf("%s", rb)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("ETag", etag)
		inm := r.Header.Get("If-None-Match")

		t.Logf("%v : %v", etag, inm)
		if etag != inm {
			w.WriteHeader(http.StatusOK)

			io.WriteString(w, "Hello, World")
		} else {
			w.WriteHeader(http.StatusNotModified)
		}
	}))
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	req, res, err := get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	c := agent.CacheStore.Get(req)
	t.Logf("%+v", c)

	req, res, err = get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 304 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	c = agent.CacheStore.Get(req)
	t.Logf("%+v", c)
}

func TestCacheWithMaxAge(t *testing.T) {
	reqCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "max-age=2")
		w.WriteHeader(http.StatusOK)

		io.WriteString(w, "Hello, World")

		reqCount++
	}))
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	req, res, err := get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	c := agent.CacheStore.Get(req)
	t.Logf("%+v", c)

	req, res, err = get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	c = agent.CacheStore.Get(req)
	t.Logf("%+v", c)

	<-time.After(3 * time.Second)
	get(agent, "/")

	if reqCount != 2 {
		t.Fatalf("missmatch req count: %d", reqCount)
	}
}

func TestCacheWithExpires(t *testing.T) {
	reqCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Expires", time.Now().UTC().Add(1*time.Second).Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)

		io.WriteString(w, "Hello, World")

		reqCount++
	}))
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	req, res, err := get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	c := agent.CacheStore.Get(req)
	t.Logf("%+v", c)

	req, res, err = get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	c = agent.CacheStore.Get(req)
	t.Logf("%+v", c)

	<-time.After(3 * time.Second)
	get(agent, "/")

	if reqCount != 2 {
		t.Fatalf("missmatch req count: %d", reqCount)
	}
}

func TestCacheWithVary(t *testing.T) {
	reqCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "max-age=200000, public")
		w.Header().Add("Vary", "User-Agent")
		w.Header().Add("Vary", "X-Cache-Count")
		w.WriteHeader(http.StatusOK)

		io.WriteString(w, "Hello, World")

		reqCount++
	}))
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	req, err := agent.Get("/")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	a := req.Clone(ctx)
	agent.Do(ctx, a)
	a = req.Clone(ctx)
	agent.Name = "Hoge"
	agent.Do(ctx, a)
	a = req.Clone(ctx)
	a.Header.Set("X-Cache-Count", "3")
	agent.Do(ctx, a)

	if reqCount != 3 {
		t.Fatalf("missmatch req count: %d", reqCount)
	}
}
