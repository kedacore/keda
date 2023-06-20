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

func newSearchableSnapshotsStatsFunc(t Transport) SearchableSnapshotsStats {
	return func(o ...func(*SearchableSnapshotsStatsRequest)) (*Response, error) {
		var r = SearchableSnapshotsStatsRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SearchableSnapshotsStats - Retrieve shard-level statistics about searchable snapshots.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/searchable-snapshots-apis.html.
type SearchableSnapshotsStats func(o ...func(*SearchableSnapshotsStatsRequest)) (*Response, error)

// SearchableSnapshotsStatsRequest configures the Searchable Snapshots Stats API request.
type SearchableSnapshotsStatsRequest struct {
	Index []string

	Level string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r SearchableSnapshotsStatsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len(strings.Join(r.Index, ",")) + 1 + len("_searchable_snapshots") + 1 + len("stats"))
	if len(r.Index) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.Index, ","))
	}
	path.WriteString("/")
	path.WriteString("_searchable_snapshots")
	path.WriteString("/")
	path.WriteString("stats")

	params = make(map[string]string)

	if r.Level != "" {
		params["level"] = r.Level
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
func (f SearchableSnapshotsStats) WithContext(v context.Context) func(*SearchableSnapshotsStatsRequest) {
	return func(r *SearchableSnapshotsStatsRequest) {
		r.ctx = v
	}
}

// WithIndex - a list of index names.
func (f SearchableSnapshotsStats) WithIndex(v ...string) func(*SearchableSnapshotsStatsRequest) {
	return func(r *SearchableSnapshotsStatsRequest) {
		r.Index = v
	}
}

// WithLevel - return stats aggregated at cluster, index or shard level.
func (f SearchableSnapshotsStats) WithLevel(v string) func(*SearchableSnapshotsStatsRequest) {
	return func(r *SearchableSnapshotsStatsRequest) {
		r.Level = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SearchableSnapshotsStats) WithPretty() func(*SearchableSnapshotsStatsRequest) {
	return func(r *SearchableSnapshotsStatsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SearchableSnapshotsStats) WithHuman() func(*SearchableSnapshotsStatsRequest) {
	return func(r *SearchableSnapshotsStatsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SearchableSnapshotsStats) WithErrorTrace() func(*SearchableSnapshotsStatsRequest) {
	return func(r *SearchableSnapshotsStatsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SearchableSnapshotsStats) WithFilterPath(v ...string) func(*SearchableSnapshotsStatsRequest) {
	return func(r *SearchableSnapshotsStatsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SearchableSnapshotsStats) WithHeader(h map[string]string) func(*SearchableSnapshotsStatsRequest) {
	return func(r *SearchableSnapshotsStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SearchableSnapshotsStats) WithOpaqueID(s string) func(*SearchableSnapshotsStatsRequest) {
	return func(r *SearchableSnapshotsStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
