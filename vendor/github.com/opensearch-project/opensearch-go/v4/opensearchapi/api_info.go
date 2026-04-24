// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// Info executes a / request with the optional InfoReq
func (c Client) Info(ctx context.Context, req *InfoReq) (*InfoResp, error) {
	if req == nil {
		req = &InfoReq{}
	}

	var (
		data InfoResp
		err  error
	)
	if data.response, err = c.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// InfoReq represents possible options for the / request
type InfoReq struct {
	Header http.Header
	Params InfoParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r InfoReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// InfoResp represents the returned struct of the / response
type InfoResp struct {
	Name        string `json:"name"`
	ClusterName string `json:"cluster_name"`
	ClusterUUID string `json:"cluster_uuid"`
	Version     struct {
		Distribution                     string `json:"distribution"`
		Number                           string `json:"number"`
		BuildType                        string `json:"build_type"`
		BuildHash                        string `json:"build_hash"`
		BuildDate                        string `json:"build_date"`
		BuildSnapshot                    bool   `json:"build_snapshot"`
		LuceneVersion                    string `json:"lucene_version"`
		MinimumWireCompatibilityVersion  string `json:"minimum_wire_compatibility_version"`
		MinimumIndexCompatibilityVersion string `json:"minimum_index_compatibility_version"`
	} `json:"version"`
	Tagline  string `json:"tagline"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r InfoResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
