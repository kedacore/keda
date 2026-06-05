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

// TemplateGetReq represents possible options for the index create request
type TemplateGetReq struct {
	Templates []string

	Header http.Header
	Params TemplateGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r TemplateGetReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		fmt.Sprintf("/_template/%s", strings.Join(r.Templates, ",")),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// TemplateGetResp represents the returned struct of the index create response
type TemplateGetResp struct {
	Templates map[string]TemplateGetDetails
	response  *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r TemplateGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// TemplateGetDetails is a sub type of TemplateGetResp containing information about an index template
type TemplateGetDetails struct {
	Order         int64           `json:"order"`
	Version       int64           `json:"version"`
	IndexPatterns []string        `json:"index_patterns"`
	Mappings      json.RawMessage `json:"mappings"`
	Settings      json.RawMessage `json:"settings"`
	Aliases       json.RawMessage `json:"aliases"`
}
