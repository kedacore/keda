// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// PointInTimeGetReq represents possible options for the index create request
type PointInTimeGetReq struct {
	Header http.Header
	Params PointInTimeGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r PointInTimeGetReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_search/point_in_time/_all",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// PointInTimeGetResp represents the returned struct of the index create response
type PointInTimeGetResp struct {
	Pits []struct {
		PitID        string `json:"pit_id"`
		CreationTime int    `json:"creation_time"`
		KeepAlive    int64  `json:"keep_alive"`
	} `json:"pits"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r PointInTimeGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
