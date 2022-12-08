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
	"time"
)

func newAsyncSearchSubmitFunc(t Transport) AsyncSearchSubmit {
	return func(o ...func(*AsyncSearchSubmitRequest)) (*Response, error) {
		var r = AsyncSearchSubmitRequest{}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// AsyncSearchSubmit - Executes a search request asynchronously.
//
// See full documentation at https://www.elastic.co/guide/en/elasticsearch/reference/current/async-search.html.
//
type AsyncSearchSubmit func(o ...func(*AsyncSearchSubmitRequest)) (*Response, error)

// AsyncSearchSubmitRequest configures the Async Search Submit API request.
//
type AsyncSearchSubmitRequest struct {
	Index []string

	Body io.Reader

	AllowNoIndices             *bool
	AllowPartialSearchResults  *bool
	Analyzer                   string
	AnalyzeWildcard            *bool
	BatchedReduceSize          *int
	DefaultOperator            string
	Df                         string
	DocvalueFields             []string
	ExpandWildcards            string
	Explain                    *bool
	From                       *int
	IgnoreThrottled            *bool
	IgnoreUnavailable          *bool
	KeepAlive                  time.Duration
	KeepOnCompletion           *bool
	Lenient                    *bool
	MaxConcurrentShardRequests *int
	Preference                 string
	Query                      string
	RequestCache               *bool
	Routing                    []string
	SearchType                 string
	SeqNoPrimaryTerm           *bool
	Size                       *int
	Sort                       []string
	Source                     []string
	SourceExcludes             []string
	SourceIncludes             []string
	Stats                      []string
	StoredFields               []string
	SuggestField               string
	SuggestMode                string
	SuggestSize                *int
	SuggestText                string
	TerminateAfter             *int
	Timeout                    time.Duration
	TrackScores                *bool
	TrackTotalHits             *bool
	TypedKeys                  *bool
	Version                    *bool
	WaitForCompletionTimeout   time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r AsyncSearchSubmitRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "POST"

	path.Grow(1 + len(strings.Join(r.Index, ",")) + 1 + len("_async_search"))
	if len(r.Index) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.Index, ","))
	}
	path.WriteString("/")
	path.WriteString("_async_search")

	params = make(map[string]string)

	if r.AllowNoIndices != nil {
		params["allow_no_indices"] = strconv.FormatBool(*r.AllowNoIndices)
	}

	if r.AllowPartialSearchResults != nil {
		params["allow_partial_search_results"] = strconv.FormatBool(*r.AllowPartialSearchResults)
	}

	if r.Analyzer != "" {
		params["analyzer"] = r.Analyzer
	}

	if r.AnalyzeWildcard != nil {
		params["analyze_wildcard"] = strconv.FormatBool(*r.AnalyzeWildcard)
	}

	if r.BatchedReduceSize != nil {
		params["batched_reduce_size"] = strconv.FormatInt(int64(*r.BatchedReduceSize), 10)
	}

	if r.DefaultOperator != "" {
		params["default_operator"] = r.DefaultOperator
	}

	if r.Df != "" {
		params["df"] = r.Df
	}

	if len(r.DocvalueFields) > 0 {
		params["docvalue_fields"] = strings.Join(r.DocvalueFields, ",")
	}

	if r.ExpandWildcards != "" {
		params["expand_wildcards"] = r.ExpandWildcards
	}

	if r.Explain != nil {
		params["explain"] = strconv.FormatBool(*r.Explain)
	}

	if r.From != nil {
		params["from"] = strconv.FormatInt(int64(*r.From), 10)
	}

	if r.IgnoreThrottled != nil {
		params["ignore_throttled"] = strconv.FormatBool(*r.IgnoreThrottled)
	}

	if r.IgnoreUnavailable != nil {
		params["ignore_unavailable"] = strconv.FormatBool(*r.IgnoreUnavailable)
	}

	if r.KeepAlive != 0 {
		params["keep_alive"] = formatDuration(r.KeepAlive)
	}

	if r.KeepOnCompletion != nil {
		params["keep_on_completion"] = strconv.FormatBool(*r.KeepOnCompletion)
	}

	if r.Lenient != nil {
		params["lenient"] = strconv.FormatBool(*r.Lenient)
	}

	if r.MaxConcurrentShardRequests != nil {
		params["max_concurrent_shard_requests"] = strconv.FormatInt(int64(*r.MaxConcurrentShardRequests), 10)
	}

	if r.Preference != "" {
		params["preference"] = r.Preference
	}

	if r.Query != "" {
		params["q"] = r.Query
	}

	if r.RequestCache != nil {
		params["request_cache"] = strconv.FormatBool(*r.RequestCache)
	}

	if len(r.Routing) > 0 {
		params["routing"] = strings.Join(r.Routing, ",")
	}

	if r.SearchType != "" {
		params["search_type"] = r.SearchType
	}

	if r.SeqNoPrimaryTerm != nil {
		params["seq_no_primary_term"] = strconv.FormatBool(*r.SeqNoPrimaryTerm)
	}

	if r.Size != nil {
		params["size"] = strconv.FormatInt(int64(*r.Size), 10)
	}

	if len(r.Sort) > 0 {
		params["sort"] = strings.Join(r.Sort, ",")
	}

	if len(r.Source) > 0 {
		params["_source"] = strings.Join(r.Source, ",")
	}

	if len(r.SourceExcludes) > 0 {
		params["_source_excludes"] = strings.Join(r.SourceExcludes, ",")
	}

	if len(r.SourceIncludes) > 0 {
		params["_source_includes"] = strings.Join(r.SourceIncludes, ",")
	}

	if len(r.Stats) > 0 {
		params["stats"] = strings.Join(r.Stats, ",")
	}

	if len(r.StoredFields) > 0 {
		params["stored_fields"] = strings.Join(r.StoredFields, ",")
	}

	if r.SuggestField != "" {
		params["suggest_field"] = r.SuggestField
	}

	if r.SuggestMode != "" {
		params["suggest_mode"] = r.SuggestMode
	}

	if r.SuggestSize != nil {
		params["suggest_size"] = strconv.FormatInt(int64(*r.SuggestSize), 10)
	}

	if r.SuggestText != "" {
		params["suggest_text"] = r.SuggestText
	}

	if r.TerminateAfter != nil {
		params["terminate_after"] = strconv.FormatInt(int64(*r.TerminateAfter), 10)
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
	}

	if r.TrackScores != nil {
		params["track_scores"] = strconv.FormatBool(*r.TrackScores)
	}

	if r.TrackTotalHits != nil {
		params["track_total_hits"] = strconv.FormatBool(*r.TrackTotalHits)
	}

	if r.TypedKeys != nil {
		params["typed_keys"] = strconv.FormatBool(*r.TypedKeys)
	}

	if r.Version != nil {
		params["version"] = strconv.FormatBool(*r.Version)
	}

	if r.WaitForCompletionTimeout != 0 {
		params["wait_for_completion_timeout"] = formatDuration(r.WaitForCompletionTimeout)
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
func (f AsyncSearchSubmit) WithContext(v context.Context) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.ctx = v
	}
}

