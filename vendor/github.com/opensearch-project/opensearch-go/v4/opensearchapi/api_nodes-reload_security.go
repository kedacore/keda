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

// NodesReloadSecurityReq represents possible options for the /_nodes request
type NodesReloadSecurityReq struct {
	NodeID []string

	Body io.Reader

	Header http.Header
	Params NodesReloadSecurityParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r NodesReloadSecurityReq) GetRequest() (*http.Request, error) {
	nodes := strings.Join(r.NodeID, ",")

	var path strings.Builder

	path.Grow(len("/_nodes//reload_secure_settings") + len(nodes))

	path.WriteString("/_nodes")
	if len(r.NodeID) > 0 {
		path.WriteString("/")
		path.WriteString(nodes)
	}
	path.WriteString("/reload_secure_settings")

	return opensearch.BuildRequest(
		"POST",
		path.String(),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// NodesReloadSecurityResp represents the returned struct of the /_nodes response
type NodesReloadSecurityResp struct {
	NodesInfo struct {
		Total      int             `json:"total"`
		Successful int             `json:"successful"`
		Failed     int             `json:"failed"`
		Failures   []FailuresCause `json:"failures"`
	} `json:"_nodes"`
	ClusterName string `json:"cluster_name"`
	Nodes       map[string]struct {
		Name string `json:"name"`
	} `json:"nodes"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r NodesReloadSecurityResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
