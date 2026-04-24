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

// IndicesRecoveryReq represents possible options for the index shrink request
type IndicesRecoveryReq struct {
	Indices []string

	Header http.Header
	Params IndicesRecoveryParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesRecoveryReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")

	var path strings.Builder
	path.Grow(11 + len(indices))
	if len(indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	path.WriteString("/_recovery")
	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// IndicesRecoveryResp represents the returned struct of the index shrink response
type IndicesRecoveryResp struct {
	Indices map[string]struct {
		Shards []struct {
			ID                int                     `json:"id"`
			Type              string                  `json:"type"`
			Stage             string                  `json:"stage"`
			Primary           bool                    `json:"primary"`
			StartTimeInMillis int64                   `json:"start_time_in_millis"`
			StopTimeInMillis  int64                   `json:"stop_time_in_millis"`
			TotalTimeInMillis int                     `json:"total_time_in_millis"`
			Source            IndicesRecoveryNodeInfo `json:"source"`
			Target            IndicesRecoveryNodeInfo `json:"target"`
			Index             struct {
				Size struct {
					TotalInBytes     int    `json:"total_in_bytes"`
					ReusedInBytes    int    `json:"reused_in_bytes"`
					RecoveredInBytes int    `json:"recovered_in_bytes"`
					Percent          string `json:"percent"`
				} `json:"size"`
				Files struct {
					Total     int    `json:"total"`
					Reused    int    `json:"reused"`
					Recovered int    `json:"recovered"`
					Percent   string `json:"percent"`
				} `json:"files"`
				TotalTimeInMillis          int `json:"total_time_in_millis"`
				SourceThrottleTimeInMillis int `json:"source_throttle_time_in_millis"`
				TargetThrottleTimeInMillis int `json:"target_throttle_time_in_millis"`
			} `json:"index"`
			Translog struct {
				Recovered         int    `json:"recovered"`
				Total             int    `json:"total"`
				Percent           string `json:"percent"`
				TotalOnStart      int    `json:"total_on_start"`
				TotalTimeInMillis int    `json:"total_time_in_millis"`
			} `json:"translog"`
			VerifyIndex struct {
				CheckIndexTimeInMillis int `json:"check_index_time_in_millis"`
				TotalTimeInMillis      int `json:"total_time_in_millis"`
			} `json:"verify_index"`
		} `json:"shards"`
	}
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesRecoveryResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// IndicesRecoveryNodeInfo is a sub type of IndicesRecoveryResp represeing Node information
type IndicesRecoveryNodeInfo struct {
	ID               string `json:"id"`
	Host             string `json:"host"`
	TransportAddress string `json:"transport_address"`
	IP               string `json:"ip"`
	Name             string `json:"name"`
}
