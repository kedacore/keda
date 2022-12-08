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
	"time"
)

func newMLStopDatafeedFunc(t Transport) MLStopDatafeed {
	return func(datafeed_id string, o ...func(*MLStopDatafeedRequest)) (*Response, error) {
		var r = MLStopDatafeedRequest{DatafeedID: datafeed_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLStopDatafeed - Stops one or more datafeeds.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-stop-datafeed.html.
//
type MLStopDatafeed func(datafeed_id string, o ...func(*MLStopDatafeedRequest)) (*Response, error)

// MLStopDatafeedRequest configures the ML Stop Datafeed API request.
//
type MLStopDatafeedRequest struct {
	Body io.Reader

	DatafeedID string

	AllowNoDatafeeds *bool
	AllowNoMatch     *bool
	Force            *bool
	Timeout          time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r MLStopDatafeedRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_ml") + 1 + len("datafeeds") + 1 + len(r.DatafeedID) + 1 + len("_stop"))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("datafeeds")
	path.WriteString("/")
	path.WriteString(r.DatafeedID)
	path.WriteString("/")
	path.WriteString("_stop")

	params = make(map[string]string)

	if r.AllowNoDatafeeds != nil {
		params["allow_no_datafeeds"] = strconv.FormatBool(*r.AllowNoDatafeeds)
	}

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.Force != nil {
		params["force"] = strconv.FormatBool(*r.Force)
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
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
func (f MLStopDatafeed) WithContext(v context.Context) func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
		r.ctx = v
	}
}

// WithBody - The URL params optionally sent in the body.
//
func (f MLStopDatafeed) WithBody(v io.Reader) func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
		r.Body = v
	}
}

// WithAllowNoDatafeeds - whether to ignore if a wildcard expression matches no datafeeds. (this includes `_all` string or when no datafeeds have been specified).
//
func (f MLStopDatafeed) WithAllowNoDatafeeds(v bool) func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
		r.AllowNoDatafeeds = &v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no datafeeds. (this includes `_all` string or when no datafeeds have been specified).
//
func (f MLStopDatafeed) WithAllowNoMatch(v bool) func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
		r.AllowNoMatch = &v
	}
}

// WithForce - true if the datafeed should be forcefully stopped..
//
func (f MLStopDatafeed) WithForce(v bool) func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
		r.Force = &v
	}
}

// WithTimeout - controls the time to wait until a datafeed has stopped. default to 20 seconds.
//
func (f MLStopDatafeed) WithTimeout(v time.Duration) func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f MLStopDatafeed) WithPretty() func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f MLStopDatafeed) WithHuman() func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f MLStopDatafeed) WithErrorTrace() func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f MLStopDatafeed) WithFilterPath(v ...string) func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f MLStopDatafeed) WithHeader(h map[string]string) func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
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
func (f MLStopDatafeed) WithOpaqueID(s string) func(*MLStopDatafeedRequest) {
	return func(r *MLStopDatafeedRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
