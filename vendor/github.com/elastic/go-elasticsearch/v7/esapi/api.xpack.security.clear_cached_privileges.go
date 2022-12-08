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
	"strings"
)

func newSecurityClearCachedPrivilegesFunc(t Transport) SecurityClearCachedPrivileges {
	return func(application []string, o ...func(*SecurityClearCachedPrivilegesRequest)) (*Response, error) {
		var r = SecurityClearCachedPrivilegesRequest{Application: application}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SecurityClearCachedPrivileges - Evicts application privileges from the native application privileges cache.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api-clear-privilege-cache.html.
//
type SecurityClearCachedPrivileges func(application []string, o ...func(*SecurityClearCachedPrivilegesRequest)) (*Response, error)

// SecurityClearCachedPrivilegesRequest configures the Security Clear Cached Privileges API request.
//
type SecurityClearCachedPrivilegesRequest struct {
	Application []string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r SecurityClearCachedPrivilegesRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	if len(r.Application) == 0 {
		return nil, errors.New("application is required and cannot be nil or empty")
	}

	path.Grow(1 + len("_security") + 1 + len("privilege") + 1 + len(strings.Join(r.Application, ",")) + 1 + len("_clear_cache"))
	path.WriteString("/")
	path.WriteString("_security")
	path.WriteString("/")
	path.WriteString("privilege")
	path.WriteString("/")
	path.WriteString(strings.Join(r.Application, ","))
	path.WriteString("/")
	path.WriteString("_clear_cache")

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
func (f SecurityClearCachedPrivileges) WithContext(v context.Context) func(*SecurityClearCachedPrivilegesRequest) {
	return func(r *SecurityClearCachedPrivilegesRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f SecurityClearCachedPrivileges) WithPretty() func(*SecurityClearCachedPrivilegesRequest) {
	return func(r *SecurityClearCachedPrivilegesRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f SecurityClearCachedPrivileges) WithHuman() func(*SecurityClearCachedPrivilegesRequest) {
	return func(r *SecurityClearCachedPrivilegesRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f SecurityClearCachedPrivileges) WithErrorTrace() func(*SecurityClearCachedPrivilegesRequest) {
	return func(r *SecurityClearCachedPrivilegesRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f SecurityClearCachedPrivileges) WithFilterPath(v ...string) func(*SecurityClearCachedPrivilegesRequest) {
	return func(r *SecurityClearCachedPrivilegesRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f SecurityClearCachedPrivileges) WithHeader(h map[string]string) func(*SecurityClearCachedPrivilegesRequest) {
	return func(r *SecurityClearCachedPrivilegesRequest) {
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
func (f SecurityClearCachedPrivileges) WithOpaqueID(s string) func(*SecurityClearCachedPrivilegesRequest) {
	return func(r *SecurityClearCachedPrivilegesRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
