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

// TemplateCreateReq represents possible options for the index create request
type TemplateCreateReq struct {
	Template string

	Body io.Reader

	Header http.Header
	Params TemplateCreateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r TemplateCreateReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"PUT",
		fmt.Sprintf("/_template/%s", r.Template),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// TemplateCreateResp represents the returned struct of the index create response
type TemplateCreateResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r TemplateCreateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
