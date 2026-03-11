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
	"strconv"
	"strings"
	"time"
)

// UpdateParams represents possible parameters for the UpdateReq
type UpdateParams struct {
	IfPrimaryTerm       *int
	IfSeqNo             *int
	Lang                string
	Refresh             string
	RequireAlias        *bool
	RetryOnConflict     *int
	Routing             string
	Source              interface{}
	SourceExcludes      []string
	SourceIncludes      []string
	Timeout             time.Duration
	WaitForActiveShards string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string
}

func (r UpdateParams) get() map[string]string {
	params := make(map[string]string)

	if r.IfPrimaryTerm != nil {
		params["if_primary_term"] = strconv.FormatInt(int64(*r.IfPrimaryTerm), 10)
	}

	if r.IfSeqNo != nil {
		params["if_seq_no"] = strconv.FormatInt(int64(*r.IfSeqNo), 10)
	}

	if r.Lang != "" {
		params["lang"] = r.Lang
	}

	if r.Refresh != "" {
		params["refresh"] = r.Refresh
	}

	if r.RequireAlias != nil {
		params["require_alias"] = strconv.FormatBool(*r.RequireAlias)
	}

	if r.RetryOnConflict != nil {
		params["retry_on_conflict"] = strconv.FormatInt(int64(*r.RetryOnConflict), 10)
	}

	if r.Routing != "" {
		params["routing"] = r.Routing
	}

	if source, ok := r.Source.(bool); ok {
		params["_source"] = strconv.FormatBool(source)
	} else if source, ok := r.Source.(string); ok && source != "" {
		params["_source"] = source
	} else if sources, ok := r.Source.([]string); ok && len(sources) > 0 {
		params["_source"] = strings.Join(sources, ",")
	}

	if len(r.SourceExcludes) > 0 {
		params["_source_excludes"] = strings.Join(r.SourceExcludes, ",")
	}

	if len(r.SourceIncludes) > 0 {
		params["_source_includes"] = strings.Join(r.SourceIncludes, ",")
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
	}

	if r.WaitForActiveShards != "" {
		params["wait_for_active_shards"] = r.WaitForActiveShards
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

	return params
}
