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
// Code generated from specification version 7.10.0: DO NOT EDIT

package esapi

import (
	"context"
	"net/http"
	"strings"
)

func newAutoscalingGetAutoscalingDecisionFunc(t Transport) AutoscalingGetAutoscalingDecision {
	return func(o ...func(*AutoscalingGetAutoscalingDecisionRequest)) (*Response, error) {
		var r = AutoscalingGetAutoscalingDecisionRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// AutoscalingGetAutoscalingDecision - Gets the current autoscaling decision based on the configured autoscaling policy, indicating whether or not autoscaling is needed.
//
// This API is experimental.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/autoscaling-get-autoscaling-decision.html.
//
type AutoscalingGetAutoscalingDecision func(o ...func(*AutoscalingGetAutoscalingDecisionRequest)) (*Response, error)

// AutoscalingGetAutoscalingDecisionRequest configures the Autoscaling Get Autoscaling Decision API request.
//
type AutoscalingGetAutoscalingDecisionRequest struct {
	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r AutoscalingGetAutoscalingDecisionRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(len("/_autoscaling/decision"))
	path.WriteString("/_autoscaling/decision")

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
func (f AutoscalingGetAutoscalingDecision) WithContext(v context.Context) func(*AutoscalingGetAutoscalingDecisionRequest) {
	return func(r *AutoscalingGetAutoscalingDecisionRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f AutoscalingGetAutoscalingDecision) WithPretty() func(*AutoscalingGetAutoscalingDecisionRequest) {
	return func(r *AutoscalingGetAutoscalingDecisionRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f AutoscalingGetAutoscalingDecision) WithHuman() func(*AutoscalingGetAutoscalingDecisionRequest) {
	return func(r *AutoscalingGetAutoscalingDecisionRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f AutoscalingGetAutoscalingDecision) WithErrorTrace() func(*AutoscalingGetAutoscalingDecisionRequest) {
	return func(r *AutoscalingGetAutoscalingDecisionRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f AutoscalingGetAutoscalingDecision) WithFilterPath(v ...string) func(*AutoscalingGetAutoscalingDecisionRequest) {
	return func(r *AutoscalingGetAutoscalingDecisionRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f AutoscalingGetAutoscalingDecision) WithHeader(h map[string]string) func(*AutoscalingGetAutoscalingDecisionRequest) {
	return func(r *AutoscalingGetAutoscalingDecisionRequest) {
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
func (f AutoscalingGetAutoscalingDecision) WithOpaqueID(s string) func(*AutoscalingGetAutoscalingDecisionRequest) {
	return func(r *AutoscalingGetAutoscalingDecisionRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
