package chttp

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
)

var clientTraceContextKey = &struct{ name string }{"client trace"}

// ContextClientTrace returns the ClientTrace associated with the
// provided context. If none, it returns nil.
func ContextClientTrace(ctx context.Context) *ClientTrace {
	trace, _ := ctx.Value(clientTraceContextKey).(*ClientTrace)
	return trace
}

// ClientTrace is a set of hooks to run at various stages of an outgoing
// HTTP request. Any particular hook may be nil. Functions may be
// called concurrently from different goroutines and some may be called
// after the request has completed or failed.
type ClientTrace struct {
	// HTTPResponse returns a cloe of the *http.Response received from the
	// server, with the body set to nil. If you need the body, use the more
	// expensive HTTPResponseBody.
	HTTPResponse func(*http.Response)

	// HTTPResponseBody returns a clone of the *http.Response received from the
	// server, with the body cloned. This can be expensive for responses
	// with large bodies.
	HTTPResponseBody func(*http.Response)

	// HTTPRequest returns a clone of the *http.Request sent to the server, with
	// the body set to nil. If you need the body, use the more expensive
	// HTTPRequestBody.
	HTTPRequest func(*http.Request)

	// HTTPRequestBody returns a clone of the *http.Request sent to the server,
	// with the body cloned, if it is set. This can be expensive for requests
	// with large bodies.
	HTTPRequestBody func(*http.Request)
}

// WithClientTrace returns a new context based on the provided parent
// ctx. HTTP client requests made with the returned context will use
// the provided trace hooks, in addition to any previous hooks
// registered with ctx. Any hooks defined in the provided trace will
// be called first.
func WithClientTrace(ctx context.Context, trace *ClientTrace) context.Context {
	if trace == nil {
		panic("nil trace")
	}
	return context.WithValue(ctx, clientTraceContextKey, trace)
}

func (t *ClientTrace) httpResponse(r *http.Response) {
	if t.HTTPResponse == nil || r == nil {
		return
	}
	clone := new(http.Response)
	*clone = *r
	clone.Body = nil
	t.HTTPResponse(clone)
}

func (t *ClientTrace) httpResponseBody(r *http.Response) {
	if t.HTTPResponseBody == nil || r == nil {
		return
	}
	clone := new(http.Response)
	*clone = *r
	rBody := r.Body
	body, readErr := ioutil.ReadAll(rBody)
	closeErr := rBody.Close()
	r.Body = newReplay(body, readErr, closeErr)
	clone.Body = newReplay(body, readErr, closeErr)
	t.HTTPResponseBody(clone)
}

func (t *ClientTrace) httpRequest(r *http.Request) {
	if t.HTTPRequest == nil {
		return
	}
	clone := new(http.Request)
	*clone = *r
	clone.Body = nil
	t.HTTPRequest(clone)
}

func (t *ClientTrace) httpRequestBody(r *http.Request) {
	if t.HTTPRequestBody == nil {
		return
	}
	clone := new(http.Request)
	*clone = *r
	if r.Body != nil {
		rBody := r.Body
		body, readErr := ioutil.ReadAll(rBody)
		closeErr := rBody.Close()
		r.Body = newReplay(body, readErr, closeErr)
		clone.Body = newReplay(body, readErr, closeErr)
	}
	t.HTTPRequestBody(clone)
}

func newReplay(body []byte, readErr, closeErr error) io.ReadCloser {
	if readErr == nil && closeErr == nil {
		return ioutil.NopCloser(bytes.NewReader(body))
	}
	return &replayReadCloser{
		Reader:   ioutil.NopCloser(bytes.NewReader(body)),
		readErr:  readErr,
		closeErr: closeErr,
	}
}

// replayReadCloser replays read and close errors
type replayReadCloser struct {
	io.Reader
	readErr  error
	closeErr error
}

func (r *replayReadCloser) Read(p []byte) (int, error) {
	c, err := r.Reader.Read(p)
	if err == io.EOF && r.readErr != nil {
		err = r.readErr
	}
	return c, err
}

func (r *replayReadCloser) Close() error {
	return r.closeErr
}
