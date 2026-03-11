// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// IndicesForcemergeReq represents possible options for the <index>/_forcemerge request
type IndicesForcemergeReq struct {
	Indices []string

	Header http.Header
	Params IndicesForcemergeParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesForcemergeReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(len("//_forcemerge") + len(indices))
	if len(indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	path.WriteString("/_forcemerge")
	return opensearch.BuildRequest(
		"POST",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// IndicesForcemergeResp represents the returned struct of the flush indices response
type IndicesForcemergeResp struct {
	Shards struct {
		Total      int             `json:"total"`
		Successful int             `json:"successful"`
		Failed     int             `json:"failed"`
		Failures   []FailuresShard `json:"failures"`
	} `json:"_shards"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesForcemergeResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
