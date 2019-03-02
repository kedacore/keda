/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package queue

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/knative/pkg/websocket"
)

var defaultTimeoutBody = "<html><head><title>Timeout</title></head><body><h1>Timeout</h1></body></html>"

// TimeToFirstByteTimeoutHandler returns a Handler that runs `h` with the
// given time limit in which the first byte of the response must be written.
//
// The new Handler calls h.ServeHTTP to handle each request, but if a
// call runs for longer than its time limit, the handler responds with
// a 503 Service Unavailable error and the given message in its body.
// (If msg is empty, a suitable default message will be sent.)
// After such a timeout, writes by h to its ResponseWriter will return
// ErrHandlerTimeout.
//
// A panic from the underlying handler is propagated as-is to be able to
// make use of custom panic behavior by HTTP handlers. See
// https://golang.org/pkg/net/http/#Handler.
//
// The implementation is largely inspired by http.TimeoutHandler.
func TimeToFirstByteTimeoutHandler(h http.Handler, dt time.Duration, msg string) http.Handler {
	return &timeoutHandler{
		handler: h,
		body:    msg,
		dt:      dt,
	}
}

type timeoutHandler struct {
	handler http.Handler
	body    string
	dt      time.Duration
}

func (h *timeoutHandler) errorBody() string {
	if h.body != "" {
		return h.body
	}
	return defaultTimeoutBody
}

func (h *timeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancelCtx := context.WithCancel(r.Context())
	defer cancelCtx()

	done := make(chan struct{})
	// The recovery value of a panic is written to this channel to be
	// propagated (panicked with) again.
	panicChan := make(chan interface{}, 1)
	defer close(panicChan)

	tw := &timeoutWriter{w: w}
	go func() {
		// The defer statements are executed in LIFO order,
		// so recover will execute first, then only, the channel will be closed.
		defer close(done)
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
		}()
		h.handler.ServeHTTP(tw, r.WithContext(ctx))
	}()

	timeout := time.After(h.dt)
	for {
		select {
		case p := <-panicChan:
			panic(p)
		case <-done:
			return
		case <-timeout:
			if tw.TimeoutAndWriteError(h.errorBody()) {
				return
			}
		}
	}
}

// timeoutWriter is a wrapper around an http.ResponseWriter. It guards
// writing an error response to whether or not the underlying writer has
// already been written to.
//
// If the underlying writer has not been written to, an error response is
// returned. If it has already been written to, the error is ignored and
// the response is allowed to continue.
type timeoutWriter struct {
	w http.ResponseWriter

	mu        sync.Mutex
	timedOut  bool
	wroteOnce bool
}

var _ http.Flusher = (*timeoutWriter)(nil)

var _ http.ResponseWriter = (*timeoutWriter)(nil)

func (tw *timeoutWriter) Flush() {
	tw.w.(http.Flusher).Flush()
}

// Hijack calls Hijack() on the wrapped http.ResponseWriter if it implements
// http.Hijacker interface, which is required for net/http/httputil/reverseproxy
// to handle connection upgrade/switching protocol.  Otherwise returns an error.
func (tw *timeoutWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return websocket.HijackIfPossible(tw.w)
}

func (tw *timeoutWriter) Header() http.Header { return tw.w.Header() }

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}

	tw.wroteOnce = true
	return tw.w.Write(p)
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return
	}

	tw.wroteOnce = true
	tw.w.WriteHeader(code)
}

// TimeoutAndError writes an error to the response write if
// nothing has been written on the writer before. Returns whether
// an error was written or not.
//
// If this writes an error, all subsequent calls to Write will
// result in http.ErrHandlerTimeout.
func (tw *timeoutWriter) TimeoutAndWriteError(msg string) bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if !tw.wroteOnce {
		tw.w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(tw.w, msg)

		tw.timedOut = true
		return true
	}

	return false
}
