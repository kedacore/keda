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
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// IngestGetReq represents possible options for the index create request
type IngestGetReq struct {
	PipelineIDs []string

	Header http.Header
	Params IngestGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IngestGetReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		fmt.Sprintf("/_ingest/pipeline/%s", strings.Join(r.PipelineIDs, ",")),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// IngestGetResp represents the returned struct of the index create response
type IngestGetResp struct {
	Pipelines map[string]struct {
		Description string                       `json:"description"`
		Processors  []map[string]json.RawMessage `json:"processors"`
	}
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IngestGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
