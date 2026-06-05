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

// NodesStatsParams represents possible parameters for the NodesStatsReq
type NodesStatsParams struct {
	CompletionFields        []string
	FielddataFields         []string
	Fields                  []string
	Groups                  *bool
	IncludeSegmentFileSizes *bool
	IncludeUnloadedSegments *bool
	Level                   string
	Timeout                 time.Duration
	Types                   []string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string
}

func (r NodesStatsParams) get() map[string]string {
	params := make(map[string]string)

	if len(r.CompletionFields) > 0 {
		params["completion_fields"] = strings.Join(r.CompletionFields, ",")
	}

	if len(r.FielddataFields) > 0 {
		params["fielddata_fields"] = strings.Join(r.FielddataFields, ",")
	}

	if len(r.Fields) > 0 {
		params["fields"] = strings.Join(r.Fields, ",")
	}

	if r.Groups != nil {
		params["groups"] = strconv.FormatBool(*r.Groups)
	}

	if r.IncludeSegmentFileSizes != nil {
		params["include_segment_file_sizes"] = strconv.FormatBool(*r.IncludeSegmentFileSizes)
	}

	if r.IncludeUnloadedSegments != nil {
		params["include_unloaded_segments"] = strconv.FormatBool(*r.IncludeUnloadedSegments)
	}

	if r.Level != "" {
		params["level"] = r.Level
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
	}

	if len(r.Types) > 0 {
		params["types"] = strings.Join(r.Types, ",")
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
