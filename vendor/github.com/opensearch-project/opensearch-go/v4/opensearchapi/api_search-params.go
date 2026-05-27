// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.
//
// Modifications Copyright OpenSearch Contributors. See
// GitHub history for details.

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

package opensearchapi

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SearchParams represents possible parameters for the SearchReq
type SearchParams struct {
	AllowNoIndices             *bool
	AllowPartialSearchResults  *bool
	Analyzer                   string
	AnalyzeWildcard            *bool
	BatchedReduceSize          *int
	CcsMinimizeRoundtrips      *bool
	DefaultOperator            string
	Df                         string
	DocvalueFields             []string
	ExpandWildcards            string
	Explain                    *bool
	From                       *int
	IgnoreThrottled            *bool
	IgnoreUnavailable          *bool
	Lenient                    *bool
	MaxConcurrentShardRequests *int
	MinCompatibleShardNode     string
	Preference                 string
	PreFilterShardSize         *int
	Query                      string
	RequestCache               *bool
	RestTotalHitsAsInt         *bool
	Routing                    []string
	Scroll                     time.Duration
	SearchPipeline             string
	SearchType                 string
	SeqNoPrimaryTerm           *bool
	Size                       *int
	Sort                       []string
	Source                     interface{}
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
	TrackTotalHits             interface{}
	TypedKeys                  *bool
	Version                    *bool
	PhaseTook                  bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string
}

func (r SearchParams) get() map[string]string {
	params := make(map[string]string)

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

	if r.CcsMinimizeRoundtrips != nil {
		params["ccs_minimize_roundtrips"] = strconv.FormatBool(*r.CcsMinimizeRoundtrips)
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

	if r.Lenient != nil {
		params["lenient"] = strconv.FormatBool(*r.Lenient)
	}

	if r.MaxConcurrentShardRequests != nil {
		params["max_concurrent_shard_requests"] = strconv.FormatInt(int64(*r.MaxConcurrentShardRequests), 10)
	}

	if r.MinCompatibleShardNode != "" {
		params["min_compatible_shard_node"] = r.MinCompatibleShardNode
	}

	if r.Preference != "" {
		params["preference"] = r.Preference
	}

	if r.PreFilterShardSize != nil {
		params["pre_filter_shard_size"] = strconv.FormatInt(int64(*r.PreFilterShardSize), 10)
	}

	if r.Query != "" {
		params["q"] = r.Query
	}

	if r.RequestCache != nil {
		params["request_cache"] = strconv.FormatBool(*r.RequestCache)
	}

	if r.RestTotalHitsAsInt != nil {
		params["rest_total_hits_as_int"] = strconv.FormatBool(*r.RestTotalHitsAsInt)
	}

	if len(r.Routing) > 0 {
		params["routing"] = strings.Join(r.Routing, ",")
	}

	if r.Scroll != 0 {
		params["scroll"] = formatDuration(r.Scroll)
	}

	if r.SearchPipeline != "" {
		params["search_pipeline"] = r.SearchPipeline
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

	switch source := r.Source.(type) {
	case bool:
		params["_source"] = strconv.FormatBool(source)
	case string:
		if source != "" {
			params["_source"] = source
		}
	case []string:
		if len(source) > 0 {
			params["_source"] = strings.Join(source, ",")
		}
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
		params["track_total_hits"] = fmt.Sprintf("%v", r.TrackTotalHits)
	}

	if r.TypedKeys != nil {
		params["typed_keys"] = strconv.FormatBool(*r.TypedKeys)
	}

	if r.Version != nil {
		params["version"] = strconv.FormatBool(*r.Version)
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

	if r.PhaseTook {
		params["phase_took"] = "true"
	}

	return params
}
