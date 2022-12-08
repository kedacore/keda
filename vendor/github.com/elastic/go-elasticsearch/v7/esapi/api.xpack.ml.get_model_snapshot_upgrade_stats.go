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
)

func newMLGetModelSnapshotUpgradeStatsFunc(t Transport) MLGetModelSnapshotUpgradeStats {
	return func(snapshot_id string, job_id string, o ...func(*MLGetModelSnapshotUpgradeStatsRequest)) (*Response, error) {
		var r = MLGetModelSnapshotUpgradeStatsRequest{SnapshotID: snapshot_id, JobID: job_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLGetModelSnapshotUpgradeStats - Gets stats for anomaly detection job model snapshot upgrades that are in progress.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-get-job-model-snapshot-upgrade-stats.html.
//
type MLGetModelSnapshotUpgradeStats func(snapshot_id string, job_id string, o ...func(*MLGetModelSnapshotUpgradeStatsRequest)) (*Response, error)

// MLGetModelSnapshotUpgradeStatsRequest configures the ML Get Model Snapshot Upgrade Stats API request.
//
type MLGetModelSnapshotUpgradeStatsRequest struct {
	JobID      string
	SnapshotID string

	AllowNoMatch *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r MLGetModelSnapshotUpgradeStatsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_ml") + 1 + len("anomaly_detectors") + 1 + len(r.JobID) + 1 + len("model_snapshots") + 1 + len(r.SnapshotID) + 1 + len("_upgrade") + 1 + len("_stats"))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("anomaly_detectors")
	path.WriteString("/")
	path.WriteString(r.JobID)
	path.WriteString("/")
	path.WriteString("model_snapshots")
	path.WriteString("/")
	path.WriteString(r.SnapshotID)
	path.WriteString("/")
	path.WriteString("_upgrade")
	path.WriteString("/")
	path.WriteString("_stats")

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
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
func (f MLGetModelSnapshotUpgradeStats) WithContext(v context.Context) func(*MLGetModelSnapshotUpgradeStatsRequest) {
	return func(r *MLGetModelSnapshotUpgradeStatsRequest) {
		r.ctx = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no jobs or no snapshots. (this includes the `_all` string.).
//
func (f MLGetModelSnapshotUpgradeStats) WithAllowNoMatch(v bool) func(*MLGetModelSnapshotUpgradeStatsRequest) {
	return func(r *MLGetModelSnapshotUpgradeStatsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f MLGetModelSnapshotUpgradeStats) WithPretty() func(*MLGetModelSnapshotUpgradeStatsRequest) {
	return func(r *MLGetModelSnapshotUpgradeStatsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f MLGetModelSnapshotUpgradeStats) WithHuman() func(*MLGetModelSnapshotUpgradeStatsRequest) {
	return func(r *MLGetModelSnapshotUpgradeStatsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f MLGetModelSnapshotUpgradeStats) WithErrorTrace() func(*MLGetModelSnapshotUpgradeStatsRequest) {
	return func(r *MLGetModelSnapshotUpgradeStatsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f MLGetModelSnapshotUpgradeStats) WithFilterPath(v ...string) func(*MLGetModelSnapshotUpgradeStatsRequest) {
	return func(r *MLGetModelSnapshotUpgradeStatsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f MLGetModelSnapshotUpgradeStats) WithHeader(h map[string]string) func(*MLGetModelSnapshotUpgradeStatsRequest) {
	return func(r *MLGetModelSnapshotUpgradeStatsRequest) {
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
func (f MLGetModelSnapshotUpgradeStats) WithOpaqueID(s string) func(*MLGetModelSnapshotUpgradeStatsRequest) {
	return func(r *MLGetModelSnapshotUpgradeStatsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
