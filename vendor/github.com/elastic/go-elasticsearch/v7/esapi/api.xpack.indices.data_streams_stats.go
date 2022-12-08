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

func newIndicesDataStreamsStatsFunc(t Transport) IndicesDataStreamsStats {
	return func(o ...func(*IndicesDataStreamsStatsRequest)) (*Response, error) {
		var r = IndicesDataStreamsStatsRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// IndicesDataStreamsStats - Provides statistics on operations happening in a data stream.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/data-streams.html.
//
type IndicesDataStreamsStats func(o ...func(*IndicesDataStreamsStatsRequest)) (*Response, error)

// IndicesDataStreamsStatsRequest configures the Indices Data Streams Stats API request.
//
type IndicesDataStreamsStatsRequest struct {
	Name []string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r IndicesDataStreamsStatsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_data_stream") + 1 + len(strings.Join(r.Name, ",")) + 1 + len("_stats"))
	path.WriteString("/")
	path.WriteString("_data_stream")
	if len(r.Name) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.Name, ","))
	}
	path.WriteString("/")
	path.WriteString("_stats")

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
func (f IndicesDataStreamsStats) WithContext(v context.Context) func(*IndicesDataStreamsStatsRequest) {
	return func(r *IndicesDataStreamsStatsRequest) {
		r.ctx = v
	}
}

// WithName - a list of data stream names; use _all to perform the operation on all data streams.
//
func (f IndicesDataStreamsStats) WithName(v ...string) func(*IndicesDataStreamsStatsRequest) {
	return func(r *IndicesDataStreamsStatsRequest) {
		r.Name = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f IndicesDataStreamsStats) WithPretty() func(*IndicesDataStreamsStatsRequest) {
	return func(r *IndicesDataStreamsStatsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f IndicesDataStreamsStats) WithHuman() func(*IndicesDataStreamsStatsRequest) {
	return func(r *IndicesDataStreamsStatsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f IndicesDataStreamsStats) WithErrorTrace() func(*IndicesDataStreamsStatsRequest) {
	return func(r *IndicesDataStreamsStatsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f IndicesDataStreamsStats) WithFilterPath(v ...string) func(*IndicesDataStreamsStatsRequest) {
	return func(r *IndicesDataStreamsStatsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f IndicesDataStreamsStats) WithHeader(h map[string]string) func(*IndicesDataStreamsStatsRequest) {
	return func(r *IndicesDataStreamsStatsRequest) {
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
func (f IndicesDataStreamsStats) WithOpaqueID(s string) func(*IndicesDataStreamsStatsRequest) {
	return func(r *IndicesDataStreamsStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
