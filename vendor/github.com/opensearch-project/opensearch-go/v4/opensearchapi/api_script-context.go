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

// ScriptContextReq represents possible options for the delete script request
type ScriptContextReq struct {
	Header http.Header
	Params ScriptContextParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ScriptContextReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_script_context",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ScriptContextResp represents the returned struct of the delete script response
type ScriptContextResp struct {
	Contexts []struct {
		Name    string `json:"name"`
		Methods []struct {
			Name       string `json:"name"`
			ReturnType string `json:"return_type"`
			Params     []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"params"`
		} `json:"methods"`
	} `json:"contexts"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ScriptContextResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
