// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"fmt"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// IndexTemplateCreateReq represents possible options for the index create request
type IndexTemplateCreateReq struct {
	IndexTemplate string

	Body io.Reader

	Header http.Header
	Params IndexTemplateCreateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndexTemplateCreateReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"PUT",
		fmt.Sprintf("/_index_template/%s", r.IndexTemplate),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// IndexTemplateCreateResp represents the returned struct of the index create response
type IndexTemplateCreateResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndexTemplateCreateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
