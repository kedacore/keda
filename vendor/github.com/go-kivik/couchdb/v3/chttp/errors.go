package chttp

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
)

// HTTPError is an error that represents an HTTP transport error.
type HTTPError struct {
	// Response is the HTTP response received by the client.  Typically the
	// response body has already been consumed, but the response and request
	// headers and other metadata will typically be in tact for debugging
	// purposes.
	Response *http.Response `json:"-"`

	// Reason is the server-supplied error reason.
	Reason string `json:"reason"`

	exitStatus int
}

func (e *HTTPError) Error() string {
	if e.Reason == "" {
		return http.StatusText(e.StatusCode())
	}
	if statusText := http.StatusText(e.StatusCode()); statusText != "" {
		return fmt.Sprintf("%s: %s", statusText, e.Reason)
	}
	return e.Reason
}

// StatusCode returns the embedded status code.
func (e *HTTPError) StatusCode() int {
	return e.Response.StatusCode
}

// ExitStatus returns the embedded exit status.
func (e *HTTPError) ExitStatus() int {
	return e.exitStatus
}

// Format implements fmt.Formatter
func (e *HTTPError) Format(f fmt.State, c rune) {
	formatError(e, f, c)
}

// FormatError satisfies the Go 1.13 errors.Formatter interface
// (golang.org/x/xerrors.Formatter for older versions of Go).
func (e *HTTPError) FormatError(p printer) error {
	p.Print(e.Error())
	if p.Detail() {
		p.Printf("REQUEST: %s %s (%d bytes)", e.Response.Request.Method, e.Response.Request.URL.String(), e.Response.Request.ContentLength)
		p.Printf("\nRESPONSE: %d / %s (%d bytes)\n", e.Response.StatusCode, http.StatusText(e.Response.StatusCode), e.Response.ContentLength)
	}
	return nil
}

// ResponseError returns an error from an *http.Response.
func ResponseError(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}
	if resp.Body != nil {
		defer resp.Body.Close() // nolint: errcheck
	}
	httpErr := &HTTPError{
		Response:   resp,
		exitStatus: ExitNotRetrieved,
	}
	if resp.Request.Method != "HEAD" && resp.ContentLength != 0 {
		if ct, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); ct == typeJSON {
			_ = json.NewDecoder(resp.Body).Decode(httpErr)
		}
	}
	return httpErr
}

type curlError struct {
	curlStatus int
	httpStatus int
	error
}

func (e *curlError) ExitStatus() int {
	return e.curlStatus
}

func (e *curlError) StatusCode() int {
	return e.httpStatus
}

func fullError(httpStatus, curlStatus int, err error) error {
	return &curlError{
		curlStatus: curlStatus,
		httpStatus: httpStatus,
		error:      err,
	}
}

func (e *curlError) Cause() error {
	return e.error
}

func (e *curlError) Unwrap() error {
	return e.error
}
