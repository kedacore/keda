// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"fmt"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// IndicesCloneReq represents possible options for the index clone request
type IndicesCloneReq struct {
	Index  string
	Target string

	Body io.Reader

	Header http.Header
	Params IndicesCloneParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesCloneReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		fmt.Sprintf("/%s/_clone/%s", r.Index, r.Target),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// IndicesCloneResp represents the returned struct of the index clone response
type IndicesCloneResp struct {
	Acknowledged       bool   `json:"acknowledged"`
	ShardsAcknowledged bool   `json:"shards_acknowledged"`
	Index              string `json:"index"`
	response           *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesCloneResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
