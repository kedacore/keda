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

// IndicesOpenReq represents possible options for the index open request
type IndicesOpenReq struct {
	Index string

	Header http.Header
	Params IndicesOpenParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesOpenReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		fmt.Sprintf("/%s/_open", r.Index),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// IndicesOpenResp represents the returned struct of the index open response
type IndicesOpenResp struct {
	Acknowledged       bool `json:"acknowledged"`
	ShardsAcknowledged bool `json:"shards_acknowledged"`
	response           *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesOpenResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
