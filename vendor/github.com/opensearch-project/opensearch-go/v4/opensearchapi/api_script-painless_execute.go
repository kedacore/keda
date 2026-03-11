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

	"github.com/opensearch-project/opensearch-go/v4"
)

// ScriptPainlessExecuteReq represents possible options for the delete script request
type ScriptPainlessExecuteReq struct {
	Body io.Reader

	Header http.Header
	Params ScriptPainlessExecuteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ScriptPainlessExecuteReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		"/_scripts/painless/_execute",
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// ScriptPainlessExecuteResp represents the returned struct of the delete script response
type ScriptPainlessExecuteResp struct {
	Result   json.RawMessage `json:"result"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ScriptPainlessExecuteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
