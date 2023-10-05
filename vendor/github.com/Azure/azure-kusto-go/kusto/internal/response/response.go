package response

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
)

type originalCloser struct {
	original io.ReadCloser
	wrapper  io.ReadCloser
}

func (o *originalCloser) Read(p []byte) (n int, err error) {
	return o.wrapper.Read(p)
}

func (o *originalCloser) Close() error {
	if err := o.wrapper.Close(); err != nil {
		return err
	}
	return o.original.Close()
}

func TranslateBody(resp *http.Response, op errors.Op) (io.ReadCloser, error) {
	body := resp.Body
	var wrapper io.ReadCloser
	switch enc := strings.ToLower(resp.Header.Get("Content-Encoding")); enc {
	case "":
		return body, nil
	case "gzip":
		var err error
		wrapper, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, errors.E(op, errors.KInternal, fmt.Errorf("gzip reader error: %w", err))
		}
	case "deflate":
		wrapper = flate.NewReader(resp.Body)
	default:
		return nil, errors.ES(op, errors.KInternal, "Content-Encoding was unrecognized: %s", enc)
	}
	return &originalCloser{
		original: body,
		wrapper:  wrapper,
	}, nil
}
