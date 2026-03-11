// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
)

type dataStreamClient struct {
	apiClient *Client
}

// Create executes a creade dataStream request with the required DataStreamCreateReq
func (c dataStreamClient) Create(ctx context.Context, req DataStreamCreateReq) (*DataStreamCreateResp, error) {
	var (
		data DataStreamCreateResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Delete executes a delete dataStream request with the required DataStreamDeleteReq
func (c dataStreamClient) Delete(ctx context.Context, req DataStreamDeleteReq) (*DataStreamDeleteResp, error) {
	var (
		data DataStreamDeleteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a get dataStream request with the optional DataStreamGetReq
func (c dataStreamClient) Get(ctx context.Context, req *DataStreamGetReq) (*DataStreamGetResp, error) {
	if req == nil {
		req = &DataStreamGetReq{}
	}

	var (
		data DataStreamGetResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Stats executes a stats dataStream request with the optional DataStreamStatsReq
func (c dataStreamClient) Stats(ctx context.Context, req *DataStreamStatsReq) (*DataStreamStatsResp, error) {
	if req == nil {
		req = &DataStreamStatsReq{}
	}

	var (
		data DataStreamStatsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}
