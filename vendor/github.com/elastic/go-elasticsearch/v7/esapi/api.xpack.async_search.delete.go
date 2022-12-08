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
// Code generated from specification version 7.17.1: DO NOT EDIT

package esapi

import (
	"context"
	"net/http"
	"strings"
)

func newAsyncSearchDeleteFunc(t Transport) AsyncSearchDelete {
	return func(id string, o ...func(*AsyncSearchDeleteRequest)) (*Response, error) {
		var r = AsyncSearchDeleteRequest{DocumentID: id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// AsyncSearchDelete - Deletes an async search by ID. If the search is still running, the search request will be cancelled. Otherwise, the saved search results are deleted.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/async-search.html.
//
type AsyncSearchDelete func(id string, o ...func(*AsyncSearchDeleteRequest)) (*Response, error)

// AsyncSearchDeleteRequest configures the Async Search Delete API request.
//
type AsyncSearchDeleteRequest struct {
	DocumentID string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r AsyncSearchDeleteRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "DELETE"

	path.Grow(1 + len("_async_search") + 1 + len(r.DocumentID))
	path.WriteString("/")
	path.WriteString("_async_search")
	path.WriteString("/")
	path.WriteString(r.DocumentID)

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

	req, err := newRequest(method, path.String(), nil)
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
//
func (f AsyncSearchDelete) WithContext(v context.Context) func(*AsyncSearchDeleteRequest) {
	return func(r *AsyncSearchDeleteRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f AsyncSearchDelete) WithPretty() func(*AsyncSearchDeleteRequest) {
	return func(r *AsyncSearchDeleteRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f AsyncSearchDelete) WithHuman() func(*AsyncSearchDeleteRequest) {
	return func(r *AsyncSearchDeleteRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f AsyncSearchDelete) WithErrorTrace() func(*AsyncSearchDeleteRequest) {
	return func(r *AsyncSearchDeleteRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f AsyncSearchDelete) WithFilterPath(v ...string) func(*AsyncSearchDeleteRequest) {
	return func(r *AsyncSearchDeleteRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f AsyncSearchDelete) WithHeader(h map[string]string) func(*AsyncSearchDeleteRequest) {
	return func(r *AsyncSearchDeleteRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
//
func (f AsyncSearchDelete) WithOpaqueID(s string) func(*AsyncSearchDeleteRequest) {
	return func(r *AsyncSearchDeleteRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
