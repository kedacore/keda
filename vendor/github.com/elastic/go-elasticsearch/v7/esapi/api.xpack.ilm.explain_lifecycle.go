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

func newILMExplainLifecycleFunc(t Transport) ILMExplainLifecycle {
	return func(index string, o ...func(*ILMExplainLifecycleRequest)) (*Response, error) {
		var r = ILMExplainLifecycleRequest{Index: index}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// ILMExplainLifecycle - Retrieves information about the index's current lifecycle state, such as the currently executing phase, action, and step.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ilm-explain-lifecycle.html.
type ILMExplainLifecycle func(index string, o ...func(*ILMExplainLifecycleRequest)) (*Response, error)

// ILMExplainLifecycleRequest configures the ILM Explain Lifecycle API request.
type ILMExplainLifecycleRequest struct {
	Index string

	OnlyErrors  *bool
	OnlyManaged *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r ILMExplainLifecycleRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len(r.Index) + 1 + len("_ilm") + 1 + len("explain"))
	path.WriteString("/")
	path.WriteString(r.Index)
	path.WriteString("/")
	path.WriteString("_ilm")
	path.WriteString("/")
	path.WriteString("explain")

	params = make(map[string]string)

	if r.OnlyErrors != nil {
		params["only_errors"] = strconv.FormatBool(*r.OnlyErrors)
	}

	if r.OnlyManaged != nil {
		params["only_managed"] = strconv.FormatBool(*r.OnlyManaged)
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
func (f ILMExplainLifecycle) WithContext(v context.Context) func(*ILMExplainLifecycleRequest) {
	return func(r *ILMExplainLifecycleRequest) {
		r.ctx = v
	}
}

// WithOnlyErrors - filters the indices included in the response to ones in an ilm error state, implies only_managed.
func (f ILMExplainLifecycle) WithOnlyErrors(v bool) func(*ILMExplainLifecycleRequest) {
	return func(r *ILMExplainLifecycleRequest) {
		r.OnlyErrors = &v
	}
}

// WithOnlyManaged - filters the indices included in the response to ones managed by ilm.
func (f ILMExplainLifecycle) WithOnlyManaged(v bool) func(*ILMExplainLifecycleRequest) {
	return func(r *ILMExplainLifecycleRequest) {
		r.OnlyManaged = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f ILMExplainLifecycle) WithPretty() func(*ILMExplainLifecycleRequest) {
	return func(r *ILMExplainLifecycleRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f ILMExplainLifecycle) WithHuman() func(*ILMExplainLifecycleRequest) {
	return func(r *ILMExplainLifecycleRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f ILMExplainLifecycle) WithErrorTrace() func(*ILMExplainLifecycleRequest) {
	return func(r *ILMExplainLifecycleRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f ILMExplainLifecycle) WithFilterPath(v ...string) func(*ILMExplainLifecycleRequest) {
	return func(r *ILMExplainLifecycleRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f ILMExplainLifecycle) WithHeader(h map[string]string) func(*ILMExplainLifecycleRequest) {
	return func(r *ILMExplainLifecycleRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f ILMExplainLifecycle) WithOpaqueID(s string) func(*ILMExplainLifecycleRequest) {
	return func(r *ILMExplainLifecycleRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
