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
	"time"
)

func newFleetGlobalCheckpointsFunc(t Transport) FleetGlobalCheckpoints {
	return func(index string, o ...func(*FleetGlobalCheckpointsRequest)) (*Response, error) {
		var r = FleetGlobalCheckpointsRequest{Index: index}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// FleetGlobalCheckpoints returns the current global checkpoints for an index. This API is design for internal use by the fleet server project.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/get-global-checkpoints.html.
type FleetGlobalCheckpoints func(index string, o ...func(*FleetGlobalCheckpointsRequest)) (*Response, error)

// FleetGlobalCheckpointsRequest configures the Fleet Global Checkpoints API request.
type FleetGlobalCheckpointsRequest struct {
	Index string

	Checkpoints    []string
	Timeout        time.Duration
	WaitForAdvance *bool
	WaitForIndex   *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r FleetGlobalCheckpointsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len(r.Index) + 1 + len("_fleet") + 1 + len("global_checkpoints"))
	path.WriteString("/")
	path.WriteString(r.Index)
	path.WriteString("/")
	path.WriteString("_fleet")
	path.WriteString("/")
	path.WriteString("global_checkpoints")

	params = make(map[string]string)

	if len(r.Checkpoints) > 0 {
		params["checkpoints"] = strings.Join(r.Checkpoints, ",")
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
	}

	if r.WaitForAdvance != nil {
		params["wait_for_advance"] = strconv.FormatBool(*r.WaitForAdvance)
	}

	if r.WaitForIndex != nil {
		params["wait_for_index"] = strconv.FormatBool(*r.WaitForIndex)
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
func (f FleetGlobalCheckpoints) WithContext(v context.Context) func(*FleetGlobalCheckpointsRequest) {
	return func(r *FleetGlobalCheckpointsRequest) {
		r.ctx = v
	}
}

// WithCheckpoints - comma separated list of checkpoints.
func (f FleetGlobalCheckpoints) WithCheckpoints(v ...string) func(*FleetGlobalCheckpointsRequest) {
	return func(r *FleetGlobalCheckpointsRequest) {
		r.Checkpoints = v
	}
}

// WithTimeout - timeout to wait for global checkpoint to advance.
func (f FleetGlobalCheckpoints) WithTimeout(v time.Duration) func(*FleetGlobalCheckpointsRequest) {
	return func(r *FleetGlobalCheckpointsRequest) {
		r.Timeout = v
	}
}

// WithWaitForAdvance - whether to wait for the global checkpoint to advance past the specified current checkpoints.
func (f FleetGlobalCheckpoints) WithWaitForAdvance(v bool) func(*FleetGlobalCheckpointsRequest) {
	return func(r *FleetGlobalCheckpointsRequest) {
		r.WaitForAdvance = &v
	}
}

// WithWaitForIndex - whether to wait for the target index to exist and all primary shards be active.
func (f FleetGlobalCheckpoints) WithWaitForIndex(v bool) func(*FleetGlobalCheckpointsRequest) {
	return func(r *FleetGlobalCheckpointsRequest) {
		r.WaitForIndex = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f FleetGlobalCheckpoints) WithPretty() func(*FleetGlobalCheckpointsRequest) {
	return func(r *FleetGlobalCheckpointsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f FleetGlobalCheckpoints) WithHuman() func(*FleetGlobalCheckpointsRequest) {
	return func(r *FleetGlobalCheckpointsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f FleetGlobalCheckpoints) WithErrorTrace() func(*FleetGlobalCheckpointsRequest) {
	return func(r *FleetGlobalCheckpointsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f FleetGlobalCheckpoints) WithFilterPath(v ...string) func(*FleetGlobalCheckpointsRequest) {
	return func(r *FleetGlobalCheckpointsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f FleetGlobalCheckpoints) WithHeader(h map[string]string) func(*FleetGlobalCheckpointsRequest) {
	return func(r *FleetGlobalCheckpointsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f FleetGlobalCheckpoints) WithOpaqueID(s string) func(*FleetGlobalCheckpointsRequest) {
	return func(r *FleetGlobalCheckpointsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
