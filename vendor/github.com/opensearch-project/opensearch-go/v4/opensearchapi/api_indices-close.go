// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"fmt"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// IndicesCloseReq represents possible options for the index close request
type IndicesCloseReq struct {
	Index string

	Header http.Header
	Params IndicesCloseParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesCloseReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		fmt.Sprintf("/%s/_close", r.Index),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// IndicesCloseResp represents the returned struct of the index close response
type IndicesCloseResp struct {
	Acknowledged       bool `json:"acknowledged"`
	ShardsAcknowledged bool `json:"shards_acknowledged"`
	Indices            map[string]struct {
		Closed       bool `json:"closed"`
		FailedShards map[string]struct {
			Failures []FailuresShard `json:"failures"`
		} `json:"failed_shards"`
	} `json:"indices"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesCloseResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
