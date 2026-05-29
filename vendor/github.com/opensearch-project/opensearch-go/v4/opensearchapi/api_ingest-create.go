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

// IngestCreateReq represents possible options for the index create request
type IngestCreateReq struct {
	PipelineID string

	Body io.Reader

	Header http.Header
	Params IngestCreateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IngestCreateReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"PUT",
		fmt.Sprintf("/_ingest/pipeline/%s", r.PipelineID),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// IngestCreateResp represents the returned struct of the index create response
type IngestCreateResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IngestCreateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
