// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
)

type tasksClient struct {
	apiClient *Client
}

// Cancel executes a delete tasks request with the required TasksCancelReq
func (c tasksClient) Cancel(ctx context.Context, req TasksCancelReq) (*TasksCancelResp, error) {
	var (
		data TasksCancelResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// List executes a get tasks request with the optional TasksListReq
func (c tasksClient) List(ctx context.Context, req *TasksListReq) (*TasksListResp, error) {
	if req == nil {
		req = &TasksListReq{}
	}

	var (
		data TasksListResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a get tasks request with the optional TasksGetReq
func (c tasksClient) Get(ctx context.Context, req TasksGetReq) (*TasksGetResp, error) {
	var (
		data TasksGetResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}
