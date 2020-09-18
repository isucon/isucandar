package agent

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

const (
	DefaultConnections    = 10000
	DefaultName           = "isucandar"
	DefaultAccept         = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
	DefaultRequestTimeout = 1 * time.Second
)

var (
	DefaultTLSConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	DefaultDialer    *net.Dialer
	DefaultTransport *http.Transport
)

func init() {
	DefaultDialer = &net.Dialer{
		Timeout:   0,
		KeepAlive: 60 * time.Second,
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		Dial:                  DefaultDialer.Dial,
		DialContext:           DefaultDialer.DialContext,
		TLSClientConfig:       DefaultTLSConfig,
		DisableCompression:    true,
		MaxIdleConns:          0,
		MaxIdleConnsPerHost:   DefaultConnections,
		MaxConnsPerHost:       0,
		TLSHandshakeTimeout:   0,
		ResponseHeaderTimeout: 0,
		IdleConnTimeout:       0,
		ForceAttemptHTTP2:     true,
	}

	DefaultTransport = transport
}

type AgentOption func(*Agent) error

type Agent struct {
	Name          string
	BaseURL       *url.URL
	DefaultAccept string
	CacheStore    CacheStore
	HttpClient    *http.Client
}

func NewAgent(opts ...AgentOption) (*Agent, error) {
	jar, _ := cookiejar.New(&cookiejar.Options{})

	agent := &Agent{
		Name:          DefaultName,
		BaseURL:       nil,
		DefaultAccept: DefaultAccept,
		CacheStore:    NewCacheStore(),
		HttpClient: &http.Client{
			CheckRedirect: useLastResponse,
			Transport:     DefaultTransport,
			Jar:           jar,
			Timeout:       DefaultRequestTimeout,
		},
	}

	for _, opt := range opts {
		if err := opt(agent); err != nil {
			return nil, err
		}
	}

	return agent, nil
}

func (a *Agent) ClearCookie() {
	if a.HttpClient.Jar != nil {
		jar, _ := cookiejar.New(&cookiejar.Options{})
		a.HttpClient.Jar = jar
	}
}

func (a *Agent) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)

	var cache *Cache
	if a.CacheStore != nil {
		cache = a.CacheStore.Get(req)
	}
	if cache != nil {
		cache.apply(req)
	}

	var res *http.Response
	var err error

	if cache != nil && !cache.requiresRevalidate(req) {
		res = cache.restoreResponse()
	} else {
		res, err = a.HttpClient.Do(req)
		if err != nil {
			return nil, err
		}

		res, err = decompress(res)
		if err != nil {
			return nil, err
		}
	}

	cache, err = newCache(res, cache.Body())
	if err != nil {
		return nil, err
	}

	if cache != nil && a.CacheStore != nil {
		a.CacheStore.Put(req, cache)
	}

	return res, nil
}

func (a *Agent) NewRequest(method string, target string, body io.Reader) (*http.Request, error) {
	reqURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	if a.BaseURL != nil {
		reqURL = a.BaseURL.ResolveReference(reqURL)
	}

	req, err := http.NewRequest(method, reqURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", a.Name)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", a.DefaultAccept)
	}

	return req, nil
}

func (a *Agent) GET(target string) (*http.Request, error) {
	return a.NewRequest(http.MethodGet, target, nil)
}

func (a *Agent) POST(target string, body io.Reader) (*http.Request, error) {
	return a.NewRequest(http.MethodPost, target, body)
}

func (a *Agent) PUT(target string, body io.Reader) (*http.Request, error) {
	return a.NewRequest(http.MethodPut, target, body)
}

func (a *Agent) PATCH(target string, body io.Reader) (*http.Request, error) {
	return a.NewRequest(http.MethodPatch, target, body)
}

func (a *Agent) DELETE(target string, body io.Reader) (*http.Request, error) {
	return a.NewRequest(http.MethodDelete, target, body)
}

func useLastResponse(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}
