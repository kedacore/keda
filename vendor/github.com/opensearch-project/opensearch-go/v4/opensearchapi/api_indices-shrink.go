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

// IndicesShrinkReq represents possible options for the index shrink request
type IndicesShrinkReq struct {
	Index  string
	Target string

	Body io.Reader

	Header http.Header
	Params IndicesShrinkParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesShrinkReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		fmt.Sprintf("/%s/_shrink/%s", r.Index, r.Target),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// IndicesShrinkResp represents the returned struct of the index shrink response
type IndicesShrinkResp struct {
	Acknowledged       bool   `json:"acknowledged"`
	ShardsAcknowledged bool   `json:"shards_acknowledged"`
	Index              string `json:"index"`
	response           *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesShrinkResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
