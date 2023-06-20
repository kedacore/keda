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

func newMLGetCategoriesFunc(t Transport) MLGetCategories {
	return func(job_id string, o ...func(*MLGetCategoriesRequest)) (*Response, error) {
		var r = MLGetCategoriesRequest{JobID: job_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLGetCategories - Retrieves anomaly detection job results for one or more categories.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-get-category.html.
type MLGetCategories func(job_id string, o ...func(*MLGetCategoriesRequest)) (*Response, error)

// MLGetCategoriesRequest configures the ML Get Categories API request.
type MLGetCategoriesRequest struct {
	Body io.Reader

	CategoryID *int
	JobID      string

	From                *int
	PartitionFieldValue string
	Size                *int

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r MLGetCategoriesRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_ml") + 1 + len("anomaly_detectors") + 1 + len(r.JobID) + 1 + len("results") + 1 + len("categories"))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("anomaly_detectors")
	path.WriteString("/")
	path.WriteString(r.JobID)
	path.WriteString("/")
	path.WriteString("results")
	path.WriteString("/")
	path.WriteString("categories")
	if r.CategoryID != nil {
		value := strconv.FormatInt(int64(*r.CategoryID), 10)
		path.Grow(1 + len(value))
		path.WriteString("/")
		path.WriteString(value)
	}

	params = make(map[string]string)

	if r.From != nil {
		params["from"] = strconv.FormatInt(int64(*r.From), 10)
	}

	if r.PartitionFieldValue != "" {
		params["partition_field_value"] = r.PartitionFieldValue
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
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
func (f MLGetCategories) WithContext(v context.Context) func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		r.ctx = v
	}
}

// WithBody - Category selection details if not provided in URI.
func (f MLGetCategories) WithBody(v io.Reader) func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		r.Body = v
	}
}

// WithCategoryID - the identifier of the category definition of interest.
func (f MLGetCategories) WithCategoryID(v int) func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		r.CategoryID = &v
	}
}

// WithFrom - skips a number of categories.
func (f MLGetCategories) WithFrom(v int) func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		r.From = &v
	}
}

// WithPartitionFieldValue - specifies the partition to retrieve categories for. this is optional, and should never be used for jobs where per-partition categorization is disabled..
func (f MLGetCategories) WithPartitionFieldValue(v string) func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		r.PartitionFieldValue = v
	}
}

// WithSize - specifies a max number of categories to get.
func (f MLGetCategories) WithSize(v int) func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		r.Size = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLGetCategories) WithPretty() func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLGetCategories) WithHuman() func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLGetCategories) WithErrorTrace() func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLGetCategories) WithFilterPath(v ...string) func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLGetCategories) WithHeader(h map[string]string) func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLGetCategories) WithOpaqueID(s string) func(*MLGetCategoriesRequest) {
	return func(r *MLGetCategoriesRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
