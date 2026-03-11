// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import "context"

type snapshotClient struct {
	apiClient  *Client
	Repository repositoryClient
}

// Create executes a creade snapshot request with the required SnapshotCreateReq
func (c snapshotClient) Create(ctx context.Context, req SnapshotCreateReq) (*SnapshotCreateResp, error) {
	var (
		data SnapshotCreateResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Delete executes a delete snapshot request with the required SnapshotDeleteReq
func (c snapshotClient) Delete(ctx context.Context, req SnapshotDeleteReq) (*SnapshotDeleteResp, error) {
	var (
		data SnapshotDeleteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a get snapshot request with the required SnapshotGetReq
func (c snapshotClient) Get(ctx context.Context, req SnapshotGetReq) (*SnapshotGetResp, error) {
	var (
		data SnapshotGetResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Clone executes a snapshot clone request with the required SnapshotCloneReq
func (c snapshotClient) Clone(ctx context.Context, req SnapshotCloneReq) (*SnapshotCloneResp, error) {
	var (
		data SnapshotCloneResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Restore executes a snapshot restore request with the required SnapshotRestoreReq
func (c snapshotClient) Restore(ctx context.Context, req SnapshotRestoreReq) (*SnapshotRestoreResp, error) {
	var (
		data SnapshotRestoreResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Status executes a snapshot status request with the required SnapshotStatusReq
func (c snapshotClient) Status(ctx context.Context, req SnapshotStatusReq) (*SnapshotStatusResp, error) {
	var (
		data SnapshotStatusResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}