// WithBody - The search definition using the Query DSL.
//
func (f AsyncSearchSubmit) WithBody(v io.Reader) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Body = v
	}
}

// WithIndex - a list of index names to search; use _all to perform the operation on all indices.
//
func (f AsyncSearchSubmit) WithIndex(v ...string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Index = v
	}
}

// WithAllowNoIndices - whether to ignore if a wildcard indices expression resolves into no concrete indices. (this includes `_all` string or when no indices have been specified).
//
func (f AsyncSearchSubmit) WithAllowNoIndices(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.AllowNoIndices = &v
	}
}

// WithAllowPartialSearchResults - indicate if an error should be returned if there is a partial search failure or timeout.
//
func (f AsyncSearchSubmit) WithAllowPartialSearchResults(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.AllowPartialSearchResults = &v
	}
}

// WithAnalyzer - the analyzer to use for the query string.
//
func (f AsyncSearchSubmit) WithAnalyzer(v string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Analyzer = v
	}
}

// WithAnalyzeWildcard - specify whether wildcard and prefix queries should be analyzed (default: false).
//
func (f AsyncSearchSubmit) WithAnalyzeWildcard(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.AnalyzeWildcard = &v
	}
}

// WithBatchedReduceSize - the number of shard results that should be reduced at once on the coordinating node. this value should be used as the granularity at which progress results will be made available..
//
func (f AsyncSearchSubmit) WithBatchedReduceSize(v int) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.BatchedReduceSize = &v
	}
}

