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
)

// TermvectorsParams represents possible parameters for the TermvectorsReq
type TermvectorsParams struct {
	Fields          []string
	FieldStatistics *bool
	Offsets         *bool
	Payloads        *bool
	Positions       *bool
	Preference      string
	Realtime        *bool
	Routing         string
	TermStatistics  *bool
	Version         *int
	VersionType     string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string
}

func (r TermvectorsParams) get() map[string]string {
	params := make(map[string]string)

	if len(r.Fields) > 0 {
		params["fields"] = strings.Join(r.Fields, ",")
	}

	if r.FieldStatistics != nil {
		params["field_statistics"] = strconv.FormatBool(*r.FieldStatistics)
	}

	if r.Offsets != nil {
		params["offsets"] = strconv.FormatBool(*r.Offsets)
	}

	if r.Payloads != nil {
		params["payloads"] = strconv.FormatBool(*r.Payloads)
	}

	if r.Positions != nil {
		params["positions"] = strconv.FormatBool(*r.Positions)
	}

	if r.Preference != "" {
		params["preference"] = r.Preference
	}

	if r.Realtime != nil {
		params["realtime"] = strconv.FormatBool(*r.Realtime)
	}

	if r.Routing != "" {
		params["routing"] = r.Routing
	}

	if r.TermStatistics != nil {
		params["term_statistics"] = strconv.FormatBool(*r.TermStatistics)
	}

	if r.Version != nil {
		params["version"] = strconv.FormatInt(int64(*r.Version), 10)
	}

	if r.VersionType != "" {
		params["version_type"] = r.VersionType
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
