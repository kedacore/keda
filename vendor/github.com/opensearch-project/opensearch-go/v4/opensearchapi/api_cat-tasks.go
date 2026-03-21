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

// CatTasksReq represent possible options for the /_cat/tasks request
type CatTasksReq struct {
	Header http.Header
	Params CatTasksParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatTasksReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_cat/tasks",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatTasksResp represents the returned struct of the /_cat/tasks response
type CatTasksResp struct {
	Tasks    []CatTaskResp
	response *opensearch.Response
}

// CatTaskResp represents one index of the CatTasksResp
type CatTaskResp struct {
	ID            string `json:"id"`
	Action        string `json:"action"`
	TaskID        string `json:"task_id"`
	ParentTaskID  string `json:"parent_task_id"`
	Type          string `json:"type"`
	StartTime     int    `json:"start_time,string"`
	Timestamp     string `json:"timestamp"`
	RunningTimeNs int    `json:"running_time_ns,string"`
	RunningTime   string `json:"running_time"`
	NodeID        string `json:"node_id"`
	IP            string `json:"ip"`
	Port          int    `json:"port,string"`
	Node          string `json:"node"`
	Version       string `json:"version"`
	XOpaqueID     string `json:"x_opaque_id"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatTasksResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
