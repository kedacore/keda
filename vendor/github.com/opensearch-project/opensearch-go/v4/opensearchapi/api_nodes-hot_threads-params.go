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

// NodesHotThreadsParams represents possible parameters for the NodesHotThreadsReq
type NodesHotThreadsParams struct {
	IgnoreIdleThreads *bool
	Interval          time.Duration
	Snapshots         *int
	Threads           *int
	Timeout           time.Duration

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string
}

func (r NodesHotThreadsParams) get() map[string]string {
	params := make(map[string]string)

	if r.IgnoreIdleThreads != nil {
		params["ignore_idle_threads"] = strconv.FormatBool(*r.IgnoreIdleThreads)
	}

	if r.Interval != 0 {
		params["interval"] = formatDuration(r.Interval)
	}

	if r.Snapshots != nil {
		params["snapshots"] = strconv.FormatInt(int64(*r.Snapshots), 10)
	}

	if r.Threads != nil {
		params["threads"] = strconv.FormatInt(int64(*r.Threads), 10)
	}

	if r.Timeout != 0 {
		params["timeout"] = formatDuration(r.Timeout)
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
