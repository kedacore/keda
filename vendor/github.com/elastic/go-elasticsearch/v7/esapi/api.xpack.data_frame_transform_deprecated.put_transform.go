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

func newDataFrameTransformDeprecatedPutTransformFunc(t Transport) DataFrameTransformDeprecatedPutTransform {
	return func(body io.Reader, transform_id string, o ...func(*DataFrameTransformDeprecatedPutTransformRequest)) (*Response, error) {
		var r = DataFrameTransformDeprecatedPutTransformRequest{Body: body, TransformID: transform_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// DataFrameTransformDeprecatedPutTransform - Instantiates a transform.
//
// This API is beta.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/put-transform.html.
type DataFrameTransformDeprecatedPutTransform func(body io.Reader, transform_id string, o ...func(*DataFrameTransformDeprecatedPutTransformRequest)) (*Response, error)

// DataFrameTransformDeprecatedPutTransformRequest configures the Data Frame Transform Deprecated Put Transform API request.
type DataFrameTransformDeprecatedPutTransformRequest struct {
	Body io.Reader

	TransformID string

	DeferValidation *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r DataFrameTransformDeprecatedPutTransformRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "PUT"

	path.Grow(1 + len("_data_frame") + 1 + len("transforms") + 1 + len(r.TransformID))
	path.WriteString("/")
	path.WriteString("_data_frame")
	path.WriteString("/")
	path.WriteString("transforms")
	path.WriteString("/")
	path.WriteString(r.TransformID)

	params = make(map[string]string)

	if r.DeferValidation != nil {
		params["defer_validation"] = strconv.FormatBool(*r.DeferValidation)
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
func (f DataFrameTransformDeprecatedPutTransform) WithContext(v context.Context) func(*DataFrameTransformDeprecatedPutTransformRequest) {
	return func(r *DataFrameTransformDeprecatedPutTransformRequest) {
		r.ctx = v
	}
}

// WithDeferValidation - if validations should be deferred until transform starts, defaults to false..
func (f DataFrameTransformDeprecatedPutTransform) WithDeferValidation(v bool) func(*DataFrameTransformDeprecatedPutTransformRequest) {
	return func(r *DataFrameTransformDeprecatedPutTransformRequest) {
		r.DeferValidation = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f DataFrameTransformDeprecatedPutTransform) WithPretty() func(*DataFrameTransformDeprecatedPutTransformRequest) {
	return func(r *DataFrameTransformDeprecatedPutTransformRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f DataFrameTransformDeprecatedPutTransform) WithHuman() func(*DataFrameTransformDeprecatedPutTransformRequest) {
	return func(r *DataFrameTransformDeprecatedPutTransformRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f DataFrameTransformDeprecatedPutTransform) WithErrorTrace() func(*DataFrameTransformDeprecatedPutTransformRequest) {
	return func(r *DataFrameTransformDeprecatedPutTransformRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f DataFrameTransformDeprecatedPutTransform) WithFilterPath(v ...string) func(*DataFrameTransformDeprecatedPutTransformRequest) {
	return func(r *DataFrameTransformDeprecatedPutTransformRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f DataFrameTransformDeprecatedPutTransform) WithHeader(h map[string]string) func(*DataFrameTransformDeprecatedPutTransformRequest) {
	return func(r *DataFrameTransformDeprecatedPutTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f DataFrameTransformDeprecatedPutTransform) WithOpaqueID(s string) func(*DataFrameTransformDeprecatedPutTransformRequest) {
	return func(r *DataFrameTransformDeprecatedPutTransformRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
