// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// RankEval executes a /_rank_eval request with the required RankEvalReq
func (c Client) RankEval(ctx context.Context, req RankEvalReq) (*RankEvalResp, error) {
	var (
		data RankEvalResp
		err  error
	)
	if data.response, err = c.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// RankEvalReq represents possible options for the /_rank_eval request
type RankEvalReq struct {
	Indices []string

	Body io.Reader

	Header http.Header
	Params RankEvalParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r RankEvalReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")
	var path strings.Builder
	path.Grow(len("//_rank_eval") + len(indices))
	if len(r.Indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	path.WriteString("/_rank_eval")
	return opensearch.BuildRequest(
		"GET",
		path.String(),
		r.Body,
		r.Params.get(),
		r.Header,
	)
}

// RankEvalResp represents the returned struct of the /_rank_eval response
type RankEvalResp struct {
	MetricScore float64         `json:"metric_score"`
	Details     json.RawMessage `json:"details"`
	Failures    json.RawMessage `json:"failures"`
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r RankEvalResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}
