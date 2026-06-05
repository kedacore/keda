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

// TemplateCreateParams represents possible parameters for the TemplateCreateReq
type TemplateCreateParams struct {
	Create                *bool
	MasterTimeout         time.Duration
	ClusterManagerTimeout time.Duration
	Order                 *int

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string
}

func (r TemplateCreateParams) get() map[string]string {
	params := make(map[string]string)

	if r.Create != nil {
		params["create"] = strconv.FormatBool(*r.Create)
	}

	if r.MasterTimeout != 0 {
		params["master_timeout"] = formatDuration(r.MasterTimeout)
	}

	if r.ClusterManagerTimeout != 0 {
		params["cluster_manager_timeout"] = formatDuration(r.ClusterManagerTimeout)
	}

	if r.Order != nil {
		params["order"] = strconv.FormatInt(int64(*r.Order), 10)
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
