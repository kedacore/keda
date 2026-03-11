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

// IndexTemplateExistsReq represents possible options for the index create request
type IndexTemplateExistsReq struct {
	IndexTemplate string

	Header http.Header
	Params IndexTemplateExistsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndexTemplateExistsReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"HEAD",
		fmt.Sprintf("/_index_template/%s", r.IndexTemplate),
		nil,
		r.Params.get(),
		r.Header,
	)
}
