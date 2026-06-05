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

// ClusterStateReq represents possible options for the /_cluster/state request
type ClusterStateReq struct {
	Metrics []string
	Indices []string

	Header http.Header
	Params ClusterStateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterStateReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")
	metrics := strings.Join(r.Metrics, ",")

	var path strings.Builder
	path.Grow(17 + len(indices) + len(metrics))
	path.WriteString("/_cluster/state")
	if len(metrics) > 0 {
		path.WriteString("/")
		path.WriteString(metrics)
		if len(indices) > 0 {
			path.WriteString("/")
			path.WriteString(indices)
		}
	}

	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ClusterStateResp represents the returned struct of the ClusterStateReq response
type ClusterStateResp struct {
	ClusterName        string `json:"cluster_name"`
	ClusterUUID        string `json:"cluster_uuid"`
	Version            int    `json:"version"`
	StateUUID          string `json:"state_uuid"`
	MasterNode         string `json:"master_node"`
	ClusterManagerNode string `json:"cluster_manager_node"`
	Blocks             struct {
		Indices map[string]map[string]ClusterStateBlocksIndex `json:"indices"`
	} `json:"blocks"`
	Nodes        map[string]ClusterStateNodes `json:"nodes"`
	Metadata     ClusterStateMetaData         `json:"metadata"`
	response     *opensearch.Response
	RoutingTable struct {
		Indices map[string]struct {
			Shards map[string][]ClusterStateRoutingIndex `json:"shards"`
		} `json:"indices"`
	} `json:"routing_table"`
	RoutingNodes ClusterStateRoutingNodes `json:"routing_nodes"`
	Snapshots    struct {
		Snapshots []json.RawMessage `json:"snapshots"`
	} `json:"snapshots"`
	SnapshotDeletions struct {
		SnapshotDeletions []json.RawMessage `json:"snapshot_deletions"`
	} `json:"snapshot_deletions"`
	RepositoryCleanup struct {
		RepositoryCleanup []json.RawMessage `json:"repository_cleanup"`
	} `json:"repository_cleanup"`
	Restore struct {
		Snapshots []json.RawMessage `json:"snapshots"`
	} `json:"restore"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterStateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// ClusterStateBlocksIndex is a sub type of ClusterStateResp
type ClusterStateBlocksIndex struct {
	Description string   `json:"description"`
	Retryable   bool     `json:"retryable"`
	Levels      []string `json:"levels"`
}

// ClusterStateNodes is a sub type of ClusterStateResp
type ClusterStateNodes struct {
	Name             string            `json:"name"`
	EphemeralID      string            `json:"ephemeral_id"`
	TransportAddress string            `json:"transport_address"`
	Attributes       map[string]string `json:"attributes"`
}

// ClusterStateMetaData is a sub type if ClusterStateResp containing metadata of the cluster
type ClusterStateMetaData struct {
	ClusterUUID          string `json:"cluster_uuid"`
	ClusterUUIDCommitted bool   `json:"cluster_uuid_committed"`
	ClusterCoordination  struct {
		Term                   int      `json:"term"`
		LastCommittedConfig    []string `json:"last_committed_config"`
		LastAcceptedConfig     []string `json:"last_accepted_config"`
		VotingConfigExclusions []struct {
			NodeID   string `json:"node_id"`
			NodeName string `json:"node_name"`
		} `json:"voting_config_exclusions"`
	} `json:"cluster_coordination"`
	Templates      map[string]json.RawMessage           `json:"templates"`
	Indices        map[string]ClusterStateMetaDataIndex `json:"indices"`
	IndexGraveyard struct {
		Tombstones []struct {
			Index struct {
				IndexName string `json:"index_name"`
				IndexUUID string `json:"index_uuid"`
			} `json:"index"`
			DeleteDateInMillis int `json:"delete_date_in_millis"`
		} `json:"tombstones"`
	} `json:"index-graveyard"`
	Repositories map[string]struct {
		Type              string            `json:"type"`
		Settings          map[string]string `json:"settings"`
		Generation        int               `json:"generation"`
		PendingGeneration int               `json:"pending_generation"`
	} `json:"repositories"`
	ComponentTemplate struct {
		ComponentTemplate map[string]json.RawMessage `json:"component_template"`
	} `json:"component_template"`
	IndexTemplate struct {
		IndexTemplate map[string]json.RawMessage `json:"index_template"`
	} `json:"index_template"`
	StoredScripts map[string]struct {
		Lang   string `json:"lang"`
		Source string `json:"source"`
	} `json:"stored_scripts"`
	Ingest struct {
		Pipeline []struct {
			ID     string `json:"id"`
			Config struct {
				Description string          `json:"description"`
				Processors  json.RawMessage `json:"processors"`
			} `json:"config"`
		} `json:"pipeline"`
	} `json:"ingest"`
	DataStream struct {
		DataStream map[string]ClusterStateMetaDataStream `json:"data_stream"`
	} `json:"data_stream"`
}

// ClusterStateMetaDataIndex is a sub type of ClusterStateMetaData containing information about an index
type ClusterStateMetaDataIndex struct {
	Version           int                 `json:"version"`
	MappingVersion    int                 `json:"mapping_version"`
	SettingsVersion   int                 `json:"settings_version"`
	AliasesVersion    int                 `json:"aliases_version"`
	RoutingNumShards  int                 `json:"routing_num_shards"`
	State             string              `json:"state"`
	Settings          json.RawMessage     `json:"settings"`
	Mappings          json.RawMessage     `json:"mappings"`
	Aliases           []string            `json:"aliases"`
	PrimaryTerms      map[string]int      `json:"primary_terms"`
	InSyncAllocations map[string][]string `json:"in_sync_allocations"`
	RolloverInfo      map[string]struct {
		MetConditions map[string]string `json:"met_conditions"`
		Time          int               `json:"time"`
	} `json:"rollover_info"`
	System bool `json:"system"`
}

// ClusterStateMetaDataStream is a sub type of ClusterStateMetaData containing information about a data stream
type ClusterStateMetaDataStream struct {
	Name           string `json:"name"`
	TimestampField struct {
		Name string `json:"name"`
	} `json:"timestamp_field"`
	Indices []struct {
		IndexName string `json:"index_name"`
		IndexUUID string `json:"index_uuid"`
	} `json:"indices"`
	Generation int `json:"generation"`
}

// ClusterStateRoutingIndex is a sub type of ClusterStateResp and ClusterStateRoutingNodes containing information about shard routing
type ClusterStateRoutingIndex struct {
	State                    string  `json:"state"`
	Primary                  bool    `json:"primary"`
	SearchOnly               bool    `json:"searchOnly"`
	Node                     *string `json:"node"`
	RelocatingNode           *string `json:"relocating_node"`
	Shard                    int     `json:"shard"`
	Index                    string  `json:"index"`
	ExpectedShardSizeInBytes int     `json:"expected_shard_size_in_bytes"`
	AllocationID             *struct {
		ID string `json:"id"`
	} `json:"allocation_id,omitempty"`
	RecoverySource *struct {
		Type string `json:"type"`
	} `json:"recovery_source,omitempty"`
	UnassignedInfo *struct {
		Reason           string `json:"reason"`
		At               string `json:"at"`
		Delayed          bool   `json:"delayed"`
		AllocationStatus string `json:"allocation_status"`
		Details          string `json:"details"`
	} `json:"unassigned_info,omitempty"`
}

// ClusterStateRoutingNodes is a sub type of ClusterStateResp containing information about shard assigned to nodes
type ClusterStateRoutingNodes struct {
	Unassigned []ClusterStateRoutingIndex            `json:"unassigned"`
	Nodes      map[string][]ClusterStateRoutingIndex `json:"nodes"`
}
