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
	"io"
	"net/http"
	"strconv"
	"strings"
)

func newILMMigrateToDataTiersFunc(t Transport) ILMMigrateToDataTiers {
	return func(o ...func(*ILMMigrateToDataTiersRequest)) (*Response, error) {
		var r = ILMMigrateToDataTiersRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// ILMMigrateToDataTiers - Migrates the indices and ILM policies away from custom node attribute allocation routing to data tiers routing
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/ilm-migrate-to-data-tiers.html.
//
type ILMMigrateToDataTiers func(o ...func(*ILMMigrateToDataTiersRequest)) (*Response, error)

// ILMMigrateToDataTiersRequest configures the ILM Migrate To Data Tiers API request.
//
type ILMMigrateToDataTiersRequest struct {
	Body io.Reader

	DryRun *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r ILMMigrateToDataTiersRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(len("/_ilm/migrate_to_data_tiers"))
	path.WriteString("/_ilm/migrate_to_data_tiers")

	params = make(map[string]string)

	if r.DryRun != nil {
		params["dry_run"] = strconv.FormatBool(*r.DryRun)
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

	if r.Body != nil {
		req.Header[headerContentType] = headerContentTypeJSON
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
func (f ILMMigrateToDataTiers) WithContext(v context.Context) func(*ILMMigrateToDataTiersRequest) {
	return func(r *ILMMigrateToDataTiersRequest) {
		r.ctx = v
	}
}

// WithBody - Optionally specify a legacy index template name to delete and optionally specify a node attribute name used for index shard routing (defaults to "data").
//
func (f ILMMigrateToDataTiers) WithBody(v io.Reader) func(*ILMMigrateToDataTiersRequest) {
	return func(r *ILMMigrateToDataTiersRequest) {
		r.Body = v
	}
}

// WithDryRun - if set to true it will simulate the migration, providing a way to retrieve the ilm policies and indices that need to be migrated. the default is false.
//
func (f ILMMigrateToDataTiers) WithDryRun(v bool) func(*ILMMigrateToDataTiersRequest) {
	return func(r *ILMMigrateToDataTiersRequest) {
		r.DryRun = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f ILMMigrateToDataTiers) WithPretty() func(*ILMMigrateToDataTiersRequest) {
	return func(r *ILMMigrateToDataTiersRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f ILMMigrateToDataTiers) WithHuman() func(*ILMMigrateToDataTiersRequest) {
	return func(r *ILMMigrateToDataTiersRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f ILMMigrateToDataTiers) WithErrorTrace() func(*ILMMigrateToDataTiersRequest) {
	return func(r *ILMMigrateToDataTiersRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f ILMMigrateToDataTiers) WithFilterPath(v ...string) func(*ILMMigrateToDataTiersRequest) {
	return func(r *ILMMigrateToDataTiersRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f ILMMigrateToDataTiers) WithHeader(h map[string]string) func(*ILMMigrateToDataTiersRequest) {
	return func(r *ILMMigrateToDataTiersRequest) {
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
func (f ILMMigrateToDataTiers) WithOpaqueID(s string) func(*ILMMigrateToDataTiersRequest) {
	return func(r *ILMMigrateToDataTiersRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
