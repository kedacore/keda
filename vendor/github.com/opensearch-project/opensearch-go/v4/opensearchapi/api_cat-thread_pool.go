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

// CatThreadPoolReq represent possible options for the /_cat/thread_pool request
type CatThreadPoolReq struct {
	Pools  []string
	Header http.Header
	Params CatThreadPoolParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatThreadPoolReq) GetRequest() (*http.Request, error) {
	pools := strings.Join(r.Pools, ",")
	var path strings.Builder
	path.Grow(len("/_cat/thread_pool/") + len(pools))
	path.WriteString("/_cat/thread_pool")
	if len(r.Pools) > 0 {
		path.WriteString("/")
		path.WriteString(pools)
	}
	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatThreadPoolResp represents the returned struct of the /_cat/thread_pool response
type CatThreadPoolResp struct {
	ThreadPool []CatThreadPoolItemResp
	response   *opensearch.Response
}

// CatThreadPoolItemResp represents one index of the CatThreadPoolResp
type CatThreadPoolItemResp struct {
	NodeName        string  `json:"node_name"`
	NodeID          string  `json:"node_id"`
	EphemeralNodeID string  `json:"ephemeral_node_id"`
	PID             int     `json:"pid,string"`
	Host            string  `json:"host"`
	IP              string  `json:"ip"`
	Port            int     `json:"port,string"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Active          int     `json:"active,string"`
	PoolSize        int     `json:"pool_size,string"`
	Queue           int     `json:"queue,string"`
	QueueSize       int     `json:"queue_size,string"`
	Rejected        int     `json:"rejected,string"`
	Largest         int     `json:"largest,string"`
	Completed       int     `json:"completed,string"`
	Core            *int    `json:"core,string"`
	Max             *int    `json:"max,string"`
	Size            *int    `json:"size,string"`
	KeepAlive       *string `json:"keep_alive"`
	TotalWaitTime   string  `json:"total_wait_time"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatThreadPoolResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
