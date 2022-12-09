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
	"strconv"
	"strings"
	"time"
)

func newFleetSearchFunc(t Transport) FleetSearch {
	return func(index string, o ...func(*FleetSearchRequest)) (*Response, error) {
		var r = FleetSearchRequest{Index: index}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// FleetSearch search API where the search will only be executed after specified checkpoints are available due to a refresh. This API is designed for internal use by the fleet server project.
//
// This API is experimental.
//
type FleetSearch func(index string, o ...func(*FleetSearchRequest)) (*Response, error)

// FleetSearchRequest configures the Fleet Search API request.
//
type FleetSearchRequest struct {
	Index string

	Body io.Reader

	AllowPartialSearchResults *bool
	WaitForCheckpoints        []string
	WaitForCheckpointsTimeout time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r FleetSearchRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len(r.Index) + 1 + len("_fleet") + 1 + len("_fleet_search"))
	path.WriteString("/")
	path.WriteString(r.Index)
	path.WriteString("/")
	path.WriteString("_fleet")
	path.WriteString("/")
	path.WriteString("_fleet_search")

	params = make(map[string]string)

	if r.AllowPartialSearchResults != nil {
		params["allow_partial_search_results"] = strconv.FormatBool(*r.AllowPartialSearchResults)
	}

	if len(r.WaitForCheckpoints) > 0 {
		params["wait_for_checkpoints"] = strings.Join(r.WaitForCheckpoints, ",")
	}

	if r.WaitForCheckpointsTimeout != 0 {
		params["wait_for_checkpoints_timeout"] = formatDuration(r.WaitForCheckpointsTimeout)
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
func (f FleetSearch) WithContext(v context.Context) func(*FleetSearchRequest) {
	return func(r *FleetSearchRequest) {
		r.ctx = v
	}
}

// WithBody - The search definition using the Query DSL.
//
func (f FleetSearch) WithBody(v io.Reader) func(*FleetSearchRequest) {
	return func(r *FleetSearchRequest) {
		r.Body = v
	}
}

// WithAllowPartialSearchResults - indicate if an error should be returned if there is a partial search failure or timeout.
//
func (f FleetSearch) WithAllowPartialSearchResults(v bool) func(*FleetSearchRequest) {
	return func(r *FleetSearchRequest) {
		r.AllowPartialSearchResults = &v
	}
}

// WithWaitForCheckpoints - comma separated list of checkpoints, one per shard.
//
func (f FleetSearch) WithWaitForCheckpoints(v ...string) func(*FleetSearchRequest) {
	return func(r *FleetSearchRequest) {
		r.WaitForCheckpoints = v
	}
}

// WithWaitForCheckpointsTimeout - explicit wait_for_checkpoints timeout.
//
func (f FleetSearch) WithWaitForCheckpointsTimeout(v time.Duration) func(*FleetSearchRequest) {
	return func(r *FleetSearchRequest) {
		r.WaitForCheckpointsTimeout = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f FleetSearch) WithPretty() func(*FleetSearchRequest) {
	return func(r *FleetSearchRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f FleetSearch) WithHuman() func(*FleetSearchRequest) {
	return func(r *FleetSearchRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f FleetSearch) WithErrorTrace() func(*FleetSearchRequest) {
	return func(r *FleetSearchRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f FleetSearch) WithFilterPath(v ...string) func(*FleetSearchRequest) {
	return func(r *FleetSearchRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f FleetSearch) WithHeader(h map[string]string) func(*FleetSearchRequest) {
	return func(r *FleetSearchRequest) {
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
func (f FleetSearch) WithOpaqueID(s string) func(*FleetSearchRequest) {
	return func(r *FleetSearchRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
