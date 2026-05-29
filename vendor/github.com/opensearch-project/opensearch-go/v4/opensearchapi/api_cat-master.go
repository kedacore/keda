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

// CatMasterReq represent possible options for the /_cat/master request
type CatMasterReq struct {
	Header http.Header
	Params CatMasterParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatMasterReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_cat/master",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatMasterResp represents the returned struct of the /_cat/master response
type CatMasterResp struct {
	Master   []CatMasterItemResp
	response *opensearch.Response
}

// CatMasterItemResp represents one index of the CatMasterResp
type CatMasterItemResp struct {
	ID   string `json:"id"`
	Host string `json:"host"`
	IP   string `json:"ip"`
	Node string `json:"node"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatMasterResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
