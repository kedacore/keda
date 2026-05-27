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

// SearchTemplate executes a /_search request with the optional SearchTemplateReq
func (c Client) SearchTemplate(ctx context.Context, req SearchTemplateReq) (*SearchTemplateResp, error) {
	var (
		data SearchTemplateResp
		err  error
	)
	if data.response, err = c.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// SearchTemplateReq represents possible options for the /_search request
type SearchTemplateReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params SearchTemplateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r SearchTemplateReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")
	var path strings.Builder
	path.Grow(len("//_search/template") + len(indices))
	if len(r.Indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	path.WriteString("/_search/template")
	return opensearch.BuildRequest(
		"POST",
		path.String(),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// SearchTemplateResp represents the returned struct of the /_search response
type SearchTemplateResp struct {
	Took    int            `json:"took"`
	Timeout bool           `json:"timed_out"`
	Shards  ResponseShards `json:"_shards"`
	Status  int            `json:"status"`
	Hits    struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore *float32    `json:"max_score"`
		Hits     []SearchHit `json:"hits"`
	} `json:"hits"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r SearchTemplateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
