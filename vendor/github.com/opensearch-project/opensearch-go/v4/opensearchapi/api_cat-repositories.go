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

// CatRepositoriesReq represent possible options for the /_cat/repositories request
type CatRepositoriesReq struct {
	Header http.Header
	Params CatRepositoriesParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatRepositoriesReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_cat/repositories",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatRepositoriesResp represents the returned struct of the /_cat/repositories response
type CatRepositoriesResp struct {
	Repositories []CatRepositorieResp
	response     *opensearch.Response
}

// CatRepositorieResp represents one index of the CatRepositoriesResp
type CatRepositorieResp struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatRepositoriesResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
