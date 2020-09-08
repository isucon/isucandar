package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const exampleHTMLDoc = `
<!DOCTYPE html>
<html>
	<head>
		<title>This is agent test</title>
		<base href="/sub/dir/">
		<base href="/sub/dir/twice">
		stylesheet じゃないならロードしない
		<link rel="apple-touch-icon" sizes="114x114" href="/apple-icon.png" type="image/png">
		<link rel="apple-touch-icon-precomposed" sizes="114x114" href="/apple-icon-precomposed.png" type="image/png">
		<link rel="stylesheet" href="/root.css">
		<link rel="stylesheet" href="../alt.css">
		<link rel="dns-prefetch" href="https://example.com">
		<link rel="manifest" href="/manifest.webmanifest">
	</head>
	<body>
		<h1>Hello, World</h1>

		<img src="/cute.png">
		<img src="beautiful.png" />
		<img src="$://broken/url">
		loading=lazy ならロードしない
		<img src="lazy.png" loading="lazy" />

		<script src="/need.js"></script>
		<script src="/defer.js" defer></script>
		<script src="/async.js" async></script>
		インラインスクリプトは無視する
		<script>console.log("inline script");</script>

		絶対 URL の指定をした場合
		<script src="https://example.com/"></script>
		<script src="http://example.com/"></script>
	</body>
</html>
`

func TestHTMLParse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, exampleHTMLDoc)
	}))
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	_, res, err := get(agent, "/test.html")
	if err != nil {
		t.Fatal(err)
	}

	resources, err := agent.ProcessHTML(context.Background(), res, res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if len(resources) != 13 {
		for k, _ := range resources {
			t.Log(k)
		}
		t.Fatalf("resouces count missmatch: %d", len(resources))
	}

	expects := []string{
		srv.URL + "/root.css",
		srv.URL + "/sub/alt.css",
		srv.URL + "/cute.png",
		srv.URL + "/sub/dir/beautiful.png",
		srv.URL + "/need.js",
		srv.URL + "/defer.js",
		srv.URL + "/async.js",
		srv.URL + "/favicon.ico",
		srv.URL + "/apple-icon-precomposed.png",
		srv.URL + "/apple-icon.png",
		srv.URL + "/manifest.webmanifest",
		"https://example.com/",
		"http://example.com/",
	}

	for _, eURL := range expects {
		if _, ok := resources[eURL]; !ok {
			t.Fatalf("resouce not reached: %s", eURL)
		}
	}
}

func BenchmarkHTMLParse(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(200)
		io.WriteString(w, exampleHTMLDoc)
	}))
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, res, err := get(agent, "/test.html")
		if err != nil {
			b.Fatal(err)
		}

		resources, err := agent.ProcessHTML(context.Background(), res, res.Body)
		if err != nil {
			b.Fatal(err)
		}

		if len(resources) != 12 {
			for k, _ := range resources {
				b.Log(k)
			}
			b.Fatalf("resouces count missmatch: %d", len(resources))
		}

		expects := []string{
			srv.URL + "/root.css",
			srv.URL + "/sub/alt.css",
			srv.URL + "/cute.png",
			srv.URL + "/sub/dir/beautiful.png",
			srv.URL + "/need.js",
			srv.URL + "/defer.js",
			srv.URL + "/async.js",
			srv.URL + "/favicon.ico",
			srv.URL + "/apple-icon-precomposed.png",
			srv.URL + "/apple-icon.png",
			"https://example.com/",
			"http://example.com/",
		}

		for _, eURL := range expects {
			if _, ok := resources[eURL]; !ok {
				b.Fatalf("resouce not reached: %s", eURL)
			}
		}
	}
}

const exampleFaviconDoc = `
<!DOCTYPE html>
<html>
	<head>
		<link rel="icon" href="x-favicon.ico">
		<link rel="shortcut icon" href="x-short-cut-favicon.ico">
	</head>
</html>
`

func TestFavicon(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, exampleFaviconDoc)
	}))
	defer srv.Close()

	agent, err := NewAgent(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	_, res, err := get(agent, "/test.html")
	if err != nil {
		t.Fatal(err)
	}

	resources, err := agent.ProcessHTML(context.Background(), res, res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if len(resources) != 2 {
		t.Fatalf("resouces count missmatch: %d", len(resources))
	}

	expects := []string{
		srv.URL + "/x-favicon.ico",
		srv.URL + "/x-short-cut-favicon.ico",
	}

	for _, eURL := range expects {
		if _, ok := resources[eURL]; !ok {
			t.Fatalf("resouce not reached: %s", eURL)
		}
	}
}
