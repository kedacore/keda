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

// ClusterPutDecommissionReq represents possible options for the /_cluster/decommission/awareness request
type ClusterPutDecommissionReq struct {
	AwarenessAttrName  string
	AwarenessAttrValue string

	Header http.Header
	Params ClusterPutDecommissionParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterPutDecommissionReq) GetRequest() (*http.Request, error) {
	var path strings.Builder
	path.Grow(34 + len(r.AwarenessAttrName) + len(r.AwarenessAttrValue))
	path.WriteString("/_cluster/decommission/awareness/")
	path.WriteString(r.AwarenessAttrName)
	path.WriteString("/")
	path.WriteString(r.AwarenessAttrValue)

	return opensearch.BuildRequest(
		"PUT",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ClusterPutDecommissionResp represents the returned struct of the /_cluster/decommission/awareness response
type ClusterPutDecommissionResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterPutDecommissionResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// ClusterDeleteDecommissionReq represents possible options for the /_cluster/decommission/awareness request
type ClusterDeleteDecommissionReq struct {
	Header http.Header
	Params ClusterDeleteDecommissionParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterDeleteDecommissionReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"DELETE",
		"/_cluster/decommission/awareness",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ClusterDeleteDecommissionResp represents the returned struct of the /_cluster/decommission/awareness response
type ClusterDeleteDecommissionResp struct {
	Acknowledged bool `json:"acknowledged"`
	response     *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterDeleteDecommissionResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// ClusterGetDecommissionReq represents possible options for the /_cluster/decommission/awareness request
type ClusterGetDecommissionReq struct {
	AwarenessAttrName string

	Header http.Header
	Params ClusterGetDecommissionParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterGetDecommissionReq) GetRequest() (*http.Request, error) {
	var path strings.Builder
	path.Grow(41 + len(r.AwarenessAttrName))
	path.WriteString("/_cluster/decommission/awareness/")
	path.WriteString(r.AwarenessAttrName)
	path.WriteString("/_status")

	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ClusterGetDecommissionResp represents the returned struct of the /_cluster/decommission/awareness response
type ClusterGetDecommissionResp struct {
	Values   map[string]string
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterGetDecommissionResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
