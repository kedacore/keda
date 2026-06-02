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

// IndicesCreateReq represents possible options for the index create request
type IndicesCreateReq struct {
	Index  string
	Body   io.Reader
	Header http.Header
	Params IndicesCreateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesCreateReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"PUT",
		fmt.Sprintf("%s%s", "/", r.Index),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// IndicesCreateResp represents the returned struct of the index create response
type IndicesCreateResp struct {
	Acknowledged       bool   `json:"acknowledged"`
	ShardsAcknowledged bool   `json:"shards_acknowledged"`
	Index              string `json:"index"`
	response           *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesCreateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
