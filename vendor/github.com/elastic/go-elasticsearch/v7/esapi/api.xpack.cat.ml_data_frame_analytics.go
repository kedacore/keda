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
)

func newCatMLDataFrameAnalyticsFunc(t Transport) CatMLDataFrameAnalytics {
	return func(o ...func(*CatMLDataFrameAnalyticsRequest)) (*Response, error) {
		var r = CatMLDataFrameAnalyticsRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// CatMLDataFrameAnalytics - Gets configuration and usage information about data frame analytics jobs.
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/current/cat-dfanalytics.html.
type CatMLDataFrameAnalytics func(o ...func(*CatMLDataFrameAnalyticsRequest)) (*Response, error)

// CatMLDataFrameAnalyticsRequest configures the CatML Data Frame Analytics API request.
type CatMLDataFrameAnalyticsRequest struct {
	DocumentID string

	AllowNoMatch *bool
	Bytes        string
	Format       string
	H            []string
	Help         *bool
	S            []string
	Time         string
	V            *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r CatMLDataFrameAnalyticsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_cat") + 1 + len("ml") + 1 + len("data_frame") + 1 + len("analytics") + 1 + len(r.DocumentID))
	path.WriteString("/")
	path.WriteString("_cat")
	path.WriteString("/")
	path.WriteString("ml")
	path.WriteString("/")
	path.WriteString("data_frame")
	path.WriteString("/")
	path.WriteString("analytics")
	if r.DocumentID != "" {
		path.WriteString("/")
		path.WriteString(r.DocumentID)
	}

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.Bytes != "" {
		params["bytes"] = r.Bytes
	}

	if r.Format != "" {
		params["format"] = r.Format
	}

	if len(r.H) > 0 {
		params["h"] = strings.Join(r.H, ",")
	}

	if r.Help != nil {
		params["help"] = strconv.FormatBool(*r.Help)
	}

	if len(r.S) > 0 {
		params["s"] = strings.Join(r.S, ",")
	}

	if r.Time != "" {
		params["time"] = r.Time
	}

	if r.V != nil {
		params["v"] = strconv.FormatBool(*r.V)
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
func (f CatMLDataFrameAnalytics) WithContext(v context.Context) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.ctx = v
	}
}

// WithDocumentID - the ID of the data frame analytics to fetch.
func (f CatMLDataFrameAnalytics) WithDocumentID(v string) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.DocumentID = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no configs. (this includes `_all` string or when no configs have been specified).
func (f CatMLDataFrameAnalytics) WithAllowNoMatch(v bool) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithBytes - the unit in which to display byte values.
func (f CatMLDataFrameAnalytics) WithBytes(v string) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.Bytes = v
	}
}

// WithFormat - a short version of the accept header, e.g. json, yaml.
func (f CatMLDataFrameAnalytics) WithFormat(v string) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.Format = v
	}
}

// WithH - comma-separated list of column names to display.
func (f CatMLDataFrameAnalytics) WithH(v ...string) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.H = v
	}
}

// WithHelp - return help information.
func (f CatMLDataFrameAnalytics) WithHelp(v bool) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.Help = &v
	}
}

// WithS - comma-separated list of column names or column aliases to sort by.
func (f CatMLDataFrameAnalytics) WithS(v ...string) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.S = v
	}
}

// WithTime - the unit in which to display time values.
func (f CatMLDataFrameAnalytics) WithTime(v string) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.Time = v
	}
}

// WithV - verbose mode. display column headers.
func (f CatMLDataFrameAnalytics) WithV(v bool) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.V = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f CatMLDataFrameAnalytics) WithPretty() func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f CatMLDataFrameAnalytics) WithHuman() func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f CatMLDataFrameAnalytics) WithErrorTrace() func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f CatMLDataFrameAnalytics) WithFilterPath(v ...string) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f CatMLDataFrameAnalytics) WithHeader(h map[string]string) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f CatMLDataFrameAnalytics) WithOpaqueID(s string) func(*CatMLDataFrameAnalyticsRequest) {
	return func(r *CatMLDataFrameAnalyticsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
