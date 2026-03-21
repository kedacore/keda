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

// IndicesShardStoresReq represents possible options for the index shrink request
type IndicesShardStoresReq struct {
	Indices []string

	Header http.Header
	Params IndicesShardStoresParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesShardStoresReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(15 + len(indices))
	if len(indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	path.WriteString("/_shard_stores")
	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// IndicesShardStoresResp represents the returned struct of the index shrink response
type IndicesShardStoresResp struct {
	Indices map[string]struct {
		Shards map[string]struct {
			Stores []json.RawMessage `json:"stores"`
		} `json:"shards"`
	} `json:"indices"`
	Failures []FailuresShard `json:"failures"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesShardStoresResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
