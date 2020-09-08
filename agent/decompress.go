package agent

import (
	"compress/flate"
	"compress/gzip"
	"github.com/dsnet/compress/brotli"
	"io"
	"net/http"
)

func decompress(res *http.Response) (*http.Response, error) {
	ce := res.Header.Get("Content-Encoding")

	var err error
	var body io.ReadCloser
	switch ce {
	case "br":
		body, err = brotli.NewReader(res.Body, &brotli.ReaderConfig{})
	case "gzip":
		body, err = gzip.NewReader(res.Body)
	case "deflate":
		body = flate.NewReader(res.Body)
	default:
		return res, nil
	}

	if err != nil {
		return nil, err
	}

	res.Header.Del("Content-Length")
	res.ContentLength = -1
	res.Uncompressed = true
	res.Body = body

	return res, nil
}
