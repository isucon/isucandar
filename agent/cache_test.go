package agent

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func req(a *Agent, method string, path string) (*http.Request, *http.Response, error) {
	req, err := a.NewRequest(method, path, nil)
	if err != nil {
		return nil, nil, err
	}
	res, err := a.Do(context.Background(), req)
	if err != nil {
		return req, nil, err
	}

	return req, res, nil
}

func get(a *Agent, path string) (*http.Request, *http.Response, error) {
	return req(a, http.MethodGet, path)
}

func TestCacheCondition(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/no-store":
			w.Header().Set("Cache-Control", "no-store, max-age=100")
		case "/invalid":
			w.Header().Set("Cache-Control", "private, max-age=-10")
		default:
			w.Header().Set("Cache-Control", "public, max-age=1000")
		}
		w.WriteHeader(200)
		io.WriteString(w, "OK")
	}))
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	r, _, _ := req(agent, "POST", "/")
	if cache := agent.CacheStore.Get(r); cache != nil {
		t.Fatalf("Stored invalid cache: %v", cache)
	}

	r, _ = agent.GET("/")
	r.Header.Set("Authorization", "Bearer X-TOKEN")
	agent.Do(context.Background(), r)
	if cache := agent.CacheStore.Get(r); cache != nil {
		t.Fatalf("Stored invalid cache: %v", cache)
	}

	r, _, _ = get(agent, "/no-store")
	if cache := agent.CacheStore.Get(r); cache != nil {
		t.Fatalf("Stored invalid cache: %v", cache)
	}

	r, _, _ = get(agent, "/invalid")
	if cache := agent.CacheStore.Get(r); cache != nil {
		t.Fatalf("Stored invalid cache: %v", cache)
	}

	r, _ = agent.GET("/")
	r.Header.Set("Cache-Control", "max-age=-1")
	agent.Do(context.Background(), r)
	if cache := agent.CacheStore.Get(r); cache != nil {
		t.Fatalf("Stored invalid cache: %v", cache)
	}
}

func TestCacheWithLastModified(t *testing.T) {
	lm := time.Now().UTC()
	lm = lm.Truncate(time.Second)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Last-Modified", lm.Format(http.TimeFormat))
		ims, _ := http.ParseTime(r.Header.Get("If-Modified-Since"))

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

	_, res, err := get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	_, res, err = get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 304 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %+v", err)
	}

	if string(body) != "Hello, World" {
		t.Fatalf("body missmatch: %x", body)
	}

	_, res, err = get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 304 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %+v", err)
	}

	if string(body) != "Hello, World" {
		t.Fatalf("body missmatch: %x", body)
	}
}

func TestCacheWithETag(t *testing.T) {
	etag := "W/deadbeaf"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("ETag", etag)
		inm := r.Header.Get("If-None-Match")

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

	_, res, err := get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	_, res, err = get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 304 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	_, res, err = get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 304 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %+v", err)
	}

	if string(body) != "Hello, World" {
		t.Fatalf("body missmatch: %x", body)
	}

	_, res, err = get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 304 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %+v", err)
	}

	if string(body) != "Hello, World" {
		t.Fatalf("body missmatch: %x", body)
	}

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

	_, res, err := get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	req, res, err := get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	c := agent.CacheStore.Get(req)
	c.now = time.Now().Add(-3 * time.Second)

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

	_, res, err := get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	_, res, err = get(agent, "/")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	<-time.After(1 * time.Second)
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

	req, err := agent.GET("/")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	a := req.Clone(ctx)
	agent.Do(ctx, a)
	a = req.Clone(ctx)
	a.Header.Set("User-Agent", "Hoge")
	agent.Do(ctx, a)
	a = req.Clone(ctx)
	a.Header.Set("X-Cache-Count", "3")
	agent.Do(ctx, a)

	if reqCount != 3 {
		t.Fatalf("missmatch req count: %d", reqCount)
	}
}

func TestCacheWithClear(t *testing.T) {
	reqCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "max-age=20000")
		w.WriteHeader(http.StatusOK)

		io.WriteString(w, "Hello, World")

		reqCount++
	}))
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	get(agent, "/")
	agent.CacheStore.Clear()
	get(agent, "/")

	if reqCount != 2 {
		t.Fatalf("missmatch req count: %d", reqCount)
	}
}

func BenchmarkCacheWithMaxAge(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "max-age=20000")
		w.WriteHeader(http.StatusOK)

		io.WriteString(w, "Hello, World")
	}))
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		b.Fatal(err)
	}

	_, res, err := get(agent, "/")
	if err != nil {
		b.Fatal(err)
	}

	if res.StatusCode != 200 {
		b.Fatalf("status code missmatch: %d", res.StatusCode)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, res, err := get(agent, "/")
		if err != nil {
			b.Fatal(err)
		}

		if res.StatusCode != 200 && res.StatusCode != 304 {
			b.Fatalf("status code missmatch: %d", res.StatusCode)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			b.Fatal(err)
		}

		if string(body) != "Hello, World" {
			b.Fatal("body missmatch")
		}
	}
}
