// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"

	"github.com/opensearch-project/opensearch-go/v4"
)

type indexTemplateClient struct {
	apiClient *Client
}

// Create executes a creade indexTemplate request with the required IndexTemplateCreateReq
func (c indexTemplateClient) Create(ctx context.Context, req IndexTemplateCreateReq) (*IndexTemplateCreateResp, error) {
	var (
		data IndexTemplateCreateResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Delete executes a delete indexTemplate request with the required IndexTemplateDeleteReq
func (c indexTemplateClient) Delete(ctx context.Context, req IndexTemplateDeleteReq) (*IndexTemplateDeleteResp, error) {
	var (
		data IndexTemplateDeleteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a get indexTemplate request with the optional IndexTemplateGetReq
func (c indexTemplateClient) Get(ctx context.Context, req *IndexTemplateGetReq) (*IndexTemplateGetResp, error) {
	if req == nil {
		req = &IndexTemplateGetReq{}
	}

	var (
		data IndexTemplateGetResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Exists executes a exists indexTemplate request with the required IndexTemplatExistsReq
func (c indexTemplateClient) Exists(ctx context.Context, req IndexTemplateExistsReq) (*opensearch.Response, error) {
	return c.apiClient.do(ctx, req, nil)
}

// Simulate executes a _simulate indexTemplate request with the required IndexTemplateSimulateReq
func (c indexTemplateClient) Simulate(ctx context.Context, req IndexTemplateSimulateReq) (*IndexTemplateSimulateResp, error) {
	var (
		data IndexTemplateSimulateResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// SimulateIndex executes a _simulate_index indexTemplate request with the required IndexTemplateSimulateIndexReq
func (c indexTemplateClient) SimulateIndex(
	ctx context.Context,
	req IndexTemplateSimulateIndexReq,
) (*IndexTemplateSimulateIndexResp, error) {
	var (
		data IndexTemplateSimulateIndexResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}
