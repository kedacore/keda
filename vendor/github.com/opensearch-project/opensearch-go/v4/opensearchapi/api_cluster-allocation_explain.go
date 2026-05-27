// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// ClusterAllocationExplainReq represents possible options for the /_nodes request
type ClusterAllocationExplainReq struct {
	Body   *ClusterAllocationExplainBody
	Header http.Header
	Params ClusterAllocationExplainParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterAllocationExplainReq) GetRequest() (*http.Request, error) {
	var reader io.Reader

	if r.Body != nil {
		body, err := json.Marshal(r.Body)
		if err != nil {
			return nil, err
		}

		reader = bytes.NewReader(body)
	}

	return opensearch.BuildRequest(
		"GET",
		"/_cluster/allocation/explain",
		reader,
		r.Params.get(),
		r.Header,
	)
}

// ClusterAllocationExplainBody represents the optional Body for the ClusterAllocationExplainReq
type ClusterAllocationExplainBody struct {
	Index   string `json:"index"`
	Shard   int    `json:"shard"`
	Primary bool   `json:"primary"`
}

// ClusterAllocationExplainResp represents the returned struct of the /_nodes response
type ClusterAllocationExplainResp struct {
	Index          string                       `json:"index"`
	Shard          int                          `json:"shard"`
	Primary        bool                         `json:"primary"`
	CurrentState   string                       `json:"current_state"`
	CurrentNode    ClusterAllocationCurrentNode `json:"current_node"`
	UnassignedInfo struct {
		Reason               string `json:"reason"`
		At                   string `json:"at"`
		LastAllocationStatus string `json:"last_allocation_status"`
	} `json:"unassigned_info"`
	CanAllocate                  string                             `json:"can_allocate"`
	CanRemainOnCurrentNode       string                             `json:"can_remain_on_current_node"`
	CanRebalanceCluster          string                             `json:"can_rebalance_cluster"`
	CanRebalanceToOtherNode      string                             `json:"can_rebalance_to_other_node"`
	RebalanceExplanation         string                             `json:"rebalance_explanation"`
	AllocateExplanation          string                             `json:"allocate_explanation"`
	NodeAllocationDecisions      []ClusterAllocationNodeDecisions   `json:"node_allocation_decisions"`
	CanRebalanceClusterDecisions []ClusterAllocationExplainDeciders `json:"can_rebalance_cluster_decisions"`
	response                     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterAllocationExplainResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// ClusterAllocationCurrentNode is a sub type of ClusterAllocationExplainResp containing information of the node the shard is on
type ClusterAllocationCurrentNode struct {
	NodeID           string `json:"id"`
	NodeName         string `json:"name"`
	TransportAddress string `json:"transport_address"`
	NodeAttributes   struct {
		ShardIndexingPressureEnabled string `json:"shard_indexing_pressure_enabled"`
	} `json:"attributes"`
	WeightRanking int `json:"weight_ranking"`
}

// ClusterAllocationNodeDecisions is a sub type of ClusterAllocationExplainResp containing information of a node allocation decission
type ClusterAllocationNodeDecisions struct {
	NodeID           string `json:"node_id"`
	NodeName         string `json:"node_name"`
	TransportAddress string `json:"transport_address"`
	NodeAttributes   struct {
		ShardIndexingPressureEnabled string `json:"shard_indexing_pressure_enabled"`
	} `json:"node_attributes"`
	NodeDecision  string                             `json:"node_decision"`
	WeightRanking int                                `json:"weight_ranking"`
	Deciders      []ClusterAllocationExplainDeciders `json:"deciders"`
}

// ClusterAllocationExplainDeciders is a sub type of ClusterAllocationExplainResp and
// ClusterAllocationNodeDecisions containing inforamtion about Deciders decissions
type ClusterAllocationExplainDeciders struct {
	Decider     string `json:"decider"`
	Decision    string `json:"decision"`
	Explanation string `json:"explanation"`
}
