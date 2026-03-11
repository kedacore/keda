// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"fmt"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// CatSnapshotsReq represent possible options for the /_cat/snapshots request
type CatSnapshotsReq struct {
	Repository string
	Header     http.Header
	Params     CatSnapshotsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatSnapshotsReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		fmt.Sprintf("%s%s", "/_cat/snapshots/", r.Repository),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatSnapshotsResp represents the returned struct of the /_cat/snapshots response
type CatSnapshotsResp struct {
	Snapshots []CatSnapshotResp
	response  *opensearch.Response
}

// CatSnapshotResp represents one index of the CatSnapshotsResp
type CatSnapshotResp struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	StartEpoch       int    `json:"start_epoch,string"`
	StartTime        string `json:"start_time"`
	EndEpoch         int    `json:"end_epoch,string"`
	EndTime          string `json:"end_time"`
	Duration         string `json:"duration"`
	Indices          int    `json:"indices,string"`
	SuccessfulShards int    `json:"successful_shards,string"`
	FailedShards     int    `json:"failed_shards,string"`
	TotalShards      int    `json:"total_shards,string"`
	Reason           string `json:"reason"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatSnapshotsResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
