// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// IndicesGetReq represents possible options for the get indices request
type IndicesGetReq struct {
	Indices []string

	Header http.Header
	Params IndicesGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesGetReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		fmt.Sprintf("/%s", strings.Join(r.Indices, ",")),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// IndicesGetResp represents the returned struct of the get indices response
type IndicesGetResp struct {
	Indices map[string]struct {
		DataStream *string             `json:"data_stream,omitempty"`
		Aliases    map[string]struct{} `json:"aliases"`
		Mappings   json.RawMessage     `json:"mappings"`
		Settings   json.RawMessage     `json:"settings"`
	}
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
