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
	"time"
)

func newSearchableSnapshotsMountFunc(t Transport) SearchableSnapshotsMount {
	return func(repository string, snapshot string, body io.Reader, o ...func(*SearchableSnapshotsMountRequest)) (*Response, error) {
		var r = SearchableSnapshotsMountRequest{Repository: repository, Snapshot: snapshot, Body: body}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SearchableSnapshotsMount - Mount a snapshot as a searchable index.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/searchable-snapshots-api-mount-snapshot.html.
type SearchableSnapshotsMount func(repository string, snapshot string, body io.Reader, o ...func(*SearchableSnapshotsMountRequest)) (*Response, error)

// SearchableSnapshotsMountRequest configures the Searchable Snapshots Mount API request.
type SearchableSnapshotsMountRequest struct {
	Body io.Reader

	Repository string
	Snapshot   string

	MasterTimeout     time.Duration
	Storage           string
	WaitForCompletion *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r SearchableSnapshotsMountRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_snapshot") + 1 + len(r.Repository) + 1 + len(r.Snapshot) + 1 + len("_mount"))
	path.WriteString("/")
	path.WriteString("_snapshot")
	path.WriteString("/")
	path.WriteString(r.Repository)
	path.WriteString("/")
	path.WriteString(r.Snapshot)
	path.WriteString("/")
	path.WriteString("_mount")

	params = make(map[string]string)

	if r.MasterTimeout != 0 {
		params["master_timeout"] = formatDuration(r.MasterTimeout)
	}

	if r.Storage != "" {
		params["storage"] = r.Storage
	}

	if r.WaitForCompletion != nil {
		params["wait_for_completion"] = strconv.FormatBool(*r.WaitForCompletion)
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
func (f SearchableSnapshotsMount) WithContext(v context.Context) func(*SearchableSnapshotsMountRequest) {
	return func(r *SearchableSnapshotsMountRequest) {
		r.ctx = v
	}
}

// WithMasterTimeout - explicit operation timeout for connection to master node.
func (f SearchableSnapshotsMount) WithMasterTimeout(v time.Duration) func(*SearchableSnapshotsMountRequest) {
	return func(r *SearchableSnapshotsMountRequest) {
		r.MasterTimeout = v
	}
}

// WithStorage - selects the kind of local storage used to accelerate searches. experimental, and defaults to `full_copy`.
func (f SearchableSnapshotsMount) WithStorage(v string) func(*SearchableSnapshotsMountRequest) {
	return func(r *SearchableSnapshotsMountRequest) {
		r.Storage = v
	}
}

// WithWaitForCompletion - should this request wait until the operation has completed before returning.
func (f SearchableSnapshotsMount) WithWaitForCompletion(v bool) func(*SearchableSnapshotsMountRequest) {
	return func(r *SearchableSnapshotsMountRequest) {
		r.WaitForCompletion = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SearchableSnapshotsMount) WithPretty() func(*SearchableSnapshotsMountRequest) {
	return func(r *SearchableSnapshotsMountRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SearchableSnapshotsMount) WithHuman() func(*SearchableSnapshotsMountRequest) {
	return func(r *SearchableSnapshotsMountRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SearchableSnapshotsMount) WithErrorTrace() func(*SearchableSnapshotsMountRequest) {
	return func(r *SearchableSnapshotsMountRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SearchableSnapshotsMount) WithFilterPath(v ...string) func(*SearchableSnapshotsMountRequest) {
	return func(r *SearchableSnapshotsMountRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SearchableSnapshotsMount) WithHeader(h map[string]string) func(*SearchableSnapshotsMountRequest) {
	return func(r *SearchableSnapshotsMountRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SearchableSnapshotsMount) WithOpaqueID(s string) func(*SearchableSnapshotsMountRequest) {
	return func(r *SearchableSnapshotsMountRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
