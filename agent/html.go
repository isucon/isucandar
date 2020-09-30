package agent

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/isucon/isucandar/failure"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Resource struct {
	InitiatorType string
	Request       *http.Request
	Response      *http.Response
	Error         error
}

type Resources map[string]*Resource

func (a *Agent) ProcessHTML(ctx context.Context, r *http.Response, body io.ReadCloser) (Resources, error) {
	defer body.Close()

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	resources := make(Resources)
	base := &*r.Request.URL
	baseChanged := false

	n := int32(0)
	favicon := &n

	resourceCollect := func(token html.Token) {
		defer wg.Done()
		var res *Resource
		switch token.DataAtom {
		case atom.Link:
			res = a.processHTMLLink(ctx, base, token)
		case atom.Script:
			res = a.processHTMLScript(ctx, base, token)
		case atom.Img:
			res = a.processHTMLImage(ctx, base, token)
		}

		if res != nil && res.Request != nil {
			if res.InitiatorType == "favicon" {
				atomic.StoreInt32(favicon, 1)
			}
			mu.Lock()
			resources[res.Request.URL.String()] = res
			mu.Unlock()
		}
	}

	doc := html.NewTokenizer(body)
	for tokenType := doc.Next(); tokenType != html.ErrorToken; tokenType = doc.Next() {
		token := doc.Token()
		if token.Type == html.StartTagToken || token.Type == html.SelfClosingTagToken {
			switch token.DataAtom {
			case atom.Base:
				if baseChanged {
					break
				}
				baseChanged = true
				href := ""
				for _, attr := range token.Attr {
					switch attr.Key {
					case "href":
						href = attr.Val
					}
				}
				if href != "" {
					newBaseURL, err := url.Parse(href)
					if err == nil {
						base = base.ResolveReference(newBaseURL)
					}
				}
			case atom.Link, atom.Script, atom.Img:
				wg.Add(1)
				go resourceCollect(token)
			}

		}
	}

	wg.Wait()

	// Automated favicon fetcher
	if atomic.LoadInt32(favicon) == 0 {
		if res := a.getResource(ctx, base, "/favicon.ico", "favicon"); res != nil && res.Request != nil {
			resources[res.Request.URL.String()] = res
		}
	}

	err := doc.Err()
	if failure.Is(err, io.EOF) {
		err = nil
	}
	return resources, err
}

func (a *Agent) processHTMLLink(ctx context.Context, base *url.URL, token html.Token) *Resource {
	rel := ""
	href := ""
	for _, attr := range token.Attr {
		switch attr.Key {
		case "rel":
			rel = attr.Val
		case "href":
			href = attr.Val
		}
	}

	switch rel {
	case "stylesheet":
		return a.getResource(ctx, base, href, "stylesheet")
	case "icon", "shortcut icon":
		return a.getResource(ctx, base, href, "favicon")
	case "apple-touch-icon", "apple-touch-icon-precomposed":
		return a.getResource(ctx, base, href, "apple-touch-icon")
	case "manifest":
		return a.getResource(ctx, base, href, "manifest")
	}

	return nil
}

func (a *Agent) processHTMLScript(ctx context.Context, base *url.URL, token html.Token) *Resource {
	src := ""
	for _, attr := range token.Attr {
		switch attr.Key {
		case "src":
			src = attr.Val
		}
	}

	if src == "" {
		return nil
	}

	return a.getResource(ctx, base, src, "script")
}

func (a *Agent) processHTMLImage(ctx context.Context, base *url.URL, token html.Token) *Resource {
	src := ""
	lazy := false // loading="lazy"
	for _, attr := range token.Attr {
		switch attr.Key {
		case "src":
			src = attr.Val
		case "loading":
			lazy = attr.Val == "lazy"
		}
	}

	if lazy || src == "" {
		return nil
	}

	return a.getResource(ctx, base, src, "img")
}

func (a *Agent) getResource(ctx context.Context, base *url.URL, ref string, initiatorType string) (res *Resource) {
	res = &Resource{
		InitiatorType: initiatorType,
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		res.Error = err
		return
	}
	refURL = base.ResolveReference(refURL)

	hreq, err := a.GET(refURL.String())
	if err != nil {
		res.Error = err
		return
	}
	res.Request = hreq

	hres, err := a.Do(ctx, hreq)
	if err != nil && err != io.EOF {
		res.Error = err
		return
	}
	res.Response = hres

	return
}
