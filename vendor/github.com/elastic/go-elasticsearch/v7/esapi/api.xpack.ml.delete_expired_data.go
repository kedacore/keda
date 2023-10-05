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
	"time"
)

func newMLDeleteExpiredDataFunc(t Transport) MLDeleteExpiredData {
	return func(o ...func(*MLDeleteExpiredDataRequest)) (*Response, error) {
		var r = MLDeleteExpiredDataRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLDeleteExpiredData - Deletes expired and unused machine learning data.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-delete-expired-data.html.
type MLDeleteExpiredData func(o ...func(*MLDeleteExpiredDataRequest)) (*Response, error)

// MLDeleteExpiredDataRequest configures the ML Delete Expired Data API request.
type MLDeleteExpiredDataRequest struct {
	Body io.Reader

	JobID string

	RequestsPerSecond *int
	Timeout           time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r MLDeleteExpiredDataRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "DELETE"

	path.Grow(1 + len("_ml") + 1 + len("_delete_expired_data") + 1 + len(r.JobID))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("_delete_expired_data")
	if r.JobID != "" {
		path.WriteString("/")
		path.WriteString(r.JobID)
	}

	params = make(map[string]string)

	if r.RequestsPerSecond != nil {
		params["requests_per_second"] = strconv.FormatInt(int64(*r.RequestsPerSecond), 10)
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
func (f MLDeleteExpiredData) WithContext(v context.Context) func(*MLDeleteExpiredDataRequest) {
	return func(r *MLDeleteExpiredDataRequest) {
		r.ctx = v
	}
}

// WithBody - deleting expired data parameters.
func (f MLDeleteExpiredData) WithBody(v io.Reader) func(*MLDeleteExpiredDataRequest) {
	return func(r *MLDeleteExpiredDataRequest) {
		r.Body = v
	}
}

// WithJobID - the ID of the job(s) to perform expired data hygiene for.
func (f MLDeleteExpiredData) WithJobID(v string) func(*MLDeleteExpiredDataRequest) {
	return func(r *MLDeleteExpiredDataRequest) {
		r.JobID = v
	}
}

// WithRequestsPerSecond - the desired requests per second for the deletion processes..
func (f MLDeleteExpiredData) WithRequestsPerSecond(v int) func(*MLDeleteExpiredDataRequest) {
	return func(r *MLDeleteExpiredDataRequest) {
		r.RequestsPerSecond = &v
	}
}

// WithTimeout - how long can the underlying delete processes run until they are canceled.
func (f MLDeleteExpiredData) WithTimeout(v time.Duration) func(*MLDeleteExpiredDataRequest) {
	return func(r *MLDeleteExpiredDataRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLDeleteExpiredData) WithPretty() func(*MLDeleteExpiredDataRequest) {
	return func(r *MLDeleteExpiredDataRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLDeleteExpiredData) WithHuman() func(*MLDeleteExpiredDataRequest) {
	return func(r *MLDeleteExpiredDataRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLDeleteExpiredData) WithErrorTrace() func(*MLDeleteExpiredDataRequest) {
	return func(r *MLDeleteExpiredDataRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLDeleteExpiredData) WithFilterPath(v ...string) func(*MLDeleteExpiredDataRequest) {
	return func(r *MLDeleteExpiredDataRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLDeleteExpiredData) WithHeader(h map[string]string) func(*MLDeleteExpiredDataRequest) {
	return func(r *MLDeleteExpiredDataRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLDeleteExpiredData) WithOpaqueID(s string) func(*MLDeleteExpiredDataRequest) {
	return func(r *MLDeleteExpiredDataRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
