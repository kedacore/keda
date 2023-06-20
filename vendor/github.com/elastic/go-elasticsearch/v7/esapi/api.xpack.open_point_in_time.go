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
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func newOpenPointInTimeFunc(t Transport) OpenPointInTime {
	return func(index []string, keep_alive string, o ...func(*OpenPointInTimeRequest)) (*Response, error) {
		var r = OpenPointInTimeRequest{Index: index, KeepAlive: keep_alive}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// OpenPointInTime - Open a point in time that can be used in subsequent searches
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/point-in-time-api.html.
type OpenPointInTime func(index []string, keep_alive string, o ...func(*OpenPointInTimeRequest)) (*Response, error)

// OpenPointInTimeRequest configures the Open Point In Time API request.
type OpenPointInTimeRequest struct {
	Index []string

	ExpandWildcards   string
	IgnoreUnavailable *bool
	KeepAlive         string
	Preference        string
	Routing           string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r OpenPointInTimeRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	if len(r.Index) == 0 {
		return nil, errors.New("index is required and cannot be nil or empty")
	}

	path.Grow(1 + len(strings.Join(r.Index, ",")) + 1 + len("_pit"))
	path.WriteString("/")
	path.WriteString(strings.Join(r.Index, ","))
	path.WriteString("/")
	path.WriteString("_pit")

	params = make(map[string]string)

	if r.ExpandWildcards != "" {
		params["expand_wildcards"] = r.ExpandWildcards
	}

	if r.IgnoreUnavailable != nil {
		params["ignore_unavailable"] = strconv.FormatBool(*r.IgnoreUnavailable)
	}

	if r.KeepAlive != "" {
		params["keep_alive"] = r.KeepAlive
	}

	if r.Preference != "" {
		params["preference"] = r.Preference
	}

	if r.Routing != "" {
		params["routing"] = r.Routing
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
func (f OpenPointInTime) WithContext(v context.Context) func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		r.ctx = v
	}
}

// WithExpandWildcards - whether to expand wildcard expression to concrete indices that are open, closed or both..
func (f OpenPointInTime) WithExpandWildcards(v string) func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		r.ExpandWildcards = v
	}
}

// WithIgnoreUnavailable - whether specified concrete indices should be ignored when unavailable (missing or closed).
func (f OpenPointInTime) WithIgnoreUnavailable(v bool) func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		r.IgnoreUnavailable = &v
	}
}

// WithKeepAlive - specific the time to live for the point in time.
func (f OpenPointInTime) WithKeepAlive(v string) func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		r.KeepAlive = v
	}
}

// WithPreference - specify the node or shard the operation should be performed on (default: random).
func (f OpenPointInTime) WithPreference(v string) func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		r.Preference = v
	}
}

// WithRouting - specific routing value.
func (f OpenPointInTime) WithRouting(v string) func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		r.Routing = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f OpenPointInTime) WithPretty() func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f OpenPointInTime) WithHuman() func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f OpenPointInTime) WithErrorTrace() func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f OpenPointInTime) WithFilterPath(v ...string) func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f OpenPointInTime) WithHeader(h map[string]string) func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f OpenPointInTime) WithOpaqueID(s string) func(*OpenPointInTimeRequest) {
	return func(r *OpenPointInTimeRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
