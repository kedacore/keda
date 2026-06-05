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

// PointInTimeCreateReq represents possible options for the index create request
type PointInTimeCreateReq struct {
	Indices []string

	Header http.Header
	Params PointInTimeCreateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r PointInTimeCreateReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(len("//_search/point_in_time") + len(indices))
	if len(r.Indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	path.WriteString("/_search/point_in_time")

	return opensearch.BuildRequest(
		"POST",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// PointInTimeCreateResp represents the returned struct of the index create response
type PointInTimeCreateResp struct {
	PitID  string `json:"pit_id"`
	Shards struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	CreationTime int64 `json:"creation_time"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r PointInTimeCreateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
