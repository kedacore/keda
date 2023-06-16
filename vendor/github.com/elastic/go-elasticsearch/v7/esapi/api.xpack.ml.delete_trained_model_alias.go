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
	"strings"
)

func newMLDeleteTrainedModelAliasFunc(t Transport) MLDeleteTrainedModelAlias {
	return func(model_alias string, model_id string, o ...func(*MLDeleteTrainedModelAliasRequest)) (*Response, error) {
		var r = MLDeleteTrainedModelAliasRequest{ModelAlias: model_alias, ModelID: model_id}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// MLDeleteTrainedModelAlias - Deletes a model alias that refers to the trained model
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/delete-trained-models-aliases.html.
type MLDeleteTrainedModelAlias func(model_alias string, model_id string, o ...func(*MLDeleteTrainedModelAliasRequest)) (*Response, error)

// MLDeleteTrainedModelAliasRequest configures the ML Delete Trained Model Alias API request.
type MLDeleteTrainedModelAliasRequest struct {
	ModelAlias string
	ModelID    string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r MLDeleteTrainedModelAliasRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "DELETE"

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
func (f MLDeleteTrainedModelAlias) WithContext(v context.Context) func(*MLDeleteTrainedModelAliasRequest) {
	return func(r *MLDeleteTrainedModelAliasRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f MLDeleteTrainedModelAlias) WithPretty() func(*MLDeleteTrainedModelAliasRequest) {
	return func(r *MLDeleteTrainedModelAliasRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f MLDeleteTrainedModelAlias) WithHuman() func(*MLDeleteTrainedModelAliasRequest) {
	return func(r *MLDeleteTrainedModelAliasRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f MLDeleteTrainedModelAlias) WithErrorTrace() func(*MLDeleteTrainedModelAliasRequest) {
	return func(r *MLDeleteTrainedModelAliasRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f MLDeleteTrainedModelAlias) WithFilterPath(v ...string) func(*MLDeleteTrainedModelAliasRequest) {
	return func(r *MLDeleteTrainedModelAliasRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f MLDeleteTrainedModelAlias) WithHeader(h map[string]string) func(*MLDeleteTrainedModelAliasRequest) {
	return func(r *MLDeleteTrainedModelAliasRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f MLDeleteTrainedModelAlias) WithOpaqueID(s string) func(*MLDeleteTrainedModelAliasRequest) {
	return func(r *MLDeleteTrainedModelAliasRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
