// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// PointInTimeDeleteReq represents possible options for the index create request
type PointInTimeDeleteReq struct {
	PitID []string

	Header http.Header
	Params PointInTimeDeleteParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r PointInTimeDeleteReq) GetRequest() (*http.Request, error) {
	var body io.Reader
	if len(r.PitID) > 0 {
		bodyStruct := PointInTimeDeleteRequestBody{PitID: r.PitID}
		bodyJSON, err := json.Marshal(bodyStruct)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(bodyJSON)
	}

	return opensearch.BuildRequest(
		"DELETE",
		"/_search/point_in_time",
		body,
		r.Params.get(),
		r.Header,
	)
}

// PointInTimeDeleteRequestBody is used to from the delete request body
type PointInTimeDeleteRequestBody struct {
	PitID []string `json:"pit_id"`
}

// PointInTimeDeleteResp represents the returned struct of the index create response
type PointInTimeDeleteResp struct {
	Pits []struct {
		PitID      string `json:"pit_id"`
		Successful bool   `json:"successful"`
	} `json:"pits"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r PointInTimeDeleteResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
