package agent

import (
	"context"
	"github.com/rosylilly/isucandar/failure"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
)

type Resource struct {
	InitiatorType string
	Request       *http.Request
	Response      *http.Response
}

type Resources map[string]*Resource

func (a *Agent) ProcessHTML(ctx context.Context, r *http.Response, body io.ReadCloser) (Resources, error) {
	defer body.Close()

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	resouces := make(Resources)
	base := &*r.Request.URL
	baseChanged := false

	n := int32(0)
	favicon := &n

	resourceCollect := func(token html.Token) {
		defer wg.Done()
		var res *Resource
		var err error
		switch token.DataAtom {
		case atom.Link:
			res, err = a.processHTMLLink(ctx, base, token)
		case atom.Script:
			res, err = a.processHTMLScript(ctx, base, token)
		case atom.Img:
			res, err = a.processHTMLImage(ctx, base, token)
		}

		if res != nil && err == nil {
			if res.InitiatorType == "favicon" {
				atomic.StoreInt32(favicon, 1)
			}
			mu.Lock()
			resouces[res.Request.URL.String()] = res
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
		if res, err := a.getResource(ctx, base, "/favicon.ico", "favicon"); err == nil && res != nil {
			resouces[res.Request.URL.String()] = res
		}
	}

	err := doc.Err()
	if failure.Is(err, io.EOF) {
		err = nil
	}
	return resouces, err
}

func (a *Agent) processHTMLLink(ctx context.Context, base *url.URL, token html.Token) (*Resource, error) {
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
	}

	return nil, nil
}

func (a *Agent) processHTMLScript(ctx context.Context, base *url.URL, token html.Token) (*Resource, error) {
	src := ""
	for _, attr := range token.Attr {
		switch attr.Key {
		case "src":
			src = attr.Val
		}
	}

	if src == "" {
		return nil, nil
	}

	return a.getResource(ctx, base, src, "script")
}

func (a *Agent) processHTMLImage(ctx context.Context, base *url.URL, token html.Token) (*Resource, error) {
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
		return nil, nil
	}

	return a.getResource(ctx, base, src, "img")
}

func (a *Agent) getResource(ctx context.Context, base *url.URL, ref string, initiatorType string) (*Resource, error) {
	refURL, err := url.Parse(ref)
	if err != nil {
		return nil, err
	}
	refURL = base.ResolveReference(refURL)

	req, err := a.Get(refURL.String())
	if err != nil {
		return nil, err
	}

	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	return &Resource{
		InitiatorType: initiatorType,
		Request:       req,
		Response:      res,
	}, nil
}
