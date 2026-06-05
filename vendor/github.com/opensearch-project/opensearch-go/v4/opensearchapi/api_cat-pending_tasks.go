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

// CatPendingTasksReq represent possible options for the /_cat/pending_tasks request
type CatPendingTasksReq struct {
	Header http.Header
	Params CatPendingTasksParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatPendingTasksReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_cat/pending_tasks",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatPendingTasksResp represents the returned struct of the /_cat/pending_tasks response
type CatPendingTasksResp struct {
	PendingTasks []CatPendingTaskResp
	response     *opensearch.Response
}

// CatPendingTaskResp represents one index of the CatPendingTasksResp
type CatPendingTaskResp struct {
	InsertOrder string `json:"insertOrder"`
	TimeInQueue string `json:"timeInQueue"`
	Priority    string `json:"priority"`
	Source      string `json:"source"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatPendingTasksResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