// WithDefaultOperator - the default operator for query string query (and or or).
//
func (f AsyncSearchSubmit) WithDefaultOperator(v string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.DefaultOperator = v
	}
}

// WithDf - the field to use as default where no field prefix is given in the query string.
//
func (f AsyncSearchSubmit) WithDf(v string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Df = v
	}
}

// WithDocvalueFields - a list of fields to return as the docvalue representation of a field for each hit.
//
func (f AsyncSearchSubmit) WithDocvalueFields(v ...string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.DocvalueFields = v
	}
}

// WithExpandWildcards - whether to expand wildcard expression to concrete indices that are open, closed or both..
//
func (f AsyncSearchSubmit) WithExpandWildcards(v string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.ExpandWildcards = v
	}
}

// WithExplain - specify whether to return detailed information about score computation as part of a hit.
//
func (f AsyncSearchSubmit) WithExplain(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Explain = &v
	}
}

// WithFrom - starting offset (default: 0).
//
func (f AsyncSearchSubmit) WithFrom(v int) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.From = &v
	}
}

// WithIgnoreThrottled - whether specified concrete, expanded or aliased indices should be ignored when throttled.
//
func (f AsyncSearchSubmit) WithIgnoreThrottled(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.IgnoreThrottled = &v
	}
}

// WithIgnoreUnavailable - whether specified concrete indices should be ignored when unavailable (missing or closed).
//
func (f AsyncSearchSubmit) WithIgnoreUnavailable(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.IgnoreUnavailable = &v
	}
}

// WithKeepAlive - update the time interval in which the results (partial or final) for this search will be available.
//
func (f AsyncSearchSubmit) WithKeepAlive(v time.Duration) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.KeepAlive = v
	}
}

// WithKeepOnCompletion - control whether the response should be stored in the cluster if it completed within the provided [wait_for_completion] time (default: false).
//
func (f AsyncSearchSubmit) WithKeepOnCompletion(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.KeepOnCompletion = &v
	}
}

// WithLenient - specify whether format-based query failures (such as providing text to a numeric field) should be ignored.
//
func (f AsyncSearchSubmit) WithLenient(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Lenient = &v
	}
}

// WithMaxConcurrentShardRequests - the number of concurrent shard requests per node this search executes concurrently. this value should be used to limit the impact of the search on the cluster in order to limit the number of concurrent shard requests.
//
func (f AsyncSearchSubmit) WithMaxConcurrentShardRequests(v int) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.MaxConcurrentShardRequests = &v
	}
}

// WithPreference - specify the node or shard the operation should be performed on (default: random).
//
func (f AsyncSearchSubmit) WithPreference(v string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Preference = v
	}
}

// WithQuery - query in the lucene query string syntax.
//
func (f AsyncSearchSubmit) WithQuery(v string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Query = v
	}
}

// WithRequestCache - specify if request cache should be used for this request or not, defaults to true.
//
func (f AsyncSearchSubmit) WithRequestCache(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.RequestCache = &v
	}
}

// WithRouting - a list of specific routing values.
//
func (f AsyncSearchSubmit) WithRouting(v ...string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Routing = v
	}
}

// WithSearchType - search operation type.
//
func (f AsyncSearchSubmit) WithSearchType(v string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.SearchType = v
	}
}

