// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
)

type repositoryClient struct {
	apiClient *Client
}

// Create executes a put repository request with the required SnapshotRepositoryCreateReq
func (c repositoryClient) Create(ctx context.Context, req SnapshotRepositoryCreateReq) (*SnapshotRepositoryCreateResp, error) {
	var (
		data SnapshotRepositoryCreateResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Delete executes a delete repository request with the required SnapshotRepositoryDeleteReq
func (c repositoryClient) Delete(ctx context.Context, req SnapshotRepositoryDeleteReq) (*SnapshotRepositoryDeleteResp, error) {
	var (
		data SnapshotRepositoryDeleteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a get repository request with the optional SnapshotRepositoryGetReq
func (c repositoryClient) Get(ctx context.Context, req *SnapshotRepositoryGetReq) (*SnapshotRepositoryGetResp, error) {
	if req == nil {
		req = &SnapshotRepositoryGetReq{}
	}
	var (
		data SnapshotRepositoryGetResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Repos); err != nil {
		return &data, err
	}

	return &data, nil
}

// Cleanup executes a cleanup repository request with the required SnapshotRepositoryCleanupReq
func (c repositoryClient) Cleanup(ctx context.Context, req SnapshotRepositoryCleanupReq) (*SnapshotRepositoryCleanupResp, error) {
	var (
		data SnapshotRepositoryCleanupResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Verify executes a verify repository request with the required SnapshotRepositoryVerifyReq
func (c repositoryClient) Verify(ctx context.Context, req SnapshotRepositoryVerifyReq) (*SnapshotRepositoryVerifyResp, error) {
	var (
		data SnapshotRepositoryVerifyResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}
