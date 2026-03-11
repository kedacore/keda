// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"io"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// ScriptPutReq represents possible options for the put script request
type ScriptPutReq struct {
	ScriptID      string
	ScriptContext string

	Body io.Reader

	Header http.Header
	Params ScriptPutParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ScriptPutReq) GetRequest() (*http.Request, error) {
	var path strings.Builder
	path.Grow(len("/_scripts//") + len(r.ScriptID) + len(r.ScriptContext))
	path.WriteString("/_scripts/")
	path.WriteString(r.ScriptID)
	if r.ScriptContext != "" {
		path.WriteString("/")
		path.WriteString(r.ScriptContext)
	}

	return opensearch.BuildRequest(
		"PUT",
		path.String(),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// ScriptPutResp represents the returned struct of the put script response
type ScriptPutResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ScriptPutResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
