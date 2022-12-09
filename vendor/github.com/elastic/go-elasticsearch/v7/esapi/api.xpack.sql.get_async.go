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
	"time"
)

func newSQLGetAsyncFunc(t Transport) SQLGetAsync {
	return func(id string, o ...func(*SQLGetAsyncRequest)) (*Response, error) {
		var r = SQLGetAsyncRequest{DocumentID: id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SQLGetAsync - Returns the current status and available results for an async SQL search or stored synchronous SQL search
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/get-async-sql-search-api.html.
//
type SQLGetAsync func(id string, o ...func(*SQLGetAsyncRequest)) (*Response, error)

// SQLGetAsyncRequest configures the SQL Get Async API request.
//
type SQLGetAsyncRequest struct {
	DocumentID string

	Delimiter                string
	Format                   string
	KeepAlive                time.Duration
	WaitForCompletionTimeout time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r SQLGetAsyncRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_sql") + 1 + len("async") + 1 + len(r.DocumentID))
	path.WriteString("/")
	path.WriteString("_sql")
	path.WriteString("/")
	path.WriteString("async")
	path.WriteString("/")
	path.WriteString(r.DocumentID)

	params = make(map[string]string)

	if r.Delimiter != "" {
		params["delimiter"] = r.Delimiter
	}

	if r.Format != "" {
		params["format"] = r.Format
	}

	if r.KeepAlive != 0 {
		params["keep_alive"] = formatDuration(r.KeepAlive)
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
//
func (f SQLGetAsync) WithContext(v context.Context) func(*SQLGetAsyncRequest) {
	return func(r *SQLGetAsyncRequest) {
		r.ctx = v
	}
}

// WithDelimiter - separator for csv results.
//
func (f SQLGetAsync) WithDelimiter(v string) func(*SQLGetAsyncRequest) {
	return func(r *SQLGetAsyncRequest) {
		r.Delimiter = v
	}
}

// WithFormat - short version of the accept header, e.g. json, yaml.
//
func (f SQLGetAsync) WithFormat(v string) func(*SQLGetAsyncRequest) {
	return func(r *SQLGetAsyncRequest) {
		r.Format = v
	}
}

// WithKeepAlive - retention period for the search and its results.
//
func (f SQLGetAsync) WithKeepAlive(v time.Duration) func(*SQLGetAsyncRequest) {
	return func(r *SQLGetAsyncRequest) {
		r.KeepAlive = v
	}
}

// WithWaitForCompletionTimeout - duration to wait for complete results.
//
func (f SQLGetAsync) WithWaitForCompletionTimeout(v time.Duration) func(*SQLGetAsyncRequest) {
	return func(r *SQLGetAsyncRequest) {
		r.WaitForCompletionTimeout = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f SQLGetAsync) WithPretty() func(*SQLGetAsyncRequest) {
	return func(r *SQLGetAsyncRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f SQLGetAsync) WithHuman() func(*SQLGetAsyncRequest) {
	return func(r *SQLGetAsyncRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f SQLGetAsync) WithErrorTrace() func(*SQLGetAsyncRequest) {
	return func(r *SQLGetAsyncRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f SQLGetAsync) WithFilterPath(v ...string) func(*SQLGetAsyncRequest) {
	return func(r *SQLGetAsyncRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f SQLGetAsync) WithHeader(h map[string]string) func(*SQLGetAsyncRequest) {
	return func(r *SQLGetAsyncRequest) {
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
func (f SQLGetAsync) WithOpaqueID(s string) func(*SQLGetAsyncRequest) {
	return func(r *SQLGetAsyncRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
