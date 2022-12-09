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
	"strings"
)

func newSecurityDeleteServiceTokenFunc(t Transport) SecurityDeleteServiceToken {
	return func(name string, namespace string, service string, o ...func(*SecurityDeleteServiceTokenRequest)) (*Response, error) {
		var r = SecurityDeleteServiceTokenRequest{Name: name, Namespace: namespace, Service: service}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SecurityDeleteServiceToken - Deletes a service account token.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api-delete-service-token.html.
//
type SecurityDeleteServiceToken func(name string, namespace string, service string, o ...func(*SecurityDeleteServiceTokenRequest)) (*Response, error)

// SecurityDeleteServiceTokenRequest configures the Security Delete Service Token API request.
//
type SecurityDeleteServiceTokenRequest struct {
	Name      string
	Namespace string
	Service   string

	Refresh string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r SecurityDeleteServiceTokenRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "DELETE"

	path.Grow(1 + len("_security") + 1 + len("service") + 1 + len(r.Namespace) + 1 + len(r.Service) + 1 + len("credential") + 1 + len("token") + 1 + len(r.Name))
	path.WriteString("/")
	path.WriteString("_security")
	path.WriteString("/")
	path.WriteString("service")
	path.WriteString("/")
	path.WriteString(r.Namespace)
	path.WriteString("/")
	path.WriteString(r.Service)
	path.WriteString("/")
	path.WriteString("credential")
	path.WriteString("/")
	path.WriteString("token")
	path.WriteString("/")
	path.WriteString(r.Name)

	params = make(map[string]string)

	if r.Refresh != "" {
		params["refresh"] = r.Refresh
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
func (f SecurityDeleteServiceToken) WithContext(v context.Context) func(*SecurityDeleteServiceTokenRequest) {
	return func(r *SecurityDeleteServiceTokenRequest) {
		r.ctx = v
	}
}

// WithRefresh - if `true` then refresh the affected shards to make this operation visible to search, if `wait_for` (the default) then wait for a refresh to make this operation visible to search, if `false` then do nothing with refreshes..
//
func (f SecurityDeleteServiceToken) WithRefresh(v string) func(*SecurityDeleteServiceTokenRequest) {
	return func(r *SecurityDeleteServiceTokenRequest) {
		r.Refresh = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f SecurityDeleteServiceToken) WithPretty() func(*SecurityDeleteServiceTokenRequest) {
	return func(r *SecurityDeleteServiceTokenRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f SecurityDeleteServiceToken) WithHuman() func(*SecurityDeleteServiceTokenRequest) {
	return func(r *SecurityDeleteServiceTokenRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f SecurityDeleteServiceToken) WithErrorTrace() func(*SecurityDeleteServiceTokenRequest) {
	return func(r *SecurityDeleteServiceTokenRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f SecurityDeleteServiceToken) WithFilterPath(v ...string) func(*SecurityDeleteServiceTokenRequest) {
	return func(r *SecurityDeleteServiceTokenRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f SecurityDeleteServiceToken) WithHeader(h map[string]string) func(*SecurityDeleteServiceTokenRequest) {
	return func(r *SecurityDeleteServiceTokenRequest) {
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
func (f SecurityDeleteServiceToken) WithOpaqueID(s string) func(*SecurityDeleteServiceTokenRequest) {
	return func(r *SecurityDeleteServiceTokenRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
