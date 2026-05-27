// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// SnapshotCreateReq represents possible options for the index create request
type SnapshotCreateReq struct {
	Repo     string
	Snapshot string

	Body io.Reader

	Header http.Header
	Params SnapshotCreateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r SnapshotCreateReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"PUT",
		fmt.Sprintf("/_snapshot/%s/%s", r.Repo, r.Snapshot),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// SnapshotCreateResp represents the returned struct of the index create response
type SnapshotCreateResp struct {
	Accepted bool `json:"accepted"`
	Snapshot struct {
		Snapshot                    string            `json:"snapshot"`
		UUID                        string            `json:"uuid"`
		VersionID                   int               `json:"version_id"`
		Version                     string            `json:"version"`
		RemoteStoreIndexShallowCopy bool              `json:"remote_store_index_shallow_copy"`
		Indices                     []string          `json:"indices"`
		DataStreams                 []json.RawMessage `json:"data_streams"`
		IncludeGlobalState          bool              `json:"include_global_state"`
		Metadata                    map[string]string `json:"metadata"`
		State                       string            `json:"state"`
		StartTime                   string            `json:"start_time"`
		StartTimeInMillis           int64             `json:"start_time_in_millis"`
		EndTime                     string            `json:"end_time"`
		EndTimeInMillis             int64             `json:"end_time_in_millis"`
		DurationInMillis            int               `json:"duration_in_millis"`
		Failures                    []json.RawMessage `json:"failures"`
		Shards                      struct {
			Total      int `json:"total"`
			Failed     int `json:"failed"`
			Successful int `json:"successful"`
		} `json:"shards"`
	} `json:"snapshot"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r SnapshotCreateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
