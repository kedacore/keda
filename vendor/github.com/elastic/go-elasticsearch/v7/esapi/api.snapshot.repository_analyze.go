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
	"net/http"
	"strconv"
	"strings"
	"time"
)

func newSnapshotRepositoryAnalyzeFunc(t Transport) SnapshotRepositoryAnalyze {
	return func(repository string, o ...func(*SnapshotRepositoryAnalyzeRequest)) (*Response, error) {
		var r = SnapshotRepositoryAnalyzeRequest{Repository: repository}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SnapshotRepositoryAnalyze analyzes a repository for correctness and performance
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/modules-snapshots.html.
//
type SnapshotRepositoryAnalyze func(repository string, o ...func(*SnapshotRepositoryAnalyzeRequest)) (*Response, error)

// SnapshotRepositoryAnalyzeRequest configures the Snapshot Repository Analyze API request.
//
type SnapshotRepositoryAnalyzeRequest struct {
	Repository string

	BlobCount             *int
	Concurrency           *int
	Detailed              *bool
	EarlyReadNodeCount    *int
	MaxBlobSize           string
	MaxTotalDataSize      string
	RareActionProbability *int
	RarelyAbortWrites     *bool
	ReadNodeCount         *int
	Seed                  *int
	Timeout               time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r SnapshotRepositoryAnalyzeRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_snapshot") + 1 + len(r.Repository) + 1 + len("_analyze"))
	path.WriteString("/")
	path.WriteString("_snapshot")
	path.WriteString("/")
	path.WriteString(r.Repository)
	path.WriteString("/")
	path.WriteString("_analyze")

	params = make(map[string]string)

	if r.BlobCount != nil {
		params["blob_count"] = strconv.FormatInt(int64(*r.BlobCount), 10)
	}

	if r.Concurrency != nil {
		params["concurrency"] = strconv.FormatInt(int64(*r.Concurrency), 10)
	}

	if r.Detailed != nil {
		params["detailed"] = strconv.FormatBool(*r.Detailed)
	}

	if r.EarlyReadNodeCount != nil {
		params["early_read_node_count"] = strconv.FormatInt(int64(*r.EarlyReadNodeCount), 10)
	}

	if r.MaxBlobSize != "" {
		params["max_blob_size"] = r.MaxBlobSize
	}

	if r.MaxTotalDataSize != "" {
		params["max_total_data_size"] = r.MaxTotalDataSize
	}

	if r.RareActionProbability != nil {
		params["rare_action_probability"] = strconv.FormatInt(int64(*r.RareActionProbability), 10)
	}

	if r.RarelyAbortWrites != nil {
		params["rarely_abort_writes"] = strconv.FormatBool(*r.RarelyAbortWrites)
	}

	if r.ReadNodeCount != nil {
		params["read_node_count"] = strconv.FormatInt(int64(*r.ReadNodeCount), 10)
	}

	if r.Seed != nil {
		params["seed"] = strconv.FormatInt(int64(*r.Seed), 10)
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
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
//
func (f SnapshotRepositoryAnalyze) WithContext(v context.Context) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.ctx = v
	}
}

// WithBlobCount - number of blobs to create during the test. defaults to 100..
//
func (f SnapshotRepositoryAnalyze) WithBlobCount(v int) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.BlobCount = &v
	}
}

// WithConcurrency - number of operations to run concurrently during the test. defaults to 10..
//
func (f SnapshotRepositoryAnalyze) WithConcurrency(v int) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.Concurrency = &v
	}
}

// WithDetailed - whether to return detailed results or a summary. defaults to 'false' so that only the summary is returned..
//
func (f SnapshotRepositoryAnalyze) WithDetailed(v bool) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.Detailed = &v
	}
}

// WithEarlyReadNodeCount - number of nodes on which to perform an early read on a blob, i.e. before writing has completed. early reads are rare actions so the 'rare_action_probability' parameter is also relevant. defaults to 2..
//
func (f SnapshotRepositoryAnalyze) WithEarlyReadNodeCount(v int) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.EarlyReadNodeCount = &v
	}
}

// WithMaxBlobSize - maximum size of a blob to create during the test, e.g '1gb' or '100mb'. defaults to '10mb'..
//
func (f SnapshotRepositoryAnalyze) WithMaxBlobSize(v string) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.MaxBlobSize = v
	}
}

// WithMaxTotalDataSize - maximum total size of all blobs to create during the test, e.g '1tb' or '100gb'. defaults to '1gb'..
//
func (f SnapshotRepositoryAnalyze) WithMaxTotalDataSize(v string) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.MaxTotalDataSize = v
	}
}

// WithRareActionProbability - probability of taking a rare action such as an early read or an overwrite. defaults to 0.02..
//
func (f SnapshotRepositoryAnalyze) WithRareActionProbability(v int) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.RareActionProbability = &v
	}
}

// WithRarelyAbortWrites - whether to rarely abort writes before they complete. defaults to 'true'..
//
func (f SnapshotRepositoryAnalyze) WithRarelyAbortWrites(v bool) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.RarelyAbortWrites = &v
	}
}

// WithReadNodeCount - number of nodes on which to read a blob after writing. defaults to 10..
//
func (f SnapshotRepositoryAnalyze) WithReadNodeCount(v int) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.ReadNodeCount = &v
	}
}

// WithSeed - seed for the random number generator used to create the test workload. defaults to a random value..
//
func (f SnapshotRepositoryAnalyze) WithSeed(v int) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.Seed = &v
	}
}

// WithTimeout - explicit operation timeout. defaults to '30s'..
//
func (f SnapshotRepositoryAnalyze) WithTimeout(v time.Duration) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.Timeout = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f SnapshotRepositoryAnalyze) WithPretty() func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f SnapshotRepositoryAnalyze) WithHuman() func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f SnapshotRepositoryAnalyze) WithErrorTrace() func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f SnapshotRepositoryAnalyze) WithFilterPath(v ...string) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f SnapshotRepositoryAnalyze) WithHeader(h map[string]string) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
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
func (f SnapshotRepositoryAnalyze) WithOpaqueID(s string) func(*SnapshotRepositoryAnalyzeRequest) {
	return func(r *SnapshotRepositoryAnalyzeRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
