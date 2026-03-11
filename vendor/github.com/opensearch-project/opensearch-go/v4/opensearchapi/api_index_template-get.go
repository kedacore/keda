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

// IndexTemplateGetReq represents possible options for the index create request
type IndexTemplateGetReq struct {
	IndexTemplates []string

	Header http.Header
	Params IndexTemplateGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndexTemplateGetReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		fmt.Sprintf("/_index_template/%s", strings.Join(r.IndexTemplates, ",")),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// IndexTemplateGetResp represents the returned struct of the index create response
type IndexTemplateGetResp struct {
	IndexTemplates []IndexTemplateGetDetails `json:"index_templates"`
	response       *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndexTemplateGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// IndexTemplateGetDetails is a sub type of IndexTemplateGetResp containing information about an index template
type IndexTemplateGetDetails struct {
	Name          string `json:"name"`
	IndexTemplate struct {
		IndexPatterns []string `json:"index_patterns"`
		Template      struct {
			Mappings json.RawMessage `json:"mappings"`
			Settings json.RawMessage `json:"settings"`
			Aliases  json.RawMessage `json:"aliases"`
		} `json:"template"`
		ComposedOf []string        `json:"composed_of"`
		Priority   int             `json:"priority"`
		Version    int             `json:"version"`
		DataStream json.RawMessage `json:"data_stream"`
	} `json:"index_template"`
}
