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

// TasksGetReq represents possible options for the index create request
type TasksGetReq struct {
	TaskID string

	Header http.Header
	Params TasksGetParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r TasksGetReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		fmt.Sprintf("/_tasks/%s", r.TaskID),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// TasksGetResp represents the returned struct of the index create response
type TasksGetResp struct {
	Completed bool `json:"completed"`
	Task      struct {
		Node               string          `json:"node"`
		ID                 int             `json:"id"`
		Type               string          `json:"type"`
		Action             string          `json:"action"`
		Description        string          `json:"description"`
		StartTimeInMillis  int64           `json:"start_time_in_millis"`
		RunningTimeInNanos int64           `json:"running_time_in_nanos"`
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
	} `json:"task"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r TasksGetResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// TasksGetDetails is a sub type of TasksGetResp containing information about an index template
type TasksGetDetails struct {
	Order         int64           `json:"order"`
	Version       int64           `json:"version"`
	IndexPatterns []string        `json:"index_patterns"`
	Mappings      json.RawMessage `json:"mappings"`
	Settings      json.RawMessage `json:"settings"`
	Aliases       json.RawMessage `json:"aliases"`
}
