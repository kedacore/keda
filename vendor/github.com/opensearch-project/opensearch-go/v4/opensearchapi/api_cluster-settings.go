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

// ClusterGetSettingsReq represents possible options for the /_cluster/settings request
type ClusterGetSettingsReq struct {
	Header http.Header
	Params ClusterGetSettingsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterGetSettingsReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_cluster/settings",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ClusterGetSettingsResp represents the returned struct of the ClusterGetSettingsReq response
type ClusterGetSettingsResp struct {
	Persistent json.RawMessage `json:"persistent"`
	Transient  json.RawMessage `json:"transient"`
	Defaults   json.RawMessage `json:"defaults"`
	response   *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterGetSettingsResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// ClusterPutSettingsReq represents possible options for the /_cluster/settings request
type ClusterPutSettingsReq struct {
	Body   io.Reader
	Header http.Header
	Params ClusterPutSettingsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterPutSettingsReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"PUT",
		"/_cluster/settings",
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// ClusterPutSettingsResp represents the returned struct of the /_cluster/settings response
type ClusterPutSettingsResp struct {
	Acknowledged bool            `json:"acknowledged"`
	Persistent   json.RawMessage `json:"persistent"`
	Transient    json.RawMessage `json:"transient"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterPutSettingsResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
