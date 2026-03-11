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

type nodesClient struct {
	apiClient *Client
}

// Stats executes a /_nodes/_stats request with the optional NodesStatsReq
func (c nodesClient) Stats(ctx context.Context, req *NodesStatsReq) (*NodesStatsResp, error) {
	if req == nil {
		req = &NodesStatsReq{}
	}

	var (
		data NodesStatsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Info executes a /_nodes request with the optional NodesInfoReq
func (c nodesClient) Info(ctx context.Context, req *NodesInfoReq) (*NodesInfoResp, error) {
	if req == nil {
		req = &NodesInfoReq{}
	}

	var (
		data NodesInfoResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// HotThreads executes a /_nodes/hot_threads request with the optional NodesHotThreadsReq
func (c nodesClient) HotThreads(ctx context.Context, req *NodesHotThreadsReq) (*opensearch.Response, error) {
	if req == nil {
		req = &NodesHotThreadsReq{}
	}
	return c.apiClient.do(ctx, req, nil)
}

// ReloadSecurity executes a /_nodes/reload_secure_settings request with the optional NodesReloadSecurityReq
func (c nodesClient) ReloadSecurity(ctx context.Context, req *NodesReloadSecurityReq) (*NodesReloadSecurityResp, error) {
	if req == nil {
		req = &NodesReloadSecurityReq{}
	}

	var (
		data NodesReloadSecurityResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Usage executes a /_nodes/usage request with the optional NodesUsageReq
func (c nodesClient) Usage(ctx context.Context, req *NodesUsageReq) (*NodesUsageResp, error) {
	if req == nil {
		req = &NodesUsageReq{}
	}

	var (
		data NodesUsageResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}
