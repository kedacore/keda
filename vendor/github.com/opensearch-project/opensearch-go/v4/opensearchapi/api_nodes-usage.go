// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// NodesUsageReq represents possible options for the /_nodes request
type NodesUsageReq struct {
	Metrics []string
	NodeID  []string

	Header http.Header
	Params NodesUsageParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r NodesUsageReq) GetRequest() (*http.Request, error) {
	nodes := strings.Join(r.NodeID, ",")
	metrics := strings.Join(r.Metrics, ",")

	var path strings.Builder

	path.Grow(len("/_nodes//usage/") + len(nodes) + len(metrics))

	path.WriteString("/_nodes")
	if len(r.NodeID) > 0 {
		path.WriteString("/")
		path.WriteString(nodes)
	}
	path.WriteString("/usage")
	if len(r.Metrics) > 0 {
		path.WriteString("/")
		path.WriteString(metrics)
	}

	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// NodesUsageResp represents the returned struct of the /_nodes response
type NodesUsageResp struct {
	NodesUsage struct {
		Total      int             `json:"total"`
		Successful int             `json:"successful"`
		Failed     int             `json:"failed"`
		Failures   []FailuresCause `json:"failures"`
	} `json:"_nodes"`
	ClusterName string                `json:"cluster_name"`
	Nodes       map[string]NodesUsage `json:"nodes"`
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r NodesUsageResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// NodesUsage is a sub type of NodesUsageResp containing stats about rest api actions
type NodesUsage struct {
	Timestamp    int64           `json:"timestamp"`
	Since        int64           `json:"since"`
	RestActions  map[string]int  `json:"rest_actions"`
	Aggregations json.RawMessage `json:"aggregations"` // Can contain unknow fields
}
