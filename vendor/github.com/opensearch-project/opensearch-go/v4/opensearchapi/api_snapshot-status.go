// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// SnapshotStatusReq represents possible options for the index create request
type SnapshotStatusReq struct {
	Repo      string
	Snapshots []string

	Header http.Header
	Params SnapshotStatusParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r SnapshotStatusReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		fmt.Sprintf("/_snapshot/%s/%s/_status", r.Repo, strings.Join(r.Snapshots, ",")),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// SnapshotStatusResp represents the returned struct of the index create response
type SnapshotStatusResp struct {
	Accepted  bool `json:"accepted"`
	Snapshots []struct {
		Snapshot           string                    `json:"snapshot"`
		Repository         string                    `json:"repository"`
		UUID               string                    `json:"uuid"`
		State              string                    `json:"state"`
		IncludeGlobalState bool                      `json:"include_global_state"`
		ShardsStats        SnapshotStatusShardsStats `json:"shards_stats"`
		Stats              SnapshotStatusStats       `json:"stats"`
		Indices            map[string]struct {
			ShardsStats SnapshotStatusShardsStats `json:"shards_stats"`
			Stats       SnapshotStatusStats       `json:"stats"`
			Shards      map[string]struct {
				Stage string              `json:"stage"`
				Stats SnapshotStatusStats `json:"stats"`
			} `json:"shards"`
		} `json:"indices"`
	} `json:"snapshots"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r SnapshotStatusResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// SnapshotStatusShardsStats is a sub type of SnapshotStatusResp containing information about shard stats
type SnapshotStatusShardsStats struct {
	Initializing int `json:"initializing"`
	Started      int `json:"started"`
	Finalizing   int `json:"finalizing"`
	Done         int `json:"done"`
	Failed       int `json:"failed"`
	Total        int `json:"total"`
}

// SnapshotStatusStats is a sub type of SnapshotStatusResp containing information about snapshot stats
type SnapshotStatusStats struct {
	Incremental struct {
		FileCount   int   `json:"file_count"`
		SizeInBytes int64 `json:"size_in_bytes"`
	} `json:"incremental"`
	Total struct {
		FileCount   int   `json:"file_count"`
		SizeInBytes int64 `json:"size_in_bytes"`
	} `json:"total"`
	StartTimeInMillis int64 `json:"start_time_in_millis"`
	TimeInMillis      int   `json:"time_in_millis"`
}
