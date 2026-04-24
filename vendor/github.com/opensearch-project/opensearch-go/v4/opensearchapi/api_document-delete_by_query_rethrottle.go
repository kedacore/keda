// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// DocumentDeleteByQueryRethrottleReq represents possible options for the /_delete_by_query/<index>/_rethrottle request
type DocumentDeleteByQueryRethrottleReq struct {
	TaskID string

	Header http.Header
	Params DocumentDeleteByQueryRethrottleParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r DocumentDeleteByQueryRethrottleReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		fmt.Sprintf("/_delete_by_query/%s/_rethrottle", r.TaskID),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// DocumentDeleteByQueryRethrottleResp represents the returned struct of the /_delete_by_query/<index>/_rethrottle response
type DocumentDeleteByQueryRethrottleResp struct {
	Nodes map[string]struct {
		Name             string            `json:"name"`
		TransportAddress string            `json:"transport_address"`
		Host             string            `json:"host"`
		IP               string            `json:"ip"`
		Roles            []string          `json:"roles"`
		Attributes       map[string]string `json:"attributes"`
		Tasks            map[string]struct {
			Node   string `json:"node"`
			ID     int    `json:"id"`
			Type   string `json:"type"`
			Action string `json:"action"`
			Status struct {
				Total            int `json:"total"`
				Updated          int `json:"updated"`
				Created          int `json:"created"`
				Deleted          int `json:"deleted"`
				Batches          int `json:"batches"`
				VersionConflicts int `json:"version_conflicts"`
				Noops            int `json:"noops"`
				Retries          struct {
					Bulk   int `json:"bulk"`
					Search int `json:"search"`
				} `json:"retries"`
				ThrottledMillis      int     `json:"throttled_millis"`
				RequestsPerSecond    float64 `json:"requests_per_second"`
				ThrottledUntilMillis int     `json:"throttled_until_millis"`
			} `json:"status"`
			Description        string          `json:"description"`
			StartTimeInMillis  int64           `json:"start_time_in_millis"`
			RunningTimeInNanos int             `json:"running_time_in_nanos"`
			Cancellable        bool            `json:"cancellable"`
			Cancelled          bool            `json:"cancelled"`
			Headers            json.RawMessage `json:"headers"`
			ResourceStats      struct {
				Average    DocumentDeleteByQueryRethrottleResourceInfo `json:"average"`
				Max        DocumentDeleteByQueryRethrottleResourceInfo `json:"max"`
				Min        DocumentDeleteByQueryRethrottleResourceInfo `json:"min"`
				Total      DocumentDeleteByQueryRethrottleResourceInfo `json:"total"`
				ThreadInfo struct {
					ActiveThreads    int `json:"active_threads"`
					ThreadExecutions int `json:"thread_executions"`
				} `json:"thread_info"`
			} `json:"resource_stats"`
		} `json:"tasks"`
	} `json:"nodes"`
	NodeFailures []FailuresCause `json:"node_failures"`
	response     *opensearch.Response
}

// DocumentDeleteByQueryRethrottleResourceInfo is a sub type of DocumentDeleteByQueryRethrottleResp containing resource stats
type DocumentDeleteByQueryRethrottleResourceInfo struct {
	CPUTimeInNanos int `json:"cpu_time_in_nanos"`
	MemoryInBytes  int `json:"memory_in_bytes"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r DocumentDeleteByQueryRethrottleResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
