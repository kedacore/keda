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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func newSnapshotGetFunc(t Transport) SnapshotGet {
	return func(repository string, snapshot []string, o ...func(*SnapshotGetRequest)) (*Response, error) {
		var r = SnapshotGetRequest{Repository: repository, Snapshot: snapshot}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SnapshotGet returns information about a snapshot.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/modules-snapshots.html.
type SnapshotGet func(repository string, snapshot []string, o ...func(*SnapshotGetRequest)) (*Response, error)

// SnapshotGetRequest configures the Snapshot Get API request.
type SnapshotGetRequest struct {
	Repository string
	Snapshot   []string

	After             string
	FromSortValue     string
	IgnoreUnavailable *bool
	IncludeRepository *bool
	IndexDetails      *bool
	MasterTimeout     time.Duration
	Offset            interface{}
	Order             string
	Size              interface{}
	SlmPolicyFilter   string
	Sort              string
	Verbose           *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r SnapshotGetRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	if len(r.Snapshot) == 0 {
		return nil, errors.New("snapshot is required and cannot be nil or empty")
	}

	path.Grow(1 + len("_snapshot") + 1 + len(r.Repository) + 1 + len(strings.Join(r.Snapshot, ",")))
	path.WriteString("/")
	path.WriteString("_snapshot")
	path.WriteString("/")
	path.WriteString(r.Repository)
	path.WriteString("/")
	path.WriteString(strings.Join(r.Snapshot, ","))

	params = make(map[string]string)

	if r.After != "" {
		params["after"] = r.After
	}

	if r.FromSortValue != "" {
		params["from_sort_value"] = r.FromSortValue
	}

	if r.IgnoreUnavailable != nil {
		params["ignore_unavailable"] = strconv.FormatBool(*r.IgnoreUnavailable)
	}

	if r.IncludeRepository != nil {
		params["include_repository"] = strconv.FormatBool(*r.IncludeRepository)
	}

	if r.IndexDetails != nil {
		params["index_details"] = strconv.FormatBool(*r.IndexDetails)
	}

	if r.MasterTimeout != 0 {
		params["master_timeout"] = formatDuration(r.MasterTimeout)
	}

	if r.Offset != nil {
		params["offset"] = fmt.Sprintf("%v", r.Offset)
	}

	if r.Order != "" {
		params["order"] = r.Order
	}

	if r.Size != nil {
		params["size"] = fmt.Sprintf("%v", r.Size)
	}

	if r.SlmPolicyFilter != "" {
		params["slm_policy_filter"] = r.SlmPolicyFilter
	}

	if r.Sort != "" {
		params["sort"] = r.Sort
	}

	if r.Verbose != nil {
		params["verbose"] = strconv.FormatBool(*r.Verbose)
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
func (f SnapshotGet) WithContext(v context.Context) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.ctx = v
	}
}

// WithAfter - offset identifier to start pagination from as returned by the 'next' field in the response body..
func (f SnapshotGet) WithAfter(v string) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.After = v
	}
}

// WithFromSortValue - value of the current sort column at which to start retrieval..
func (f SnapshotGet) WithFromSortValue(v string) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.FromSortValue = v
	}
}

// WithIgnoreUnavailable - whether to ignore unavailable snapshots, defaults to false which means a snapshotmissingexception is thrown.
func (f SnapshotGet) WithIgnoreUnavailable(v bool) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.IgnoreUnavailable = &v
	}
}

// WithIncludeRepository - whether to include the repository name in the snapshot info. defaults to true..
func (f SnapshotGet) WithIncludeRepository(v bool) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.IncludeRepository = &v
	}
}

// WithIndexDetails - whether to include details of each index in the snapshot, if those details are available. defaults to false..
func (f SnapshotGet) WithIndexDetails(v bool) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.IndexDetails = &v
	}
}

// WithMasterTimeout - explicit operation timeout for connection to master node.
func (f SnapshotGet) WithMasterTimeout(v time.Duration) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.MasterTimeout = v
	}
}

// WithOffset - numeric offset to start pagination based on the snapshots matching the request. defaults to 0.
func (f SnapshotGet) WithOffset(v interface{}) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.Offset = v
	}
}

// WithOrder - sort order.
func (f SnapshotGet) WithOrder(v string) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.Order = v
	}
}

// WithSize - maximum number of snapshots to return. defaults to 0 which means return all that match without limit..
func (f SnapshotGet) WithSize(v interface{}) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.Size = v
	}
}

// WithSlmPolicyFilter - filter snapshots by a list of slm policy names that snapshots belong to. accepts wildcards. use the special pattern '_none' to match snapshots without an slm policy.
func (f SnapshotGet) WithSlmPolicyFilter(v string) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.SlmPolicyFilter = v
	}
}

// WithSort - allows setting a sort order for the result. defaults to start_time.
func (f SnapshotGet) WithSort(v string) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.Sort = v
	}
}

// WithVerbose - whether to show verbose snapshot info or only show the basic info found in the repository index blob.
func (f SnapshotGet) WithVerbose(v bool) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.Verbose = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SnapshotGet) WithPretty() func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SnapshotGet) WithHuman() func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SnapshotGet) WithErrorTrace() func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SnapshotGet) WithFilterPath(v ...string) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SnapshotGet) WithHeader(h map[string]string) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SnapshotGet) WithOpaqueID(s string) func(*SnapshotGetRequest) {
	return func(r *SnapshotGetRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
