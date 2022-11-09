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
// Code generated from specification version 7.x: DO NOT EDIT

package esapi

import (
	"context"
	"net/http"
	"strings"
)

func newNodesGetMeteringInfoFunc(t Transport) NodesGetMeteringInfo {
	return func(node_id []string, o ...func(*NodesGetMeteringInfoRequest)) (*Response, error) {
		var r = NodesGetMeteringInfoRequest{NodeID: node_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// NodesGetMeteringInfo returns cluster repositories metering information.
//
// This API is experimental.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/get-repositories-metering-api.html.
//
type NodesGetMeteringInfo func(node_id []string, o ...func(*NodesGetMeteringInfoRequest)) (*Response, error)

// NodesGetMeteringInfoRequest configures the Nodes Get Metering Info API request.
//
type NodesGetMeteringInfoRequest struct {
	NodeID []string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r NodesGetMeteringInfoRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_nodes") + 1 + len(strings.Join(r.NodeID, ",")) + 1 + len("_repositories_metering"))
	path.WriteString("/")
	path.WriteString("_nodes")
	path.WriteString("/")
	path.WriteString(strings.Join(r.NodeID, ","))
	path.WriteString("/")
	path.WriteString("_repositories_metering")

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
func (f NodesGetMeteringInfo) WithContext(v context.Context) func(*NodesGetMeteringInfoRequest) {
	return func(r *NodesGetMeteringInfoRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f NodesGetMeteringInfo) WithPretty() func(*NodesGetMeteringInfoRequest) {
	return func(r *NodesGetMeteringInfoRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f NodesGetMeteringInfo) WithHuman() func(*NodesGetMeteringInfoRequest) {
	return func(r *NodesGetMeteringInfoRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f NodesGetMeteringInfo) WithErrorTrace() func(*NodesGetMeteringInfoRequest) {
	return func(r *NodesGetMeteringInfoRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f NodesGetMeteringInfo) WithFilterPath(v ...string) func(*NodesGetMeteringInfoRequest) {
	return func(r *NodesGetMeteringInfoRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f NodesGetMeteringInfo) WithHeader(h map[string]string) func(*NodesGetMeteringInfoRequest) {
	return func(r *NodesGetMeteringInfoRequest) {
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
func (f NodesGetMeteringInfo) WithOpaqueID(s string) func(*NodesGetMeteringInfoRequest) {
	return func(r *NodesGetMeteringInfoRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
