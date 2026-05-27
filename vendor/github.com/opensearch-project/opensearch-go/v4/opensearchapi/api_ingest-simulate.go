// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// IngestSimulateReq represents possible options for the index create request
type IngestSimulateReq struct {
	PipelineID string

	Body io.Reader

	Header http.Header
	Params IngestSimulateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IngestSimulateReq) GetRequest() (*http.Request, error) {
	var path strings.Builder
	path.Grow(len("/_ingest/pipeline//_simulate") + len(r.PipelineID))
	path.WriteString("/_ingest/pipeline/")
	if len(r.PipelineID) > 0 {
		path.WriteString(r.PipelineID)
		path.WriteString("/")
	}
	path.WriteString("_simulate")
	return opensearch.BuildRequest(
		"POST",
		path.String(),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// IngestSimulateResp represents the returned struct of the index create response
type IngestSimulateResp struct {
	Docs     []json.RawMessage `json:"docs"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IngestSimulateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
