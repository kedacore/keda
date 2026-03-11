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

// SnapshotRepositoryGetReq represents possible options for the index create request
type SnapshotRepositoryGetReq struct {
	Repos []string

	Header http.Header
	Params SnapshotRepositoryGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r SnapshotRepositoryGetReq) GetRequest() (*http.Request, error) {
	repos := strings.Join(r.Repos, ",")

	var path strings.Builder
	path.Grow(len("/_snapshot/") + len(repos))
	path.WriteString("/_snapshot")
	if len(r.Repos) > 0 {
		path.WriteString("/")
		path.WriteString(repos)
	}

	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// SnapshotRepositoryGetResp represents the returned struct of the index create response
type SnapshotRepositoryGetResp struct {
	Repos map[string]struct {
		Type     string            `json:"type"`
		Settings map[string]string `json:"settings"`
	}
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r SnapshotRepositoryGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
