// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// DocumentSourceReq represents possible options for the /<Index>/_source/<DocumentID> get request
type DocumentSourceReq struct {
	Index      string
	DocumentID string

	Header http.Header
	Params DocumentSourceParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DocumentSourceReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		fmt.Sprintf("/%s/_source/%s", r.Index, r.DocumentID),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// DocumentSourceResp represents the returned struct of the /<Index>/_source/<DocumentID> get response
type DocumentSourceResp struct {
	Source   json.RawMessage
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DocumentSourceResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
