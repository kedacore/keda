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

// DataStreamCreateReq represents possible options for the _data_stream create request
type DataStreamCreateReq struct {
	DataStream string

	Header http.Header
	Params DataStreamCreateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DataStreamCreateReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"PUT",
		fmt.Sprintf("/_data_stream/%s", r.DataStream),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// DataStreamCreateResp represents the returned struct of the _data_stream create response
type DataStreamCreateResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DataStreamCreateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