// WithSeqNoPrimaryTerm - specify whether to return sequence number and primary term of the last modification of each hit.
//
func (f AsyncSearchSubmit) WithSeqNoPrimaryTerm(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.SeqNoPrimaryTerm = &v
	}
}

// WithSize - number of hits to return (default: 10).
//
func (f AsyncSearchSubmit) WithSize(v int) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Size = &v
	}
}

// WithSort - a list of <field>:<direction> pairs.
//
func (f AsyncSearchSubmit) WithSort(v ...string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Sort = v
	}
}

// WithSource - true or false to return the _source field or not, or a list of fields to return.
//
func (f AsyncSearchSubmit) WithSource(v ...string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Source = v
	}
}

// WithSourceExcludes - a list of fields to exclude from the returned _source field.
//
func (f AsyncSearchSubmit) WithSourceExcludes(v ...string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.SourceExcludes = v
	}
}

// WithSourceIncludes - a list of fields to extract and return from the _source field.
//
func (f AsyncSearchSubmit) WithSourceIncludes(v ...string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.SourceIncludes = v
	}
}

// WithStats - specific 'tag' of the request for logging and statistical purposes.
//
func (f AsyncSearchSubmit) WithStats(v ...string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Stats = v
	}
}

// WithStoredFields - a list of stored fields to return as part of a hit.
//
func (f AsyncSearchSubmit) WithStoredFields(v ...string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.StoredFields = v
	}
}

// WithSuggestField - specify which field to use for suggestions.
//
func (f AsyncSearchSubmit) WithSuggestField(v string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.SuggestField = v
	}
}

// WithSuggestMode - specify suggest mode.
//
func (f AsyncSearchSubmit) WithSuggestMode(v string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.SuggestMode = v
	}
}

// WithSuggestSize - how many suggestions to return in response.
//
func (f AsyncSearchSubmit) WithSuggestSize(v int) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.SuggestSize = &v
	}
}

// WithSuggestText - the source text for which the suggestions should be returned.
//
func (f AsyncSearchSubmit) WithSuggestText(v string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.SuggestText = v
	}
}

// WithTerminateAfter - the maximum number of documents to collect for each shard, upon reaching which the query execution will terminate early..
//
func (f AsyncSearchSubmit) WithTerminateAfter(v int) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.TerminateAfter = &v
	}
}

// WithTimeout - explicit operation timeout.
//
func (f AsyncSearchSubmit) WithTimeout(v time.Duration) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Timeout = v
	}
}

// WithTrackScores - whether to calculate and return scores even if they are not used for sorting.
//
func (f AsyncSearchSubmit) WithTrackScores(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.TrackScores = &v
	}
}

// WithTrackTotalHits - indicate if the number of documents that match the query should be tracked.
//
func (f AsyncSearchSubmit) WithTrackTotalHits(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.TrackTotalHits = &v
	}
}

// WithTypedKeys - specify whether aggregation and suggester names should be prefixed by their respective types in the response.
//
func (f AsyncSearchSubmit) WithTypedKeys(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.TypedKeys = &v
	}
}

// WithVersion - specify whether to return document version as part of a hit.
//
func (f AsyncSearchSubmit) WithVersion(v bool) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Version = &v
	}
}

// WithWaitForCompletionTimeout - specify the time that the request should block waiting for the final response.
//
func (f AsyncSearchSubmit) WithWaitForCompletionTimeout(v time.Duration) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.WaitForCompletionTimeout = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f AsyncSearchSubmit) WithPretty() func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f AsyncSearchSubmit) WithHuman() func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f AsyncSearchSubmit) WithErrorTrace() func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f AsyncSearchSubmit) WithFilterPath(v ...string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f AsyncSearchSubmit) WithHeader(h map[string]string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
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
func (f AsyncSearchSubmit) WithOpaqueID(s string) func(*AsyncSearchSubmitRequest) {
	return func(r *AsyncSearchSubmitRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
