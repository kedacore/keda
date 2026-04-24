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

// IndicesClearCacheReq represents possible options for the index clear cache request
type IndicesClearCacheReq struct {
	Indices []string

	Header http.Header
	Params IndicesClearCacheParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesClearCacheReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(len("//_cache/clear") + len(indices))
	if len(indices) != 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	path.WriteString("/_cache/clear")

	return opensearch.BuildRequest(
		"POST",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// IndicesClearCacheResp represents the returned struct of the index clear cache response
type IndicesClearCacheResp struct {
	Shards struct {
		Total      int             `json:"total"`
		Successful int             `json:"successful"`
		Failed     int             `json:"failed"`
		Failures   []FailuresShard `json:"failures"`
	} `json:"_shards"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesClearCacheResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
