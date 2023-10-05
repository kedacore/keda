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
	"io"
	"net/http"
	"strings"
)

func newMLValidateDetectorFunc(t Transport) MLValidateDetector {
	return func(body io.Reader, o ...func(*MLValidateDetectorRequest)) (*Response, error) {
		var r = MLValidateDetectorRequest{Body: body}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLValidateDetector - Validates an anomaly detection detector.
//
// See full documentation at https://www.elastic.co/guide/en/machine-learning/current/ml-jobs.html.
type MLValidateDetector func(body io.Reader, o ...func(*MLValidateDetectorRequest)) (*Response, error)

// MLValidateDetectorRequest configures the ML Validate Detector API request.
type MLValidateDetectorRequest struct {
	Body io.Reader

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r MLValidateDetectorRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(len("/_ml/anomaly_detectors/_validate/detector"))
	path.WriteString("/_ml/anomaly_detectors/_validate/detector")

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

	if r.Body != nil && req.Header.Get(headerContentType) == "" {
		req.Header[headerContentType] = headerContentTypeJSON
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
func (f MLValidateDetector) WithContext(v context.Context) func(*MLValidateDetectorRequest) {
	return func(r *MLValidateDetectorRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLValidateDetector) WithPretty() func(*MLValidateDetectorRequest) {
	return func(r *MLValidateDetectorRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLValidateDetector) WithHuman() func(*MLValidateDetectorRequest) {
	return func(r *MLValidateDetectorRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLValidateDetector) WithErrorTrace() func(*MLValidateDetectorRequest) {
	return func(r *MLValidateDetectorRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLValidateDetector) WithFilterPath(v ...string) func(*MLValidateDetectorRequest) {
	return func(r *MLValidateDetectorRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLValidateDetector) WithHeader(h map[string]string) func(*MLValidateDetectorRequest) {
	return func(r *MLValidateDetectorRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLValidateDetector) WithOpaqueID(s string) func(*MLValidateDetectorRequest) {
	return func(r *MLValidateDetectorRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
