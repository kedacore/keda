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

// DocumentDeleteReq represents possible options for the /<index>/_doc/<DocID> delete request
type DocumentDeleteReq struct {
	Index      string
	DocumentID string

	Header http.Header
	Params DocumentDeleteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DocumentDeleteReq) GetRequest() (*http.Request, error) {
	var path strings.Builder
	path.Grow(7 + len(r.Index) + len(r.DocumentID))
	path.WriteString("/")
	path.WriteString(r.Index)
	path.WriteString("/_doc/")
	path.WriteString(r.DocumentID)
	return opensearch.BuildRequest(
		"DELETE",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// DocumentDeleteResp represents the returned struct of the /<index>/_doc/<DocID> response
type DocumentDeleteResp struct {
	Index   string `json:"_index"`
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	Result  string `json:"result"`
	Type    string `json:"_type"` // Deprecated field
	Shards  struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	SeqNo       int `json:"_seq_no"`
	PrimaryTerm int `json:"_primary_term"`
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DocumentDeleteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
