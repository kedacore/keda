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

// Ping executes a / request with the optional PingReq
func (c Client) Ping(ctx context.Context, req *PingReq) (*opensearch.Response, error) {
	if req == nil {
		req = &PingReq{}
	}

	return c.do(ctx, req, nil)
}

// PingReq represents possible options for the / request
type PingReq struct {
	Header http.Header
	Params PingParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r PingReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"HEAD",
		"/",
		nil,
		r.Params.get(),
		r.Header,
	)
}
