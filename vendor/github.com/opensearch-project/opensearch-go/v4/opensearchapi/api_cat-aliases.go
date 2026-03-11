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

// CatAliasesReq represent possible options for the /_cat/aliases request
type CatAliasesReq struct {
	Aliases []string
	Header  http.Header
	Params  CatAliasesParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatAliasesReq) GetRequest() (*http.Request, error) {
	aliases := strings.Join(r.Aliases, ",")
	var path strings.Builder
	path.Grow(len("/_cat/aliases/") + len(aliases))
	path.WriteString("/_cat/aliases")
	if len(r.Aliases) > 0 {
		path.WriteString("/")
		path.WriteString(aliases)
	}
	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatAliasesResp represents the returned struct of the /_cat/aliases response
type CatAliasesResp struct {
	Aliases  []CatAliasResp
	response *opensearch.Response
}

// CatAliasResp represents one index of the CatAliasesResp
type CatAliasResp struct {
	Alias         string `json:"alias"`
	Index         string `json:"index"`
	Filter        string `json:"filter"`
	RoutingIndex  string `json:"routing.index"`
	RoutingSearch string `json:"routing.search"`
	IsWriteIndex  string `json:"is_write_index"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatAliasesResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
