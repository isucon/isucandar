package agent

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"

	"github.com/dsnet/compress/brotli"
)

func decompress(res *http.Response) (*http.Response, error) {
	ce := res.Header.Get("Content-Encoding")

	var err error
	var body io.ReadCloser
	switch ce {
	case "br":
		body, err = brotli.NewReader(res.Body, &brotli.ReaderConfig{})
	case "gzip":
		body = &gzipReader{body: res.Body}
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

type gzipReader struct {
	body io.ReadCloser
	zr   *gzip.Reader
	zerr error
}

func (gz *gzipReader) Read(p []byte) (n int, err error) {
	if gz.zr == nil {
		if gz.zerr == nil {
			gz.zr, gz.zerr = gzip.NewReader(gz.body)
		}
		if gz.zerr != nil {
			return 0, gz.zerr
		}
	}

	if err != nil {
		return 0, err
	}
	return gz.zr.Read(p)
}

func (gz *gzipReader) Close() error {
	return gz.body.Close()
}
