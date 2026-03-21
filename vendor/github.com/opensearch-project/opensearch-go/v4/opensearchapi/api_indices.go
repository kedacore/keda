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

type indicesClient struct {
	apiClient *Client
	Alias     aliasClient
	Mapping   mappingClient
	Settings  settingsClient
}

// Delete executes a delete indices request with the required IndicesDeleteReq
func (c indicesClient) Delete(ctx context.Context, req IndicesDeleteReq) (*IndicesDeleteResp, error) {
	var (
		data IndicesDeleteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Create executes a creade indices request with the required IndicesCreateReq
func (c indicesClient) Create(ctx context.Context, req IndicesCreateReq) (*IndicesCreateResp, error) {
	var (
		data IndicesCreateResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Exists executes a exists indices request with the required IndicesExistsReq
func (c indicesClient) Exists(ctx context.Context, req IndicesExistsReq) (*opensearch.Response, error) {
	return c.apiClient.do(ctx, req, nil)
}

// Block executes a /<index>/_block request with the required IndicesBlockReq
func (c indicesClient) Block(ctx context.Context, req IndicesBlockReq) (*IndicesBlockResp, error) {
	var (
		data IndicesBlockResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Analyze executes a /<index>/_analyze request with the required IndicesAnalyzeReq
func (c indicesClient) Analyze(ctx context.Context, req IndicesAnalyzeReq) (*IndicesAnalyzeResp, error) {
	var (
		data IndicesAnalyzeResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// ClearCache executes a /<index>/_cache/clear request with the optional IndicesClearCacheReq
func (c indicesClient) ClearCache(ctx context.Context, req *IndicesClearCacheReq) (*IndicesClearCacheResp, error) {
	if req == nil {
		req = &IndicesClearCacheReq{}
	}

	var (
		data IndicesClearCacheResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Clone executes a /<index>/_clone/<target> request with the required IndicesCloneReq
func (c indicesClient) Clone(ctx context.Context, req IndicesCloneReq) (*IndicesCloneResp, error) {
	var (
		data IndicesCloneResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Close executes a /<index>/_close request with the required IndicesCloseReq
func (c indicesClient) Close(ctx context.Context, req IndicesCloseReq) (*IndicesCloseResp, error) {
	var (
		data IndicesCloseResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Get executes a /<index> request with the required IndicesGetReq
func (c indicesClient) Get(ctx context.Context, req IndicesGetReq) (*IndicesGetResp, error) {
	var (
		data IndicesGetResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Indices); err != nil {
		return &data, err
	}

	return &data, nil
}

// Open executes a /<index>/_open request with the required IndicesOpenReq
func (c indicesClient) Open(ctx context.Context, req IndicesOpenReq) (*IndicesOpenResp, error) {
	var (
		data IndicesOpenResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Shrink executes a /<index>/_shrink/<target> request with the required IndicesShrinkReq
func (c indicesClient) Shrink(ctx context.Context, req IndicesShrinkReq) (*IndicesShrinkResp, error) {
	var (
		data IndicesShrinkResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Split executes a /<index>/_split/<target> request with the required IndicesSplitReq
func (c indicesClient) Split(ctx context.Context, req IndicesSplitReq) (*IndicesSplitResp, error) {
	var (
		data IndicesSplitResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Flush executes a /<index>/_flush request with the optional IndicesFlushReq
func (c indicesClient) Flush(ctx context.Context, req *IndicesFlushReq) (*IndicesFlushResp, error) {
	if req == nil {
		req = &IndicesFlushReq{}
	}

	var (
		data IndicesFlushResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Forcemerge executes a /<index>/_forcemerge request with the optional IndicesForcemergeReq
func (c indicesClient) Forcemerge(ctx context.Context, req *IndicesForcemergeReq) (*IndicesForcemergeResp, error) {
	if req == nil {
		req = &IndicesForcemergeReq{}
	}

	var (
		data IndicesForcemergeResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Recovery executes a /<index>/_recovery request with the optional IndicesRecoveryReq
func (c indicesClient) Recovery(ctx context.Context, req *IndicesRecoveryReq) (*IndicesRecoveryResp, error) {
	if req == nil {
		req = &IndicesRecoveryReq{}
	}

	var (
		data IndicesRecoveryResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Indices); err != nil {
		return &data, err
	}

	return &data, nil
}

// Refresh executes a /<index>/_refresh request with the optional IndicesRefreshReq
func (c indicesClient) Refresh(ctx context.Context, req *IndicesRefreshReq) (*IndicesRefreshResp, error) {
	if req == nil {
		req = &IndicesRefreshReq{}
	}

	var (
		data IndicesRefreshResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Rollover executes a /<index>/_rollover request with the required IndicesRolloverReq
func (c indicesClient) Rollover(ctx context.Context, req IndicesRolloverReq) (*IndicesRolloverResp, error) {
	var (
		data IndicesRolloverResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Segments executes a /<index>/_segments request with the optional IndicesSegmentsReq
func (c indicesClient) Segments(ctx context.Context, req *IndicesSegmentsReq) (*IndicesSegmentsResp, error) {
	if req == nil {
		req = &IndicesSegmentsReq{}
	}

	var (
		data IndicesSegmentsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// ShardStores executes a /<index>/_shard_stores request with the optional IndicesShardStoresReq
func (c indicesClient) ShardStores(ctx context.Context, req *IndicesShardStoresReq) (*IndicesShardStoresResp, error) {
	if req == nil {
		req = &IndicesShardStoresReq{}
	}

	var (
		data IndicesShardStoresResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Stats executes a /<index>/_stats request with the optional IndicesStatsReq
func (c indicesClient) Stats(ctx context.Context, req *IndicesStatsReq) (*IndicesStatsResp, error) {
	if req == nil {
		req = &IndicesStatsReq{}
	}

	var (
		data IndicesStatsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// ValidateQuery executes a /<index>/_validate/query request with the required IndicesValidateQueryReq
func (c indicesClient) ValidateQuery(ctx context.Context, req IndicesValidateQueryReq) (*IndicesValidateQueryResp, error) {
	var (
		data IndicesValidateQueryResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Count executes a /<index>/_count request with the required IndicesCountReq
func (c indicesClient) Count(ctx context.Context, req *IndicesCountReq) (*IndicesCountResp, error) {
	if req == nil {
		req = &IndicesCountReq{}
	}

	var (
		data IndicesCountResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// FieldCaps executes a /<index>/_field_caps request with the required IndicesFieldCapsReq
func (c indicesClient) FieldCaps(ctx context.Context, req IndicesFieldCapsReq) (*IndicesFieldCapsResp, error) {
	var (
		data IndicesFieldCapsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Resolve executes a /_resolve/index/<indices> request with the required IndicesResolveReq
func (c indicesClient) Resolve(ctx context.Context, req IndicesResolveReq) (*IndicesResolveResp, error) {
	var (
		data IndicesResolveResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}
