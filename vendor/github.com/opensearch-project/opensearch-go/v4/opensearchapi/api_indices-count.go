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

// IndicesCountReq represents possible options for the index shrink request
type IndicesCountReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params IndicesCountParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesCountReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(8 + len(indices))
	if len(indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	path.WriteString("/_count")
	return opensearch.BuildRequest(
		"POST",
		path.String(),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// IndicesCountResp represents the returned struct of the index shrink response
type IndicesCountResp struct {
	Shards struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Count    int `json:"count"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesCountResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
