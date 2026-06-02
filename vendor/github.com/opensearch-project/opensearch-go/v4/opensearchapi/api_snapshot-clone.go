// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"fmt"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// SnapshotCloneReq represents possible options for the index create request
type SnapshotCloneReq struct {
	Repo           string
	Snapshot       string
	TargetSnapshot string

	Body io.Reader

	Header http.Header
	Params SnapshotCloneParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r SnapshotCloneReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"PUT",
		fmt.Sprintf("/_snapshot/%s/%s/_clone/%s", r.Repo, r.Snapshot, r.TargetSnapshot),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// SnapshotCloneResp represents the returned struct of the index create response
type SnapshotCloneResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r SnapshotCloneResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
