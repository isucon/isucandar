package agent

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pquerna/cachecontrol/cacheobject"
)

var (
	cacheableStatusCodes = map[int]bool{
		200: true,
		203: true,
		204: true,
		206: true,
		300: true,
		301: true,
		304: true,
		404: true,
		405: true,
		410: true,
		414: true,
		501: true,
	}
)

type Cache struct {
	mu            sync.RWMutex
	now           time.Time
	body          []byte
	res           *http.Response
	ReqDirectives *cacheobject.RequestCacheDirectives
	ResDirectives *cacheobject.ResponseCacheDirectives
	Expires       *time.Time
	Date          *time.Time
	LastModified  *time.Time
	ETag          *string
	Varies        []string
	VariesKey     string
}

func newCache(res *http.Response, cache *Cache) (*Cache, error) {
	// Do not cache request without get method
	if res.Request.Method != http.MethodGet {
		return nil, nil
	}

	// Do not cache request with authorization header
	if auth := res.Request.Header.Get("Authorization"); auth != "" {
		return nil, nil
	}

	// Do not cache specified status code
	if _, found := cacheableStatusCodes[res.StatusCode]; !found {
		return nil, nil
	}

	resDirs, err := cacheobject.ParseResponseCacheControl(res.Header.Get("Cache-Control"))
	if err != nil {
		return nil, err
	}

	if resDirs.NoStore {
		return nil, nil
	}

	reqDirs, err := cacheobject.ParseRequestCacheControl(res.Request.Header.Get("Cache-Control"))
	if err != nil {
		return nil, err
	}

	if cache == nil {
		cache = &Cache{
			mu:            sync.RWMutex{},
			now:           time.Now(),
			ReqDirectives: reqDirs,
			ResDirectives: resDirs,
		}
		cache.mu.Lock()
		defer cache.mu.Unlock()
	} else {
		cache.mu.Lock()
		defer cache.mu.Unlock()
		cache.ReqDirectives = reqDirs
		cache.ResDirectives = resDirs
	}

	if t, err := http.ParseTime(res.Header.Get("Expires")); err == nil {
		cache.Expires = &t
	}

	if t, err := http.ParseTime(res.Header.Get("Date")); err == nil {
		cache.Date = &t
	}

	if t, err := http.ParseTime(res.Header.Get("Last-Modified")); err == nil {
		cache.LastModified = &t
	}

	if etag := res.Header.Get("ETag"); len(etag) > 0 {
		cache.ETag = &etag
	}

	varies := make([]string, 0, 3)
	for _, v := range res.Header.Values("Vary") {
		for _, k := range strings.Split(v, ",") {
			varies = append(varies, strings.TrimSpace(k))
		}
	}
	sort.Strings(varies)
	cache.Varies = varies

	key := ""
	for _, h := range varies {
		key += strings.Join(res.Request.Header.Values(h), ", ")
	}
	cache.VariesKey = key

	if res.StatusCode != 304 {
		cache.res = res
		cache.body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		res.Body.Close()
	}
	res.Body = ioutil.NopCloser(bytes.NewReader(cache.body))

	return cache, nil
}

func (c *Cache) apply(req *http.Request) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.LastModified != nil {
		req.Header.Set("If-Modified-Since", c.LastModified.Format(http.TimeFormat))
	}
	if c.ETag != nil {
		req.Header.Set("If-None-Match", *c.ETag)
	}
}

func (c *Cache) isOutdated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now().UTC()

	if c.ResDirectives.MaxAge <= 0 && c.Expires == nil {
		return true
	}

	if c.ResDirectives.MaxAge > 0 && now.After(c.now.Add(time.Duration(c.ResDirectives.MaxAge)*time.Second)) {
		return true
	}
	return (c.Expires != nil && now.After(*c.Expires))
}

func (c *Cache) matchVariesKey(req *http.Request) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := ""
	for _, h := range c.Varies {
		key += strings.Join(req.Header.Values(h), ", ")
	}

	return key == c.VariesKey
}

func (c *Cache) requiresRevalidate(req *http.Request) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.ResDirectives.MustRevalidate || !c.matchVariesKey(req) || c.isOutdated()
}

func (c *Cache) restoreResponse() *http.Response {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var res http.Response
	res = *c.res
	res.Body = ioutil.NopCloser(bytes.NewReader(c.body))
	return &res
}
