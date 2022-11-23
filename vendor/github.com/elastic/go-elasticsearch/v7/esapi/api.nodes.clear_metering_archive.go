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
	"strconv"
	"strings"
)

func newNodesClearMeteringArchiveFunc(t Transport) NodesClearMeteringArchive {
	return func(max_archive_version *int, node_id []string, o ...func(*NodesClearMeteringArchiveRequest)) (*Response, error) {
		var r = NodesClearMeteringArchiveRequest{NodeID: node_id, MaxArchiveVersion: max_archive_version}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// NodesClearMeteringArchive removes the archived repositories metering information present in the cluster.
//
// This API is experimental.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/clear-repositories-metering-archive-api.html.
//
type NodesClearMeteringArchive func(max_archive_version *int, node_id []string, o ...func(*NodesClearMeteringArchiveRequest)) (*Response, error)

// NodesClearMeteringArchiveRequest configures the Nodes Clear Metering Archive API request.
//
type NodesClearMeteringArchiveRequest struct {
	MaxArchiveVersion *int
	NodeID            []string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r NodesClearMeteringArchiveRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "DELETE"

	path.Grow(1 + len("_nodes") + 1 + len(strings.Join(r.NodeID, ",")) + 1 + len("_repositories_metering") + 1 + len(strconv.Itoa(*r.MaxArchiveVersion)))
	path.WriteString("/")
	path.WriteString("_nodes")
	path.WriteString("/")
	path.WriteString(strings.Join(r.NodeID, ","))
	path.WriteString("/")
	path.WriteString("_repositories_metering")
	path.WriteString("/")
	path.WriteString(strconv.Itoa(*r.MaxArchiveVersion))

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
func (f NodesClearMeteringArchive) WithContext(v context.Context) func(*NodesClearMeteringArchiveRequest) {
	return func(r *NodesClearMeteringArchiveRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f NodesClearMeteringArchive) WithPretty() func(*NodesClearMeteringArchiveRequest) {
	return func(r *NodesClearMeteringArchiveRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f NodesClearMeteringArchive) WithHuman() func(*NodesClearMeteringArchiveRequest) {
	return func(r *NodesClearMeteringArchiveRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f NodesClearMeteringArchive) WithErrorTrace() func(*NodesClearMeteringArchiveRequest) {
	return func(r *NodesClearMeteringArchiveRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f NodesClearMeteringArchive) WithFilterPath(v ...string) func(*NodesClearMeteringArchiveRequest) {
	return func(r *NodesClearMeteringArchiveRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f NodesClearMeteringArchive) WithHeader(h map[string]string) func(*NodesClearMeteringArchiveRequest) {
	return func(r *NodesClearMeteringArchiveRequest) {
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
func (f NodesClearMeteringArchive) WithOpaqueID(s string) func(*NodesClearMeteringArchiveRequest) {
	return func(r *NodesClearMeteringArchiveRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
