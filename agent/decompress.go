package agent

import (
	"compress/flate"
	"compress/gzip"
	"github.com/dsnet/compress/brotli"
	"net/http"
)

func decompress(res *http.Response) (*http.Response, error) {
	ce := res.Header.Get("Content-Encoding")

	var err error
	switch ce {
	case "br":
		res.Body, err = brotli.NewReader(res.Body, &brotli.ReaderConfig{})
		if err != nil {
			return nil, err
		}
	case "gzip":
		res.Body, err = gzip.NewReader(res.Body)
		if err != nil {
			return nil, err
		}
	case "deflate":
		res.Body = flate.NewReader(res.Body)
	default:
		return res, nil
	}

	res.Header.Del("Content-Length")
	res.ContentLength = -1
	res.Uncompressed = true

	return res, nil
}
