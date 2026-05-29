// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// DataStreamGetReq represents possible options for the _data_stream get request
type DataStreamGetReq struct {
	DataStreams []string

	Header http.Header
	Params DataStreamGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DataStreamGetReq) GetRequest() (*http.Request, error) {
	dataStreams := strings.Join(r.DataStreams, ",")

	var path strings.Builder
	path.Grow(len("/_data_stream/") + len(dataStreams))
	path.WriteString("/_data_stream")
	if len(r.DataStreams) > 0 {
		path.WriteString("/")
		path.WriteString(dataStreams)
	}

	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// DataStreamGetResp represents the returned struct of the _data_stream get response
type DataStreamGetResp struct {
	DataStreams []DataStreamGetDetails `json:"data_streams"`
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DataStreamGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// DataStreamGetDetails is a sub type if DataStreamGetResp containing information about a data stream
type DataStreamGetDetails struct {
	Name           string `json:"name"`
	TimestampField struct {
		Name string `json:"name"`
	} `json:"timestamp_field"`
	Indices    []DataStreamIndices `json:"indices"`
	Generation int                 `json:"generation"`
	Status     string              `json:"status"`
	Template   string              `json:"template"`
}

// DataStreamIndices is a sub type of DataStreamGetDetails containing information about an index
type DataStreamIndices struct {
	Name string `json:"index_name"`
	UUID string `json:"index_uuid"`
}
