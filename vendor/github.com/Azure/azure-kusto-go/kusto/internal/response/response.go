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

func TranslateBody(resp *http.Response, op errors.Op) (io.ReadCloser, error) {
	body := resp.Body
	switch enc := strings.ToLower(resp.Header.Get("Content-Encoding")); enc {
	case "":
		return body, nil
	case "gzip":
		var err error
		body, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, errors.E(op, errors.KInternal, fmt.Errorf("gzip reader error: %w", err))
		}
	case "deflate":
		body = flate.NewReader(resp.Body)
	default:
		return nil, errors.ES(op, errors.KInternal, "Content-Encoding was unrecognized: %s", enc)
	}
	return body, nil
}
