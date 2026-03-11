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
	"io"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// UpdateByQuery executes a /_update_by_query request with the optional UpdateByQueryReq
func (c Client) UpdateByQuery(ctx context.Context, req UpdateByQueryReq) (*UpdateByQueryResp, error) {
	var (
		data UpdateByQueryResp
		err  error
	)
	if data.response, err = c.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// UpdateByQueryReq represents possible options for the /_update_by_query request
type UpdateByQueryReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params UpdateByQueryParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r UpdateByQueryReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"POST",
		fmt.Sprintf("/%s/_update_by_query", strings.Join(r.Indices, ",")),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// UpdateByQueryResp represents the returned struct of the /_update_by_query response
type UpdateByQueryResp struct {
	Took             int  `json:"took"`
	TimedOut         bool `json:"timed_out"`
	Total            int  `json:"total"`
	Updated          int  `json:"updated"`
	Deleted          int  `json:"deleted"`
	Batches          int  `json:"batches"`
	VersionConflicts int  `json:"version_conflicts"`
	Noops            int  `json:"noops"`
	Retries          struct {
		Bulk   int `json:"bulk"`
		Search int `json:"search"`
	} `json:"retries"`
	ThrottledMillis      int               `json:"throttled_millis"`
	RequestsPerSecond    float32           `json:"requests_per_second"`
	ThrottledUntilMillis int               `json:"throttled_until_millis"`
	Failures             []json.RawMessage `json:"failures"`
	Type                 string            `json:"_type"` // Deprecated field
	Task                 string            `json:"task"`
	response             *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r UpdateByQueryResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
