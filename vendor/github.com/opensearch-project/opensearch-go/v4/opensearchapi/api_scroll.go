// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
)

type scrollClient struct {
	apiClient *Client
}

// Delete executes a delete scroll request with the required ScrollDeleteReq
func (c scrollClient) Delete(ctx context.Context, req ScrollDeleteReq) (*ScrollDeleteResp, error) {
	var (
		data ScrollDeleteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a get scroll request with the required ScrollGetReq
func (c scrollClient) Get(ctx context.Context, req ScrollGetReq) (*ScrollGetResp, error) {
	var (
		data ScrollGetResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}
