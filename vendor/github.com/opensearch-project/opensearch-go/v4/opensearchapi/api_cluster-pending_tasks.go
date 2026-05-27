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

// ClusterPendingTasksReq represents possible options for the /_cluster/pending_tasks request
type ClusterPendingTasksReq struct {
	Header http.Header
	Params ClusterPendingTasksParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterPendingTasksReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_cluster/pending_tasks",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ClusterPendingTasksResp represents the returned struct of the  ClusterPendingTasksReq response
type ClusterPendingTasksResp struct {
	Tasks    []ClusterPendingTasksItem `json:"tasks"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterPendingTasksResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// ClusterPendingTasksItem is a sub type if ClusterPendingTasksResp containing information about a task
type ClusterPendingTasksItem struct {
	InsertOrder       int    `json:"insert_order"`
	Priority          string `json:"priority"`
	Source            string `json:"source"`
	TimeInQueueMillis int    `json:"time_in_queue_millis"`
	TimeInQueue       string `json:"time_in_queue"`
	Executing         bool   `json:"executing"`
}
