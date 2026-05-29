// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// IndexTemplateSimulateReq represents possible options for the index create request
type IndexTemplateSimulateReq struct {
	IndexTemplate string

	Body io.Reader

	Header http.Header
	Params IndexTemplateSimulateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndexTemplateSimulateReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		fmt.Sprintf("/_index_template/_simulate/%s", r.IndexTemplate),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// IndexTemplateSimulateResp represents the returned struct of the index create response
type IndexTemplateSimulateResp struct {
	Template struct {
		Mappings json.RawMessage `json:"mappings"`
		Settings json.RawMessage `json:"settings"`
		Aliases  json.RawMessage `json:"aliases"`
	} `json:"template"`
	Overlapping []struct {
		Name          string   `json:"name"`
		IndexPatterns []string `json:"index_patterns"`
	} `json:"overlapping"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndexTemplateSimulateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
