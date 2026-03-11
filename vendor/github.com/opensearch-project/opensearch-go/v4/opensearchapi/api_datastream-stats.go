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

// DataStreamStatsReq represents possible options for the _data_stream stats request
type DataStreamStatsReq struct {
	DataStreams []string

	Header http.Header
	Params DataStreamStatsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DataStreamStatsReq) GetRequest() (*http.Request, error) {
	dataStreams := strings.Join(r.DataStreams, ",")

	var path strings.Builder
	path.Grow(len("/_data_stream//_stats") + len(dataStreams))
	path.WriteString("/_data_stream/")
	if len(r.DataStreams) > 0 {
		path.WriteString(dataStreams)
		path.WriteString("/")
	}
	path.WriteString("_stats")

	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// DataStreamStatsResp represents the returned struct of the _data_stream stats response
type DataStreamStatsResp struct {
	Shards struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
		Failures   []struct {
			Shard  int           `json:"shard"`
			Index  string        `json:"index"`
			Status string        `json:"status"`
			Reason FailuresCause `json:"reason"`
		} `json:"failures"`
	} `json:"_shards"`
	DataStreamCount     int                      `json:"data_stream_count"`
	BackingIndices      int                      `json:"backing_indices"`
	TotalStoreSizeBytes int64                    `json:"total_store_size_bytes"`
	DataStreams         []DataStreamStatsDetails `json:"data_streams"`
	response            *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DataStreamStatsResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// DataStreamStatsDetails is a sub type of DataStreamStatsResp containing information about a data stream
type DataStreamStatsDetails struct {
	DataStream       string `json:"data_stream"`
	BackingIndices   int    `json:"backing_indices"`
	StoreSizeBytes   int64  `json:"store_size_bytes"`
	MaximumTimestamp int64  `json:"maximum_timestamp"`
}
