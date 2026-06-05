// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// MSearchTemplate executes a /_msearch request with the optional MSearchTemplateReq
func (c Client) MSearchTemplate(ctx context.Context, req MSearchTemplateReq) (*MSearchTemplateResp, error) {
	var (
		data MSearchTemplateResp
		err  error
	)
	if data.response, err = c.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// MSearchTemplateReq represents possible options for the /_msearch request
type MSearchTemplateReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params MSearchTemplateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r MSearchTemplateReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")
	var path strings.Builder
	path.Grow(len("//_msearch/template") + len(indices))
	if len(r.Indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	path.WriteString("/_msearch/template")
	return opensearch.BuildRequest(
		"POST",
		path.String(),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// MSearchTemplateResp represents the returned struct of the /_msearch response
type MSearchTemplateResp struct {
	Took      int `json:"took"`
	Responses []struct {
		Took    int            `json:"took"`
		Timeout bool           `json:"timed_out"`
		Shards  ResponseShards `json:"_shards"`
		Hits    struct {
			Total struct {
				Value    int    `json:"value"`
				Relation string `json:"relation"`
			} `json:"total"`
			MaxScore *float32    `json:"max_score"`
			Hits     []SearchHit `json:"hits"`
		} `json:"hits"`
		Status int `json:"status"`
	} `json:"responses"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r MSearchTemplateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
