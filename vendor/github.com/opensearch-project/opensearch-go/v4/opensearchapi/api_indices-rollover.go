// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"io"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// IndicesRolloverReq represents possible options for the index shrink request
type IndicesRolloverReq struct {
	Alias string
	Index string

	Body io.Reader

	Header http.Header
	Params IndicesRolloverParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesRolloverReq) GetRequest() (*http.Request, error) {
	var path strings.Builder
	path.Grow(12 + len(r.Alias) + len(r.Index))
	path.WriteString("/")
	path.WriteString(r.Alias)
	path.WriteString("/_rollover")
	if len(r.Index) > 0 {
		path.WriteString("/")
		path.WriteString(r.Index)
	}
	return opensearch.BuildRequest(
		"POST",
		path.String(),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// IndicesRolloverResp represents the returned struct of the index shrink response
type IndicesRolloverResp struct {
	Acknowledged       bool            `json:"acknowledged"`
	ShardsAcknowledged bool            `json:"shards_acknowledged"`
	OldIndex           string          `json:"old_index"`
	NewIndex           string          `json:"new_index"`
	RolledOver         bool            `json:"rolled_over"`
	DryRun             bool            `json:"dry_run"`
	Conditions         map[string]bool `json:"conditions"`
	response           *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesRolloverResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
