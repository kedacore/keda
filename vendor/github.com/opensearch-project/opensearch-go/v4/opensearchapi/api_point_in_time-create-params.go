// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"strings"
	"time"
)

// PointInTimeCreateParams represents possible parameters for the PointInTimeCreateReq
type PointInTimeCreateParams struct {
	KeepAlive               time.Duration
	Preference              string
	Routing                 string
	ExpandWildcards         string
	AllowPartialPitCreation bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string
}

func (r PointInTimeCreateParams) get() map[string]string {
	params := make(map[string]string)

	if r.KeepAlive != 0 {
		params["keep_alive"] = formatDuration(r.KeepAlive)
	}

	if r.Preference != "" {
		params["preference"] = r.Preference
	}

	if r.Routing != "" {
		params["routing"] = r.Routing
	}

	if r.ExpandWildcards != "" {
		params["expand_wildcards"] = r.ExpandWildcards
	}

	if r.AllowPartialPitCreation {
		params["allow_partial_pit_creation"] = "true"
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
