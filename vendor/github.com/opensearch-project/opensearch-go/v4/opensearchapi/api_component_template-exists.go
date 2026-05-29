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

// ComponentTemplateExistsReq represents possible options for the _component_template exists request
type ComponentTemplateExistsReq struct {
	ComponentTemplate string

	Header http.Header
	Params ComponentTemplateExistsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ComponentTemplateExistsReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"HEAD",
		fmt.Sprintf("/_component_template/%s", r.ComponentTemplate),
		nil,
		r.Params.get(),
		r.Header,
	)
}
