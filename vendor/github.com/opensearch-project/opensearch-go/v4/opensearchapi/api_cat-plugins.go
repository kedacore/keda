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

// CatPluginsReq represent possible options for the /_cat/plugins request
type CatPluginsReq struct {
	Header http.Header
	Params CatPluginsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatPluginsReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_cat/plugins",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatPluginsResp represents the returned struct of the /_cat/plugins response
type CatPluginsResp struct {
	Plugins  []CatPluginResp
	response *opensearch.Response
}

// CatPluginResp represents one index of the CatPluginsResp
type CatPluginResp struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Component   string `json:"component,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatPluginsResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
