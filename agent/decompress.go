package agent

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dsnet/compress/brotli"
)

func decompress(res *http.Response) (*http.Response, error) {
	contentEncoding := res.Header.Get("Content-Encoding")
	if contentEncoding == "" {
		return res, nil
	}

	var err error
	var body io.ReadCloser = res.Body

	encodings := strings.Split(contentEncoding, ",")
	for i := len(encodings) - 1; i >= 0; i-- {
		encoding := encodings[i]
		switch strings.TrimSpace(encoding) {
		case "br":
			body, err = brotli.NewReader(body, &brotli.ReaderConfig{})
		case "gzip":
			body = &gzipReader{body: body}
		case "deflate":
			body = flate.NewReader(body)
		case "identity", "":
			// nop
		default:
			err = fmt.Errorf("unknown content encoding: %s: %w", encoding, ErrUnknownContentEncoding)
		}

		if err != nil {
			return nil, err
		}
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

func (gz *gzipReader) Read(p []byte) (int, error) {
	if gz.zr == nil {
		var err error
		gz.zr, err = gzip.NewReader(gz.body)
		if err != nil {
			return 0, err
		}
	}

	return gz.zr.Read(p)
}

func (gz *gzipReader) Close() error {
	return gz.body.Close()
}
