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
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func newTextStructureFindStructureFunc(t Transport) TextStructureFindStructure {
	return func(body io.Reader, o ...func(*TextStructureFindStructureRequest)) (*Response, error) {
		var r = TextStructureFindStructureRequest{Body: body}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// TextStructureFindStructure - Finds the structure of a text file. The text file must contain data that is suitable to be ingested into Elasticsearch.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/find-structure.html.
type TextStructureFindStructure func(body io.Reader, o ...func(*TextStructureFindStructureRequest)) (*Response, error)

// TextStructureFindStructureRequest configures the Text Structure Find Structure API request.
type TextStructureFindStructureRequest struct {
	Body io.Reader

	Charset            string
	ColumnNames        []string
	Delimiter          string
	Explain            *bool
	Format             string
	GrokPattern        string
	HasHeaderRow       *bool
	LineMergeSizeLimit *int
	LinesToSample      *int
	Quote              string
	ShouldTrimFields   *bool
	Timeout            time.Duration
	TimestampField     string
	TimestampFormat    string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
func (r TextStructureFindStructureRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(len("/_text_structure/find_structure"))
	path.WriteString("/_text_structure/find_structure")

	params = make(map[string]string)

	if r.Charset != "" {
		params["charset"] = r.Charset
	}

	if len(r.ColumnNames) > 0 {
		params["column_names"] = strings.Join(r.ColumnNames, ",")
	}

	if r.Delimiter != "" {
		params["delimiter"] = r.Delimiter
	}

	if r.Explain != nil {
		params["explain"] = strconv.FormatBool(*r.Explain)
	}

	if r.Format != "" {
		params["format"] = r.Format
	}

	if r.GrokPattern != "" {
		params["grok_pattern"] = r.GrokPattern
	}

	if r.HasHeaderRow != nil {
		params["has_header_row"] = strconv.FormatBool(*r.HasHeaderRow)
	}

	if r.LineMergeSizeLimit != nil {
		params["line_merge_size_limit"] = strconv.FormatInt(int64(*r.LineMergeSizeLimit), 10)
	}

	if r.LinesToSample != nil {
		params["lines_to_sample"] = strconv.FormatInt(int64(*r.LinesToSample), 10)
	}

	if r.Quote != "" {
		params["quote"] = r.Quote
	}

	if r.ShouldTrimFields != nil {
		params["should_trim_fields"] = strconv.FormatBool(*r.ShouldTrimFields)
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
	}

	if r.TimestampField != "" {
		params["timestamp_field"] = r.TimestampField
	}

	if r.TimestampFormat != "" {
		params["timestamp_format"] = r.TimestampFormat
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
func (f TextStructureFindStructure) WithContext(v context.Context) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.ctx = v
	}
}

// WithCharset - optional parameter to specify the character set of the file.
func (f TextStructureFindStructure) WithCharset(v string) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.Charset = v
	}
}

// WithColumnNames - optional parameter containing a comma separated list of the column names for a delimited file.
func (f TextStructureFindStructure) WithColumnNames(v ...string) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.ColumnNames = v
	}
}

// WithDelimiter - optional parameter to specify the delimiter character for a delimited file - must be a single character.
func (f TextStructureFindStructure) WithDelimiter(v string) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.Delimiter = v
	}
}

// WithExplain - whether to include a commentary on how the structure was derived.
func (f TextStructureFindStructure) WithExplain(v bool) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.Explain = &v
	}
}

// WithFormat - optional parameter to specify the high level file format.
func (f TextStructureFindStructure) WithFormat(v string) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.Format = v
	}
}

// WithGrokPattern - optional parameter to specify the grok pattern that should be used to extract fields from messages in a semi-structured text file.
func (f TextStructureFindStructure) WithGrokPattern(v string) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.GrokPattern = v
	}
}

// WithHasHeaderRow - optional parameter to specify whether a delimited file includes the column names in its first row.
func (f TextStructureFindStructure) WithHasHeaderRow(v bool) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.HasHeaderRow = &v
	}
}

// WithLineMergeSizeLimit - maximum number of characters permitted in a single message when lines are merged to create messages..
func (f TextStructureFindStructure) WithLineMergeSizeLimit(v int) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.LineMergeSizeLimit = &v
	}
}

// WithLinesToSample - how many lines of the file should be included in the analysis.
func (f TextStructureFindStructure) WithLinesToSample(v int) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.LinesToSample = &v
	}
}

// WithQuote - optional parameter to specify the quote character for a delimited file - must be a single character.
func (f TextStructureFindStructure) WithQuote(v string) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.Quote = v
	}
}

// WithShouldTrimFields - optional parameter to specify whether the values between delimiters in a delimited file should have whitespace trimmed from them.
func (f TextStructureFindStructure) WithShouldTrimFields(v bool) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.ShouldTrimFields = &v
	}
}

// WithTimeout - timeout after which the analysis will be aborted.
func (f TextStructureFindStructure) WithTimeout(v time.Duration) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.Timeout = v
	}
}

// WithTimestampField - optional parameter to specify the timestamp field in the file.
func (f TextStructureFindStructure) WithTimestampField(v string) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.TimestampField = v
	}
}

// WithTimestampFormat - optional parameter to specify the timestamp format in the file - may be either a joda or java time format.
func (f TextStructureFindStructure) WithTimestampFormat(v string) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.TimestampFormat = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f TextStructureFindStructure) WithPretty() func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f TextStructureFindStructure) WithHuman() func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f TextStructureFindStructure) WithErrorTrace() func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f TextStructureFindStructure) WithFilterPath(v ...string) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f TextStructureFindStructure) WithHeader(h map[string]string) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f TextStructureFindStructure) WithOpaqueID(s string) func(*TextStructureFindStructureRequest) {
	return func(r *TextStructureFindStructureRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
