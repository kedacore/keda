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
	"io"
	"net/http"
	"strconv"
	"strings"
)

func newMLPutDatafeedFunc(t Transport) MLPutDatafeed {
	return func(body io.Reader, datafeed_id string, o ...func(*MLPutDatafeedRequest)) (*Response, error) {
		var r = MLPutDatafeedRequest{Body: body, DatafeedID: datafeed_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLPutDatafeed - Instantiates a datafeed.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-put-datafeed.html.
//
type MLPutDatafeed func(body io.Reader, datafeed_id string, o ...func(*MLPutDatafeedRequest)) (*Response, error)

// MLPutDatafeedRequest configures the ML Put Datafeed API request.
//
type MLPutDatafeedRequest struct {
	Body io.Reader

	DatafeedID string

	AllowNoIndices    *bool
	ExpandWildcards   string
	IgnoreThrottled   *bool
	IgnoreUnavailable *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r MLPutDatafeedRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "PUT"

	path.Grow(1 + len("_ml") + 1 + len("datafeeds") + 1 + len(r.DatafeedID))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("datafeeds")
	path.WriteString("/")
	path.WriteString(r.DatafeedID)

	params = make(map[string]string)

	if r.AllowNoIndices != nil {
		params["allow_no_indices"] = strconv.FormatBool(*r.AllowNoIndices)
	}

	if r.ExpandWildcards != "" {
		params["expand_wildcards"] = r.ExpandWildcards
	}

	if r.IgnoreThrottled != nil {
		params["ignore_throttled"] = strconv.FormatBool(*r.IgnoreThrottled)
	}

	if r.IgnoreUnavailable != nil {
		params["ignore_unavailable"] = strconv.FormatBool(*r.IgnoreUnavailable)
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

	if r.Body != nil {
		req.Header[headerContentType] = headerContentTypeJSON
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
func (f MLPutDatafeed) WithContext(v context.Context) func(*MLPutDatafeedRequest) {
	return func(r *MLPutDatafeedRequest) {
		r.ctx = v
	}
}

// WithAllowNoIndices - ignore if the source indices expressions resolves to no concrete indices (default: true).
//
func (f MLPutDatafeed) WithAllowNoIndices(v bool) func(*MLPutDatafeedRequest) {
	return func(r *MLPutDatafeedRequest) {
		r.AllowNoIndices = &v
	}
}

// WithExpandWildcards - whether source index expressions should get expanded to open or closed indices (default: open).
//
func (f MLPutDatafeed) WithExpandWildcards(v string) func(*MLPutDatafeedRequest) {
	return func(r *MLPutDatafeedRequest) {
		r.ExpandWildcards = v
	}
}

// WithIgnoreThrottled - ignore indices that are marked as throttled (default: true).
//
func (f MLPutDatafeed) WithIgnoreThrottled(v bool) func(*MLPutDatafeedRequest) {
	return func(r *MLPutDatafeedRequest) {
		r.IgnoreThrottled = &v
	}
}

// WithIgnoreUnavailable - ignore unavailable indexes (default: false).
//
func (f MLPutDatafeed) WithIgnoreUnavailable(v bool) func(*MLPutDatafeedRequest) {
	return func(r *MLPutDatafeedRequest) {
		r.IgnoreUnavailable = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f MLPutDatafeed) WithPretty() func(*MLPutDatafeedRequest) {
	return func(r *MLPutDatafeedRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f MLPutDatafeed) WithHuman() func(*MLPutDatafeedRequest) {
	return func(r *MLPutDatafeedRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f MLPutDatafeed) WithErrorTrace() func(*MLPutDatafeedRequest) {
	return func(r *MLPutDatafeedRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f MLPutDatafeed) WithFilterPath(v ...string) func(*MLPutDatafeedRequest) {
	return func(r *MLPutDatafeedRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f MLPutDatafeed) WithHeader(h map[string]string) func(*MLPutDatafeedRequest) {
	return func(r *MLPutDatafeedRequest) {
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
func (f MLPutDatafeed) WithOpaqueID(s string) func(*MLPutDatafeedRequest) {
	return func(r *MLPutDatafeedRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
