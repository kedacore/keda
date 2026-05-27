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

// CatClusterManagerReq represent possible options for the /_cat/cluster_manager request
type CatClusterManagerReq struct {
	Header http.Header
	Params CatClusterManagerParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatClusterManagerReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_cat/cluster_manager",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatClusterManagersResp represents the returned struct of the /_cat/cluster_manager response
type CatClusterManagersResp struct {
	ClusterManagers []CatClusterManagerResp
	response        *opensearch.Response
}

// CatClusterManagerResp represents one index of the CatClusterManagerResp
type CatClusterManagerResp struct {
	ID   string `json:"id"`
	Host string `json:"host"`
	IP   string `json:"ip"`
	Node string `json:"node"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatClusterManagersResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
