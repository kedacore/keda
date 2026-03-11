// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import "strings"

// PointInTimeDeleteParams represents possible parameters for the PointInTimeDeleteReq
type PointInTimeDeleteParams struct {
	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string
}

func (r PointInTimeDeleteParams) get() map[string]string {
	params := make(map[string]string)

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
