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
	"strconv"
	"strings"
)

func newMLGetDatafeedsFunc(t Transport) MLGetDatafeeds {
	return func(o ...func(*MLGetDatafeedsRequest)) (*Response, error) {
		var r = MLGetDatafeedsRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLGetDatafeeds - Retrieves configuration information for datafeeds.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-get-datafeed.html.
//
type MLGetDatafeeds func(o ...func(*MLGetDatafeedsRequest)) (*Response, error)

// MLGetDatafeedsRequest configures the ML Get Datafeeds API request.
//
type MLGetDatafeedsRequest struct {
	DatafeedID string

	AllowNoDatafeeds *bool
	AllowNoMatch     *bool
	ExcludeGenerated *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r MLGetDatafeedsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_ml") + 1 + len("datafeeds") + 1 + len(r.DatafeedID))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("datafeeds")
	if r.DatafeedID != "" {
		path.WriteString("/")
		path.WriteString(r.DatafeedID)
	}

	params = make(map[string]string)

	if r.AllowNoDatafeeds != nil {
		params["allow_no_datafeeds"] = strconv.FormatBool(*r.AllowNoDatafeeds)
	}

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.ExcludeGenerated != nil {
		params["exclude_generated"] = strconv.FormatBool(*r.ExcludeGenerated)
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
func (f MLGetDatafeeds) WithContext(v context.Context) func(*MLGetDatafeedsRequest) {
	return func(r *MLGetDatafeedsRequest) {
		r.ctx = v
	}
}

// WithDatafeedID - the ID of the datafeeds to fetch.
//
func (f MLGetDatafeeds) WithDatafeedID(v string) func(*MLGetDatafeedsRequest) {
	return func(r *MLGetDatafeedsRequest) {
		r.DatafeedID = v
	}
}

// WithAllowNoDatafeeds - whether to ignore if a wildcard expression matches no datafeeds. (this includes `_all` string or when no datafeeds have been specified).
//
func (f MLGetDatafeeds) WithAllowNoDatafeeds(v bool) func(*MLGetDatafeedsRequest) {
	return func(r *MLGetDatafeedsRequest) {
		r.AllowNoDatafeeds = &v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no datafeeds. (this includes `_all` string or when no datafeeds have been specified).
//
func (f MLGetDatafeeds) WithAllowNoMatch(v bool) func(*MLGetDatafeedsRequest) {
	return func(r *MLGetDatafeedsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithExcludeGenerated - omits fields that are illegal to set on datafeed put.
//
func (f MLGetDatafeeds) WithExcludeGenerated(v bool) func(*MLGetDatafeedsRequest) {
	return func(r *MLGetDatafeedsRequest) {
		r.ExcludeGenerated = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f MLGetDatafeeds) WithPretty() func(*MLGetDatafeedsRequest) {
	return func(r *MLGetDatafeedsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f MLGetDatafeeds) WithHuman() func(*MLGetDatafeedsRequest) {
	return func(r *MLGetDatafeedsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f MLGetDatafeeds) WithErrorTrace() func(*MLGetDatafeedsRequest) {
	return func(r *MLGetDatafeedsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f MLGetDatafeeds) WithFilterPath(v ...string) func(*MLGetDatafeedsRequest) {
	return func(r *MLGetDatafeedsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f MLGetDatafeeds) WithHeader(h map[string]string) func(*MLGetDatafeedsRequest) {
	return func(r *MLGetDatafeedsRequest) {
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
func (f MLGetDatafeeds) WithOpaqueID(s string) func(*MLGetDatafeedsRequest) {
	return func(r *MLGetDatafeedsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
