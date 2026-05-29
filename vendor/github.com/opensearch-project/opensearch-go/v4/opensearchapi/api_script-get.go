// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"fmt"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// ScriptGetReq represents possible options for the get script request
type ScriptGetReq struct {
	ScriptID string

	Header http.Header
	Params ScriptGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ScriptGetReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		fmt.Sprintf("/_scripts/%s", r.ScriptID),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ScriptGetResp represents the returned struct of the get script response
type ScriptGetResp struct {
	ID     string `json:"_id"`
	Found  bool   `json:"found"`
	Script struct {
		Lang   string `json:"lang"`
		Source string `json:"source"`
	} `json:"script"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ScriptGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
