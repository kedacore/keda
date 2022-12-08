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

func newCatTransformsFunc(t Transport) CatTransforms {
	return func(o ...func(*CatTransformsRequest)) (*Response, error) {
		var r = CatTransformsRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// CatTransforms - Gets configuration and usage information about transforms.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/cat-transforms.html.
//
type CatTransforms func(o ...func(*CatTransformsRequest)) (*Response, error)

// CatTransformsRequest configures the Cat Transforms API request.
//
type CatTransformsRequest struct {
	TransformID string

	AllowNoMatch *bool
	Format       string
	From         *int
	H            []string
	Help         *bool
	S            []string
	Size         *int
	Time         string
	V            *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r CatTransformsRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "GET"

	path.Grow(1 + len("_cat") + 1 + len("transforms") + 1 + len(r.TransformID))
	path.WriteString("/")
	path.WriteString("_cat")
	path.WriteString("/")
	path.WriteString("transforms")
	if r.TransformID != "" {
		path.WriteString("/")
		path.WriteString(r.TransformID)
	}

	params = make(map[string]string)

	if r.AllowNoMatch != nil {
		params["allow_no_match"] = strconv.FormatBool(*r.AllowNoMatch)
	}

	if r.Format != "" {
		params["format"] = r.Format
	}

	if r.From != nil {
		params["from"] = strconv.FormatInt(int64(*r.From), 10)
	}

	if len(r.H) > 0 {
		params["h"] = strings.Join(r.H, ",")
	}

	if r.Help != nil {
		params["help"] = strconv.FormatBool(*r.Help)
	}

	if len(r.S) > 0 {
		params["s"] = strings.Join(r.S, ",")
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
	}

	if r.Time != "" {
		params["time"] = r.Time
	}

	if r.V != nil {
		params["v"] = strconv.FormatBool(*r.V)
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
func (f CatTransforms) WithContext(v context.Context) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.ctx = v
	}
}

// WithTransformID - the ID of the transform for which to get stats. '_all' or '*' implies all transforms.
//
func (f CatTransforms) WithTransformID(v string) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.TransformID = v
	}
}

// WithAllowNoMatch - whether to ignore if a wildcard expression matches no transforms. (this includes `_all` string or when no transforms have been specified).
//
func (f CatTransforms) WithAllowNoMatch(v bool) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.AllowNoMatch = &v
	}
}

// WithFormat - a short version of the accept header, e.g. json, yaml.
//
func (f CatTransforms) WithFormat(v string) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.Format = v
	}
}

// WithFrom - skips a number of transform configs, defaults to 0.
//
func (f CatTransforms) WithFrom(v int) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.From = &v
	}
}

// WithH - comma-separated list of column names to display.
//
func (f CatTransforms) WithH(v ...string) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.H = v
	}
}

// WithHelp - return help information.
//
func (f CatTransforms) WithHelp(v bool) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.Help = &v
	}
}

// WithS - comma-separated list of column names or column aliases to sort by.
//
func (f CatTransforms) WithS(v ...string) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.S = v
	}
}

// WithSize - specifies a max number of transforms to get, defaults to 100.
//
func (f CatTransforms) WithSize(v int) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.Size = &v
	}
}

// WithTime - the unit in which to display time values.
//
func (f CatTransforms) WithTime(v string) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.Time = v
	}
}

// WithV - verbose mode. display column headers.
//
func (f CatTransforms) WithV(v bool) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.V = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f CatTransforms) WithPretty() func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f CatTransforms) WithHuman() func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f CatTransforms) WithErrorTrace() func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f CatTransforms) WithFilterPath(v ...string) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f CatTransforms) WithHeader(h map[string]string) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
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
func (f CatTransforms) WithOpaqueID(s string) func(*CatTransformsRequest) {
	return func(r *CatTransformsRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
