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

// ScrollGetReq represents possible options for the index create request
type ScrollGetReq struct {
	ScrollID string

	Header http.Header
	Params ScrollGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ScrollGetReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		"/_search/scroll",
		strings.NewReader(fmt.Sprintf(`{"scroll_id":"%s"}`, r.ScrollID)),
		r.Params.get(),
		r.Header,
	)
}

// ScrollGetResp represents the returned struct of the index create response
type ScrollGetResp struct {
	Took    int            `json:"took"`
	Timeout bool           `json:"timed_out"`
	Shards  ResponseShards `json:"_shards"`
	Hits    struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore *float32    `json:"max_score"`
		Hits     []SearchHit `json:"hits"`
	} `json:"hits"`
	ScrollID        *string  `json:"_scroll_id,omitempty"`
	TerminatedEarly bool     `json:"terminated_early"`
	MaxScore        *float32 `json:"max_score"`
	response        *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ScrollGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
