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

// IndicesSplitReq represents possible options for the index split request
type IndicesSplitReq struct {
	Index  string
	Target string

	Body io.Reader

	Header http.Header
	Params IndicesSplitParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesSplitReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		fmt.Sprintf("/%s/_split/%s", r.Index, r.Target),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// IndicesSplitResp represents the returned struct of the index split response
type IndicesSplitResp struct {
	Acknowledged       bool   `json:"acknowledged"`
	ShardsAcknowledged bool   `json:"shards_acknowledged"`
	Index              string `json:"index"`
	response           *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesSplitResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
