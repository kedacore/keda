// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// ClusterHealthReq represents possible options for the /_cluster/health request
type ClusterHealthReq struct {
	Indices []string
	Header  http.Header
	Params  ClusterHealthParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterHealthReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(17 + len(indices))
	path.WriteString("/_cluster/health")
	if len(indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}

	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ClusterHealthResp represents the returned struct of the ClusterHealthReq response
type ClusterHealthResp struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	DiscoveredMaster            bool    `json:"discovered_master"`
	DiscoveredClusterManager    bool    `json:"discovered_cluster_manager"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
	response                    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterHealthResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
