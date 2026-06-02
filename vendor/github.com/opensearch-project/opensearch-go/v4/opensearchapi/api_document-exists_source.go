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

// DocumentExistsSourceReq represents possible options for the _source exists request
type DocumentExistsSourceReq struct {
	Index      string
	DocumentID string

	Header http.Header
	Params DocumentExistsSourceParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DocumentExistsSourceReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"HEAD",
		fmt.Sprintf("/%s/_source/%s", r.Index, r.DocumentID),
		nil,
		r.Params.get(),
		r.Header,
	)
}
