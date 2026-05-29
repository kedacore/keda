// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// IngestGrokReq represents possible options for the index create request
type IngestGrokReq struct {
	Header http.Header
	Params IngestGrokParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IngestGrokReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_ingest/processor/grok",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// IngestGrokResp represents the returned struct of the index create response
type IngestGrokResp struct {
	Patterns map[string]string `json:"patterns"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IngestGrokResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
