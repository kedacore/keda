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
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func newNodesClearRepositoriesMeteringArchiveFunc(t Transport) NodesClearRepositoriesMeteringArchive {
	return func(node_id []string, max_archive_version *int, o ...func(*NodesClearRepositoriesMeteringArchiveRequest)) (*Response, error) {
		var r = NodesClearRepositoriesMeteringArchiveRequest{NodeID: node_id, MaxArchiveVersion: max_archive_version}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// NodesClearRepositoriesMeteringArchive removes the archived repositories metering information present in the cluster.
//
// This API is experimental.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/clear-repositories-metering-archive-api.html.
//
type NodesClearRepositoriesMeteringArchive func(node_id []string, max_archive_version *int, o ...func(*NodesClearRepositoriesMeteringArchiveRequest)) (*Response, error)

// NodesClearRepositoriesMeteringArchiveRequest configures the Nodes Clear Repositories Metering Archive API request.
//
type NodesClearRepositoriesMeteringArchiveRequest struct {
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
func (r NodesClearRepositoriesMeteringArchiveRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "DELETE"

	if len(r.NodeID) == 0 {
		return nil, errors.New("node_id is required and cannot be nil or empty")
	}
	if r.MaxArchiveVersion == nil {
		return nil, errors.New("max_archive_version is required and cannot be nil")
	}

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
func (f NodesClearRepositoriesMeteringArchive) WithContext(v context.Context) func(*NodesClearRepositoriesMeteringArchiveRequest) {
	return func(r *NodesClearRepositoriesMeteringArchiveRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f NodesClearRepositoriesMeteringArchive) WithPretty() func(*NodesClearRepositoriesMeteringArchiveRequest) {
	return func(r *NodesClearRepositoriesMeteringArchiveRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f NodesClearRepositoriesMeteringArchive) WithHuman() func(*NodesClearRepositoriesMeteringArchiveRequest) {
	return func(r *NodesClearRepositoriesMeteringArchiveRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f NodesClearRepositoriesMeteringArchive) WithErrorTrace() func(*NodesClearRepositoriesMeteringArchiveRequest) {
	return func(r *NodesClearRepositoriesMeteringArchiveRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f NodesClearRepositoriesMeteringArchive) WithFilterPath(v ...string) func(*NodesClearRepositoriesMeteringArchiveRequest) {
	return func(r *NodesClearRepositoriesMeteringArchiveRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f NodesClearRepositoriesMeteringArchive) WithHeader(h map[string]string) func(*NodesClearRepositoriesMeteringArchiveRequest) {
	return func(r *NodesClearRepositoriesMeteringArchiveRequest) {
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
func (f NodesClearRepositoriesMeteringArchive) WithOpaqueID(s string) func(*NodesClearRepositoriesMeteringArchiveRequest) {
	return func(r *NodesClearRepositoriesMeteringArchiveRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
