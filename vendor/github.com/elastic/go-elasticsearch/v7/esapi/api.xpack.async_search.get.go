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
	"net/http"
	"strconv"
	"strings"
	"time"
)

func newAsyncSearchGetFunc(t Transport) AsyncSearchGet {
	return func(id string, o ...func(*AsyncSearchGetRequest)) (*Response, error) {
		var r = AsyncSearchGetRequest{DocumentID: id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// AsyncSearchGet - Retrieves the results of a previously submitted async search request given its ID.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/async-search.html.
type AsyncSearchGet func(id string, o ...func(*AsyncSearchGetRequest)) (*Response, error)

// AsyncSearchGetRequest configures the Async Search Get API request.
type AsyncSearchGetRequest struct {
	DocumentID string

	KeepAlive                time.Duration
	TypedKeys                *bool
	WaitForCompletionTimeout time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r AsyncSearchGetRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_async_search") + 1 + len(r.DocumentID))
	path.WriteString("/")
	path.WriteString("_async_search")
	path.WriteString("/")
	path.WriteString(r.DocumentID)

	params = make(map[string]string)

	if r.KeepAlive != 0 {
		params["keep_alive"] = formatDuration(r.KeepAlive)
	}

	if r.TypedKeys != nil {
		params["typed_keys"] = strconv.FormatBool(*r.TypedKeys)
	}

	if r.WaitForCompletionTimeout != 0 {
		params["wait_for_completion_timeout"] = formatDuration(r.WaitForCompletionTimeout)
	}

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
func (f AsyncSearchGet) WithContext(v context.Context) func(*AsyncSearchGetRequest) {
	return func(r *AsyncSearchGetRequest) {
		r.ctx = v
	}
}

// WithKeepAlive - specify the time interval in which the results (partial or final) for this search will be available.
func (f AsyncSearchGet) WithKeepAlive(v time.Duration) func(*AsyncSearchGetRequest) {
	return func(r *AsyncSearchGetRequest) {
		r.KeepAlive = v
	}
}

// WithTypedKeys - specify whether aggregation and suggester names should be prefixed by their respective types in the response.
func (f AsyncSearchGet) WithTypedKeys(v bool) func(*AsyncSearchGetRequest) {
	return func(r *AsyncSearchGetRequest) {
		r.TypedKeys = &v
	}
}

// WithWaitForCompletionTimeout - specify the time that the request should block waiting for the final response.
func (f AsyncSearchGet) WithWaitForCompletionTimeout(v time.Duration) func(*AsyncSearchGetRequest) {
	return func(r *AsyncSearchGetRequest) {
		r.WaitForCompletionTimeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f AsyncSearchGet) WithPretty() func(*AsyncSearchGetRequest) {
	return func(r *AsyncSearchGetRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f AsyncSearchGet) WithHuman() func(*AsyncSearchGetRequest) {
	return func(r *AsyncSearchGetRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f AsyncSearchGet) WithErrorTrace() func(*AsyncSearchGetRequest) {
	return func(r *AsyncSearchGetRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f AsyncSearchGet) WithFilterPath(v ...string) func(*AsyncSearchGetRequest) {
	return func(r *AsyncSearchGetRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f AsyncSearchGet) WithHeader(h map[string]string) func(*AsyncSearchGetRequest) {
	return func(r *AsyncSearchGetRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f AsyncSearchGet) WithOpaqueID(s string) func(*AsyncSearchGetRequest) {
	return func(r *AsyncSearchGetRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
