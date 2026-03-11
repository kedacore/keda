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

// IndicesRefreshReq represents possible options for the <index>/_refresh request
type IndicesRefreshReq struct {
	Indices []string

	Header http.Header
	Params IndicesRefreshParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesRefreshReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(len("//_refresh") + len(indices))
	if len(indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	path.WriteString("/_refresh")
	return opensearch.BuildRequest(
		"POST",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// IndicesRefreshResp represents the returned struct of the index shrink response
type IndicesRefreshResp struct {
	Shards struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesRefreshResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
