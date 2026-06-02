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

// CatRecoveryReq represent possible options for the /_cat/recovery request
type CatRecoveryReq struct {
	Indices []string
	Header  http.Header
	Params  CatRecoveryParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatRecoveryReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")
	var path strings.Builder
	path.Grow(len("/_cat/recovery/") + len(indices))
	path.WriteString("/_cat/recovery")
	if len(r.Indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatRecoveryResp represents the returned struct of the /_cat/recovery response
type CatRecoveryResp struct {
	Recovery []CatRecoveryItemResp
	response *opensearch.Response
}

// CatRecoveryItemResp represents one index of the CatRecoveryResp
type CatRecoveryItemResp struct {
	Index                string `json:"index"`
	Shard                int    `json:"shard,string"`
	StartTime            string `json:"start_time"`
	StartTimeMillis      int    `json:"start_time_millis,string"`
	StopTime             string `json:"stop_time"`
	StopTimeMillis       int    `json:"stop_time_millis,string"`
	Time                 string `json:"time"`
	Type                 string `json:"type"`
	Stage                string `json:"stage"`
	SourceHost           string `json:"source_host"`
	SourceNode           string `json:"source_node"`
	TargetHost           string `json:"target_host"`
	TargetNode           string `json:"target_node"`
	Repository           string `json:"repository"`
	Snapshot             string `json:"snapshot"`
	Files                int    `json:"files,string"`
	FilesRecovered       int    `json:"files_recovered,string"`
	FilesPercent         string `json:"files_percent"`
	FilesTotal           int    `json:"files_total,string"`
	Bytes                string `json:"bytes"`
	BytesRecovered       string `json:"bytes_recovered"`
	BytesPercent         string `json:"bytes_percent"`
	BytesTotal           string `json:"bytes_total"`
	TranslogOps          int    `json:"translog_ops,string"`
	TranslogOpsRecovered int    `json:"translog_ops_recovered,string"`
	TranslogOpsPercent   string `json:"translog_ops_percent"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatRecoveryResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
