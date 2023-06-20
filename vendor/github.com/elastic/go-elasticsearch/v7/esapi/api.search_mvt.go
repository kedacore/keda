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
	"io"
	"net/http"
	"strconv"
	"strings"
)

func newSearchMvtFunc(t Transport) SearchMvt {
	return func(index []string, field string, zoom *int, x *int, y *int, o ...func(*SearchMvtRequest)) (*Response, error) {
		var r = SearchMvtRequest{Index: index, Y: y, Field: field, Zoom: zoom, X: x}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// SearchMvt searches a vector tile for geospatial values. Returns results as a binary Mapbox vector tile.
//
// This API is experimental.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/master/search-vector-tile-api.html.
type SearchMvt func(index []string, field string, zoom *int, x *int, y *int, o ...func(*SearchMvtRequest)) (*Response, error)

// SearchMvtRequest configures the Search Mvt API request.
type SearchMvtRequest struct {
	Index []string

	Body io.Reader

	Field string
	X     *int
	Y     *int
	Zoom  *int

	ExactBounds    *bool
	Extent         *int
	GridPrecision  *int
	GridType       string
	Size           *int
	TrackTotalHits interface{}

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r SearchMvtRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	if len(r.Index) == 0 {
		return nil, errors.New("index is required and cannot be nil or empty")
	}
	if r.Zoom == nil {
		return nil, errors.New("zoom is required and cannot be nil")
	}
	if r.X == nil {
		return nil, errors.New("x is required and cannot be nil")
	}
	if r.Y == nil {
		return nil, errors.New("y is required and cannot be nil")
	}

	path.Grow(1 + len(strings.Join(r.Index, ",")) + 1 + len("_mvt") + 1 + len(r.Field) + 1 + len(strconv.Itoa(*r.Zoom)) + 1 + len(strconv.Itoa(*r.X)) + 1 + len(strconv.Itoa(*r.Y)))
	path.WriteString("/")
	path.WriteString(strings.Join(r.Index, ","))
	path.WriteString("/")
	path.WriteString("_mvt")
	path.WriteString("/")
	path.WriteString(r.Field)
	path.WriteString("/")
	path.WriteString(strconv.Itoa(*r.Zoom))
	path.WriteString("/")
	path.WriteString(strconv.Itoa(*r.X))
	path.WriteString("/")
	path.WriteString(strconv.Itoa(*r.Y))

	params = make(map[string]string)

	if r.ExactBounds != nil {
		params["exact_bounds"] = strconv.FormatBool(*r.ExactBounds)
	}

	if r.Extent != nil {
		params["extent"] = strconv.FormatInt(int64(*r.Extent), 10)
	}

	if r.GridPrecision != nil {
		params["grid_precision"] = strconv.FormatInt(int64(*r.GridPrecision), 10)
	}

	if r.GridType != "" {
		params["grid_type"] = r.GridType
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
	}

	if r.TrackTotalHits != nil {
		params["track_total_hits"] = fmt.Sprintf("%v", r.TrackTotalHits)
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

	if r.Body != nil && req.Header.Get(headerContentType) == "" {
		req.Header[headerContentType] = headerContentTypeJSON
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
func (f SearchMvt) WithContext(v context.Context) func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.ctx = v
	}
}

// WithBody - Search request body..
func (f SearchMvt) WithBody(v io.Reader) func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.Body = v
	}
}

// WithExactBounds - if false, the meta layer's feature is the bounding box of the tile. if true, the meta layer's feature is a bounding box resulting from a `geo_bounds` aggregation..
func (f SearchMvt) WithExactBounds(v bool) func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.ExactBounds = &v
	}
}

// WithExtent - size, in pixels, of a side of the vector tile..
func (f SearchMvt) WithExtent(v int) func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.Extent = &v
	}
}

// WithGridPrecision - additional zoom levels available through the aggs layer. accepts 0-8..
func (f SearchMvt) WithGridPrecision(v int) func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.GridPrecision = &v
	}
}

// WithGridType - determines the geometry type for features in the aggs layer..
func (f SearchMvt) WithGridType(v string) func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.GridType = v
	}
}

// WithSize - maximum number of features to return in the hits layer. accepts 0-10000..
func (f SearchMvt) WithSize(v int) func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.Size = &v
	}
}

// WithTrackTotalHits - indicate if the number of documents that match the query should be tracked. a number can also be specified, to accurately track the total hit count up to the number..
func (f SearchMvt) WithTrackTotalHits(v interface{}) func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.TrackTotalHits = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f SearchMvt) WithPretty() func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f SearchMvt) WithHuman() func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f SearchMvt) WithErrorTrace() func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f SearchMvt) WithFilterPath(v ...string) func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f SearchMvt) WithHeader(h map[string]string) func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f SearchMvt) WithOpaqueID(s string) func(*SearchMvtRequest) {
	return func(r *SearchMvtRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
