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

// DanglingImportReq represents possible options for the dangling import request
type DanglingImportReq struct {
	IndexUUID string

	Header http.Header
	Params DanglingImportParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DanglingImportReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		fmt.Sprintf("/_dangling/%s", r.IndexUUID),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// DanglingImportResp represents the returned struct of thedangling import response
type DanglingImportResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DanglingImportResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
