// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// ReindexRethrottle executes a / request with the optional ReindexRethrottleReq
func (c Client) ReindexRethrottle(ctx context.Context, req ReindexRethrottleReq) (*ReindexRethrottleResp, error) {
	var (
		data ReindexRethrottleResp
		err  error
	)
	if data.response, err = c.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// ReindexRethrottleReq represents possible options for the / request
type ReindexRethrottleReq struct {
	TaskID string

	Header http.Header
	Params ReindexRethrottleParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ReindexRethrottleReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		fmt.Sprintf("/_reindex/%s/_rethrottle", r.TaskID),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ReindexRethrottleResp represents the returned struct of the / response
type ReindexRethrottleResp struct {
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
				Average struct {
					CPUTimeInNanos int `json:"cpu_time_in_nanos"`
					MemoryInBytes  int `json:"memory_in_bytes"`
				} `json:"average"`
				Total struct {
					CPUTimeInNanos int `json:"cpu_time_in_nanos"`
					MemoryInBytes  int `json:"memory_in_bytes"`
				} `json:"total"`
				Min struct {
					CPUTimeInNanos int `json:"cpu_time_in_nanos"`
					MemoryInBytes  int `json:"memory_in_bytes"`
				} `json:"min"`
				Max struct {
					CPUTimeInNanos int `json:"cpu_time_in_nanos"`
					MemoryInBytes  int `json:"memory_in_bytes"`
				} `json:"max"`
				ThreadInfo struct {
					ThreadExecutions int `json:"thread_executions"`
					ActiveThreads    int `json:"active_threads"`
				} `json:"thread_info"`
			} `json:"resource_stats"`
		} `json:"tasks"`
	} `json:"nodes"`
	NodeFailures []FailuresCause `json:"node_failures"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ReindexRethrottleResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
