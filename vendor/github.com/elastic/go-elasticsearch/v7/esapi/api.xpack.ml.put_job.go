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
	"strconv"
	"strings"
)

func newMLPutJobFunc(t Transport) MLPutJob {
	return func(job_id string, body io.Reader, o ...func(*MLPutJobRequest)) (*Response, error) {
		var r = MLPutJobRequest{JobID: job_id, Body: body}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLPutJob - Instantiates an anomaly detection job.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-put-job.html.
type MLPutJob func(job_id string, body io.Reader, o ...func(*MLPutJobRequest)) (*Response, error)

// MLPutJobRequest configures the ML Put Job API request.
type MLPutJobRequest struct {
	Body io.Reader

	JobID string

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
func (r MLPutJobRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "PUT"

	path.Grow(1 + len("_ml") + 1 + len("anomaly_detectors") + 1 + len(r.JobID))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("anomaly_detectors")
	path.WriteString("/")
	path.WriteString(r.JobID)

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
func (f MLPutJob) WithContext(v context.Context) func(*MLPutJobRequest) {
	return func(r *MLPutJobRequest) {
		r.ctx = v
	}
}

// WithAllowNoIndices - ignore if the source indices expressions resolves to no concrete indices (default: true). only set if datafeed_config is provided..
func (f MLPutJob) WithAllowNoIndices(v bool) func(*MLPutJobRequest) {
	return func(r *MLPutJobRequest) {
		r.AllowNoIndices = &v
	}
}

// WithExpandWildcards - whether source index expressions should get expanded to open or closed indices (default: open). only set if datafeed_config is provided..
func (f MLPutJob) WithExpandWildcards(v string) func(*MLPutJobRequest) {
	return func(r *MLPutJobRequest) {
		r.ExpandWildcards = v
	}
}

// WithIgnoreThrottled - ignore indices that are marked as throttled (default: true). only set if datafeed_config is provided..
func (f MLPutJob) WithIgnoreThrottled(v bool) func(*MLPutJobRequest) {
	return func(r *MLPutJobRequest) {
		r.IgnoreThrottled = &v
	}
}

// WithIgnoreUnavailable - ignore unavailable indexes (default: false). only set if datafeed_config is provided..
func (f MLPutJob) WithIgnoreUnavailable(v bool) func(*MLPutJobRequest) {
	return func(r *MLPutJobRequest) {
		r.IgnoreUnavailable = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLPutJob) WithPretty() func(*MLPutJobRequest) {
	return func(r *MLPutJobRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLPutJob) WithHuman() func(*MLPutJobRequest) {
	return func(r *MLPutJobRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLPutJob) WithErrorTrace() func(*MLPutJobRequest) {
	return func(r *MLPutJobRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLPutJob) WithFilterPath(v ...string) func(*MLPutJobRequest) {
	return func(r *MLPutJobRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLPutJob) WithHeader(h map[string]string) func(*MLPutJobRequest) {
	return func(r *MLPutJobRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLPutJob) WithOpaqueID(s string) func(*MLPutJobRequest) {
	return func(r *MLPutJobRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
