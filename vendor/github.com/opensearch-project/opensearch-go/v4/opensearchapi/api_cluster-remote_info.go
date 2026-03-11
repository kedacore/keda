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

// ClusterRemoteInfoReq represents possible options for the /_remote/info request
type ClusterRemoteInfoReq struct {
	Header http.Header
	Params ClusterRemoteInfoParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterRemoteInfoReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_remote/info",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ClusterRemoteInfoResp represents the returned struct of the ClusterRemoteInfoReq response
type ClusterRemoteInfoResp struct {
	Clusters map[string]ClusterRemoteInfoDetails
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterRemoteInfoResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// ClusterRemoteInfoDetails is a sub type of ClusterRemoteInfoResp contains information about a remote connection
type ClusterRemoteInfoDetails struct {
	Connected                bool     `json:"connected"`
	Mode                     string   `json:"mode"`
	Seeds                    []string `json:"seeds"`
	NumNodesConnected        int      `json:"num_nodes_connected"`
	MaxConnectionsPerCluster int      `json:"max_connections_per_cluster"`
	InitialConnectTimeout    string   `json:"initial_connect_timeout"`
	SkipUnavailable          bool     `json:"skip_unavailable"`
}
