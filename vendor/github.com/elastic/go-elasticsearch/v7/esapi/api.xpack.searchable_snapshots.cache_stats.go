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
	"strings"
)

func newSearchableSnapshotsCacheStatsFunc(t Transport) SearchableSnapshotsCacheStats {
	return func(o ...func(*SearchableSnapshotsCacheStatsRequest)) (*Response, error) {
		var r = SearchableSnapshotsCacheStatsRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SearchableSnapshotsCacheStats - Retrieve node-level cache statistics about searchable snapshots.
//
// This API is experimental.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/searchable-snapshots-apis.html.
type SearchableSnapshotsCacheStats func(o ...func(*SearchableSnapshotsCacheStatsRequest)) (*Response, error)

// SearchableSnapshotsCacheStatsRequest configures the Searchable Snapshots Cache Stats API request.
type SearchableSnapshotsCacheStatsRequest struct {
	NodeID []string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r SearchableSnapshotsCacheStatsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_searchable_snapshots") + 1 + len(strings.Join(r.NodeID, ",")) + 1 + len("cache") + 1 + len("stats"))
	path.WriteString("/")
	path.WriteString("_searchable_snapshots")
	if len(r.NodeID) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.NodeID, ","))
	}
	path.WriteString("/")
	path.WriteString("cache")
	path.WriteString("/")
	path.WriteString("stats")

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
func (f SearchableSnapshotsCacheStats) WithContext(v context.Context) func(*SearchableSnapshotsCacheStatsRequest) {
	return func(r *SearchableSnapshotsCacheStatsRequest) {
		r.ctx = v
	}
}

// WithNodeID - a list of node ids or names to limit the returned information; use `_local` to return information from the node you're connecting to, leave empty to get information from all nodes.
func (f SearchableSnapshotsCacheStats) WithNodeID(v ...string) func(*SearchableSnapshotsCacheStatsRequest) {
	return func(r *SearchableSnapshotsCacheStatsRequest) {
		r.NodeID = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SearchableSnapshotsCacheStats) WithPretty() func(*SearchableSnapshotsCacheStatsRequest) {
	return func(r *SearchableSnapshotsCacheStatsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SearchableSnapshotsCacheStats) WithHuman() func(*SearchableSnapshotsCacheStatsRequest) {
	return func(r *SearchableSnapshotsCacheStatsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SearchableSnapshotsCacheStats) WithErrorTrace() func(*SearchableSnapshotsCacheStatsRequest) {
	return func(r *SearchableSnapshotsCacheStatsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SearchableSnapshotsCacheStats) WithFilterPath(v ...string) func(*SearchableSnapshotsCacheStatsRequest) {
	return func(r *SearchableSnapshotsCacheStatsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SearchableSnapshotsCacheStats) WithHeader(h map[string]string) func(*SearchableSnapshotsCacheStatsRequest) {
	return func(r *SearchableSnapshotsCacheStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SearchableSnapshotsCacheStats) WithOpaqueID(s string) func(*SearchableSnapshotsCacheStatsRequest) {
	return func(r *SearchableSnapshotsCacheStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
