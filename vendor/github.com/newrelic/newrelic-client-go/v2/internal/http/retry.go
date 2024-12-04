package http

import (
	"bytes"
	"context"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"github.com/valyala/fastjson"
)

var (
	// A regular expression to match the error returned by net/http when the
	// configured number of redirects is exhausted. This error isn't typed
	// specifically so we resort to matching on the error string.
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)

	// A regular expression to match the error returned by net/http when the
	// scheme specified in the URL is invalid. This error isn't typed
	// specifically so we resort to matching on the error string.
	schemeErrorRe = regexp.MustCompile(`unsupported protocol scheme`)
)

// RetryPolicy provides a callback for retryablehttp's CheckRetry, which
// will retry on connection errors and server errors.
func RetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// do not retry on context.Canceled or context.DeadlineExceeded
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		if v, ok := err.(*url.Error); ok {
			// Don't retry if the error was due to too many redirects.
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to an invalid protocol scheme.
			if schemeErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to TLS cert verification failure.
			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, nil
			}
		}

		// The error is likely recoverable so retry.
		return true, nil
	}

	// Check the response code. We retry on 5xx responses to allow
	// the server time to recover, as 5xx's are typically not permanent
	// errors and may relate to outages on the server side. We are notably
	// disallowing retries for 500 errors here as the underlying APIs use them to
	// provide useful error validation messages that should be passed back to the
	// end user. This will catch invalid response codes as well, like 0 and 999.
	// 429 Too Many Requests is retried as well to handle the aggressive rate limiting
	// of the Synthetics API.
	if resp.StatusCode == 0 ||
		resp.StatusCode == 429 ||
		resp.StatusCode == 500 ||
		resp.StatusCode >= 502 {
		return true, nil
	}

	// Check for json response and description for retries.
	// i.e.: when receiving TOO_MANY_REQUESTS it is a HTTP 200 code with an
	// error status that needed to be retried.
	json, jErr := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewBuffer(json))
	if jErr == nil {
		errorClass := fastjson.GetString(json, "errors", "0", "extensions", "errorClass")
		if errorClass == ErrClassTooManyRequests.Error() {
			return true, ErrClassTooManyRequests
		}
	}

	return false, nil
}
