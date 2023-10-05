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
	"strconv"
	"strings"
	"time"
)

func newMLUpgradeJobSnapshotFunc(t Transport) MLUpgradeJobSnapshot {
	return func(snapshot_id string, job_id string, o ...func(*MLUpgradeJobSnapshotRequest)) (*Response, error) {
		var r = MLUpgradeJobSnapshotRequest{SnapshotID: snapshot_id, JobID: job_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLUpgradeJobSnapshot - Upgrades a given job snapshot to the current major version.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-upgrade-job-model-snapshot.html.
type MLUpgradeJobSnapshot func(snapshot_id string, job_id string, o ...func(*MLUpgradeJobSnapshotRequest)) (*Response, error)

// MLUpgradeJobSnapshotRequest configures the ML Upgrade Job Snapshot API request.
type MLUpgradeJobSnapshotRequest struct {
	JobID      string
	SnapshotID string

	Timeout           time.Duration
	WaitForCompletion *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r MLUpgradeJobSnapshotRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len("_ml") + 1 + len("anomaly_detectors") + 1 + len(r.JobID) + 1 + len("model_snapshots") + 1 + len(r.SnapshotID) + 1 + len("_upgrade"))
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

	params = make(map[string]string)

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
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
func (f MLUpgradeJobSnapshot) WithContext(v context.Context) func(*MLUpgradeJobSnapshotRequest) {
	return func(r *MLUpgradeJobSnapshotRequest) {
		r.ctx = v
	}
}

// WithTimeout - how long should the api wait for the job to be opened and the old snapshot to be loaded..
func (f MLUpgradeJobSnapshot) WithTimeout(v time.Duration) func(*MLUpgradeJobSnapshotRequest) {
	return func(r *MLUpgradeJobSnapshotRequest) {
		r.Timeout = v
	}
}

// WithWaitForCompletion - should the request wait until the task is complete before responding to the caller. default is false..
func (f MLUpgradeJobSnapshot) WithWaitForCompletion(v bool) func(*MLUpgradeJobSnapshotRequest) {
	return func(r *MLUpgradeJobSnapshotRequest) {
		r.WaitForCompletion = &v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLUpgradeJobSnapshot) WithPretty() func(*MLUpgradeJobSnapshotRequest) {
	return func(r *MLUpgradeJobSnapshotRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLUpgradeJobSnapshot) WithHuman() func(*MLUpgradeJobSnapshotRequest) {
	return func(r *MLUpgradeJobSnapshotRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLUpgradeJobSnapshot) WithErrorTrace() func(*MLUpgradeJobSnapshotRequest) {
	return func(r *MLUpgradeJobSnapshotRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLUpgradeJobSnapshot) WithFilterPath(v ...string) func(*MLUpgradeJobSnapshotRequest) {
	return func(r *MLUpgradeJobSnapshotRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLUpgradeJobSnapshot) WithHeader(h map[string]string) func(*MLUpgradeJobSnapshotRequest) {
	return func(r *MLUpgradeJobSnapshotRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLUpgradeJobSnapshot) WithOpaqueID(s string) func(*MLUpgradeJobSnapshotRequest) {
	return func(r *MLUpgradeJobSnapshotRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
