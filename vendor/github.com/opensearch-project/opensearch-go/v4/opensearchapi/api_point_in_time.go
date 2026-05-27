// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
)

type pointInTimeClient struct {
	apiClient *Client
}

// Create executes a creade pointInTime request with the required PointInTimeCreateReq
func (c pointInTimeClient) Create(ctx context.Context, req PointInTimeCreateReq) (*PointInTimeCreateResp, error) {
	var (
		data PointInTimeCreateResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Delete executes a delete pointInTime request with the required PointInTimeDeleteReq
func (c pointInTimeClient) Delete(ctx context.Context, req PointInTimeDeleteReq) (*PointInTimeDeleteResp, error) {
	var (
		data PointInTimeDeleteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a get pointInTime request with the optional PointInTimeGetReq
func (c pointInTimeClient) Get(ctx context.Context, req *PointInTimeGetReq) (*PointInTimeGetResp, error) {
	if req == nil {
		req = &PointInTimeGetReq{}
	}

	var (
		data PointInTimeGetResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}
