// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// Code generated from specification version 7.17.10: DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"net/http"
	"strings"
)

func newWatcherQueryWatchesFunc(t Transport) WatcherQueryWatches {
	return func(o ...func(*WatcherQueryWatchesRequest)) (*Response, error) {
		var r = WatcherQueryWatchesRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// WatcherQueryWatches - Retrieves stored watches.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/watcher-api-query-watches.html.
type WatcherQueryWatches func(o ...func(*WatcherQueryWatchesRequest)) (*Response, error)

// WatcherQueryWatchesRequest configures the Watcher Query Watches API request.
type WatcherQueryWatchesRequest struct {
	Body io.Reader

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r WatcherQueryWatchesRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(len("/_watcher/_query/watches"))
	path.WriteString("/_watcher/_query/watches")

	params = make(map[string]string)

	if r.Pretty {
		params["pretty"] = "true"
	}

	if r.Human {
		params["human"] = "true"
	}

	if r.ErrorTrace {
		params["error_trace"] = "true"
	}

	if len(r.FilterPath) > 0 {
		params["filter_path"] = strings.Join(r.FilterPath, ",")
	}

	req, err := newRequest(method, path.String(), r.Body)
	if err != nil {
		return nil, err
	}

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	if len(r.Header) > 0 {
		if len(req.Header) == 0 {
			req.Header = r.Header
		} else {
			for k, vv := range r.Header {
				for _, v := range vv {
					req.Header.Add(k, v)
				}
			}
		}
	}

	if r.Body != nil && req.Header.Get(headerContentType) == "" {
		req.Header[headerContentType] = headerContentTypeJSON
	}

	if ctx != nil {
		req = req.WithContext(ctx)
	}

	res, err := transport.Perform(req)
	if err != nil {
		return nil, err
	}

	response := Response{
		StatusCode: res.StatusCode,
		Body:       res.Body,
		Header:     res.Header,
	}

	return &response, nil
}

// WithContext sets the request context.
func (f WatcherQueryWatches) WithContext(v context.Context) func(*WatcherQueryWatchesRequest) {
	return func(r *WatcherQueryWatchesRequest) {
		r.ctx = v
	}
}

// WithBody - From, size, query, sort and search_after.
func (f WatcherQueryWatches) WithBody(v io.Reader) func(*WatcherQueryWatchesRequest) {
	return func(r *WatcherQueryWatchesRequest) {
		r.Body = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f WatcherQueryWatches) WithPretty() func(*WatcherQueryWatchesRequest) {
	return func(r *WatcherQueryWatchesRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f WatcherQueryWatches) WithHuman() func(*WatcherQueryWatchesRequest) {
	return func(r *WatcherQueryWatchesRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f WatcherQueryWatches) WithErrorTrace() func(*WatcherQueryWatchesRequest) {
	return func(r *WatcherQueryWatchesRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f WatcherQueryWatches) WithFilterPath(v ...string) func(*WatcherQueryWatchesRequest) {
	return func(r *WatcherQueryWatchesRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f WatcherQueryWatches) WithHeader(h map[string]string) func(*WatcherQueryWatchesRequest) {
	return func(r *WatcherQueryWatchesRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f WatcherQueryWatches) WithOpaqueID(s string) func(*WatcherQueryWatchesRequest) {
	return func(r *WatcherQueryWatchesRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
