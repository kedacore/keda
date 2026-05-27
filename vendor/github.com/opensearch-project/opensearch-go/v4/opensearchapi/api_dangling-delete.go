// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"fmt"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// DanglingDeleteReq represents possible options for the delete dangling request
type DanglingDeleteReq struct {
	IndexUUID string

	Header http.Header
	Params DanglingDeleteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DanglingDeleteReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"DELETE",
		fmt.Sprintf("/_dangling/%s", r.IndexUUID),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// DanglingDeleteResp represents the returned struct of the delete dangling response
type DanglingDeleteResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DanglingDeleteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
