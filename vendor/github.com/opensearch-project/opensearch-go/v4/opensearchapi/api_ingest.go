// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
)

type ingestClient struct {
	apiClient *Client
}

// Create executes a creade ingest request with the required IngestCreateReq
func (c ingestClient) Create(ctx context.Context, req IngestCreateReq) (*IngestCreateResp, error) {
	var (
		data IngestCreateResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Delete executes a delete ingest request with the required IngestDeleteReq
func (c ingestClient) Delete(ctx context.Context, req IngestDeleteReq) (*IngestDeleteResp, error) {
	var (
		data IngestDeleteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a get ingest request with the optional IngestGetReq
func (c ingestClient) Get(ctx context.Context, req *IngestGetReq) (*IngestGetResp, error) {
	if req == nil {
		req = &IngestGetReq{}
	}

	var (
		data IngestGetResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Pipelines); err != nil {
		return &data, err
	}

	return &data, nil
}

// Simulate executes a stats ingest request with the optional IngestSimulateReq
func (c ingestClient) Simulate(ctx context.Context, req IngestSimulateReq) (*IngestSimulateResp, error) {
	var (
		data IngestSimulateResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Grok executes a get ingest request with the optional IngestGrokReq
func (c ingestClient) Grok(ctx context.Context, req *IngestGrokReq) (*IngestGrokResp, error) {
	if req == nil {
		req = &IngestGrokReq{}
	}

	var (
		data IngestGrokResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}
