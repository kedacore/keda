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

// DataStreamDeleteReq represents possible options for the index _data_stream delete request
type DataStreamDeleteReq struct {
	DataStream string

	Header http.Header
	Params DataStreamDeleteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DataStreamDeleteReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"DELETE",
		fmt.Sprintf("/_data_stream/%s", r.DataStream),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// DataStreamDeleteResp represents the returned struct of the _data_stream delete response
type DataStreamDeleteResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DataStreamDeleteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
