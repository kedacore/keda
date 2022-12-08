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
	"strings"
)

func newMLPreviewDataFrameAnalyticsFunc(t Transport) MLPreviewDataFrameAnalytics {
	return func(o ...func(*MLPreviewDataFrameAnalyticsRequest)) (*Response, error) {
		var r = MLPreviewDataFrameAnalyticsRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLPreviewDataFrameAnalytics - Previews that will be analyzed given a data frame analytics config.
//
// See full documentation at http://www.elastic.co/guide/en/elasticsearch/reference/current/preview-dfanalytics.html.
//
type MLPreviewDataFrameAnalytics func(o ...func(*MLPreviewDataFrameAnalyticsRequest)) (*Response, error)

// MLPreviewDataFrameAnalyticsRequest configures the ML Preview Data Frame Analytics API request.
//
type MLPreviewDataFrameAnalyticsRequest struct {
	DocumentID string

	Body io.Reader

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r MLPreviewDataFrameAnalyticsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_ml") + 1 + len("data_frame") + 1 + len("analytics") + 1 + len(r.DocumentID) + 1 + len("_preview"))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("data_frame")
	path.WriteString("/")
	path.WriteString("analytics")
	if r.DocumentID != "" {
		path.WriteString("/")
		path.WriteString(r.DocumentID)
	}
	path.WriteString("/")
	path.WriteString("_preview")

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
func (f MLPreviewDataFrameAnalytics) WithContext(v context.Context) func(*MLPreviewDataFrameAnalyticsRequest) {
	return func(r *MLPreviewDataFrameAnalyticsRequest) {
		r.ctx = v
	}
}

// WithBody - The data frame analytics config to preview.
//
func (f MLPreviewDataFrameAnalytics) WithBody(v io.Reader) func(*MLPreviewDataFrameAnalyticsRequest) {
	return func(r *MLPreviewDataFrameAnalyticsRequest) {
		r.Body = v
	}
}

// WithDocumentID - the ID of the data frame analytics to preview.
//
func (f MLPreviewDataFrameAnalytics) WithDocumentID(v string) func(*MLPreviewDataFrameAnalyticsRequest) {
	return func(r *MLPreviewDataFrameAnalyticsRequest) {
		r.DocumentID = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f MLPreviewDataFrameAnalytics) WithPretty() func(*MLPreviewDataFrameAnalyticsRequest) {
	return func(r *MLPreviewDataFrameAnalyticsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f MLPreviewDataFrameAnalytics) WithHuman() func(*MLPreviewDataFrameAnalyticsRequest) {
	return func(r *MLPreviewDataFrameAnalyticsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f MLPreviewDataFrameAnalytics) WithErrorTrace() func(*MLPreviewDataFrameAnalyticsRequest) {
	return func(r *MLPreviewDataFrameAnalyticsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f MLPreviewDataFrameAnalytics) WithFilterPath(v ...string) func(*MLPreviewDataFrameAnalyticsRequest) {
	return func(r *MLPreviewDataFrameAnalyticsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f MLPreviewDataFrameAnalytics) WithHeader(h map[string]string) func(*MLPreviewDataFrameAnalyticsRequest) {
	return func(r *MLPreviewDataFrameAnalyticsRequest) {
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
func (f MLPreviewDataFrameAnalytics) WithOpaqueID(s string) func(*MLPreviewDataFrameAnalyticsRequest) {
	return func(r *MLPreviewDataFrameAnalyticsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
