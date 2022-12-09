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

func newMLPutTrainedModelAliasFunc(t Transport) MLPutTrainedModelAlias {
	return func(model_alias string, model_id string, o ...func(*MLPutTrainedModelAliasRequest)) (*Response, error) {
		var r = MLPutTrainedModelAliasRequest{ModelAlias: model_alias, ModelID: model_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLPutTrainedModelAlias - Creates a new model alias (or reassigns an existing one) to refer to the trained model
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/put-trained-models-aliases.html.
//
type MLPutTrainedModelAlias func(model_alias string, model_id string, o ...func(*MLPutTrainedModelAliasRequest)) (*Response, error)

// MLPutTrainedModelAliasRequest configures the ML Put Trained Model Alias API request.
//
type MLPutTrainedModelAliasRequest struct {
	ModelAlias string
	ModelID    string

	Reassign *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r MLPutTrainedModelAliasRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "PUT"

	path.Grow(1 + len("_ml") + 1 + len("trained_models") + 1 + len(r.ModelID) + 1 + len("model_aliases") + 1 + len(r.ModelAlias))
	path.WriteString("/")
	path.WriteString("_ml")
	path.WriteString("/")
	path.WriteString("trained_models")
	path.WriteString("/")
	path.WriteString(r.ModelID)
	path.WriteString("/")
	path.WriteString("model_aliases")
	path.WriteString("/")
	path.WriteString(r.ModelAlias)

	params = make(map[string]string)

	if r.Reassign != nil {
		params["reassign"] = strconv.FormatBool(*r.Reassign)
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
func (f MLPutTrainedModelAlias) WithContext(v context.Context) func(*MLPutTrainedModelAliasRequest) {
	return func(r *MLPutTrainedModelAliasRequest) {
		r.ctx = v
	}
}

// WithReassign - if the model_alias already exists and points to a separate model_id, this parameter must be true. defaults to false..
//
func (f MLPutTrainedModelAlias) WithReassign(v bool) func(*MLPutTrainedModelAliasRequest) {
	return func(r *MLPutTrainedModelAliasRequest) {
		r.Reassign = &v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f MLPutTrainedModelAlias) WithPretty() func(*MLPutTrainedModelAliasRequest) {
	return func(r *MLPutTrainedModelAliasRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f MLPutTrainedModelAlias) WithHuman() func(*MLPutTrainedModelAliasRequest) {
	return func(r *MLPutTrainedModelAliasRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f MLPutTrainedModelAlias) WithErrorTrace() func(*MLPutTrainedModelAliasRequest) {
	return func(r *MLPutTrainedModelAliasRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f MLPutTrainedModelAlias) WithFilterPath(v ...string) func(*MLPutTrainedModelAliasRequest) {
	return func(r *MLPutTrainedModelAliasRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f MLPutTrainedModelAlias) WithHeader(h map[string]string) func(*MLPutTrainedModelAliasRequest) {
	return func(r *MLPutTrainedModelAliasRequest) {
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
func (f MLPutTrainedModelAlias) WithOpaqueID(s string) func(*MLPutTrainedModelAliasRequest) {
	return func(r *MLPutTrainedModelAliasRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
