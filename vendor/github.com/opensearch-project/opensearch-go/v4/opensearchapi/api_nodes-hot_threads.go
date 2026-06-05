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

// NodesHotThreadsReq represents possible options for the /_nodes request
type NodesHotThreadsReq struct {
	NodeID []string

	Header http.Header
	Params NodesHotThreadsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r NodesHotThreadsReq) GetRequest() (*http.Request, error) {
	nodes := strings.Join(r.NodeID, ",")

	var path strings.Builder
	path.Grow(len("/_nodes//hot_threads") + len(nodes))
	path.WriteString("/_nodes/")
	if len(r.NodeID) > 0 {
		path.WriteString(nodes)
		path.WriteString("/")
	}
	path.WriteString("hot_threads")

	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}
