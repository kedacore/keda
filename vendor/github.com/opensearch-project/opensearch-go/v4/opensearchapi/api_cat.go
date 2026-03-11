// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
)

type catClient struct {
	apiClient *Client
}

// Aliases executes a /_cat/aliases request with the optional CatAliasesReq
func (c catClient) Aliases(ctx context.Context, req *CatAliasesReq) (*CatAliasesResp, error) {
	if req == nil {
		req = &CatAliasesReq{}
	}

	var (
		data CatAliasesResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Aliases); err != nil {
		return &data, err
	}

	return &data, nil
}

// Allocation executes a /_cat/allocation request with the optional CatAllocationReq
func (c catClient) Allocation(ctx context.Context, req *CatAllocationReq) (*CatAllocationsResp, error) {
	if req == nil {
		req = &CatAllocationReq{}
	}

	var (
		data CatAllocationsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Allocations); err != nil {
		return &data, err
	}

	return &data, nil
}

// ClusterManager executes a /_cat/cluster_manager request with the optional CatClusterManagerReq
func (c catClient) ClusterManager(ctx context.Context, req *CatClusterManagerReq) (*CatClusterManagersResp, error) {
	if req == nil {
		req = &CatClusterManagerReq{}
	}

	var (
		data CatClusterManagersResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.ClusterManagers); err != nil {
		return &data, err
	}

	return &data, nil
}

// Count executes a /_cat/count request with the optional CatCountReq
func (c catClient) Count(ctx context.Context, req *CatCountReq) (*CatCountsResp, error) {
	if req == nil {
		req = &CatCountReq{}
	}

	var (
		data CatCountsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Counts); err != nil {
		return &data, err
	}

	return &data, nil
}

// FieldData executes a /_cat/fielddata request with the optional CatFieldDataReq
func (c catClient) FieldData(ctx context.Context, req *CatFieldDataReq) (*CatFieldDataResp, error) {
	if req == nil {
		req = &CatFieldDataReq{}
	}

	var (
		data CatFieldDataResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.FieldData); err != nil {
		return &data, err
	}

	return &data, nil
}

// Health executes a /_cat/health request with the optional CatHealthReq
func (c catClient) Health(ctx context.Context, req *CatHealthReq) (*CatHealthResp, error) {
	if req == nil {
		req = &CatHealthReq{}
	}

	var (
		data CatHealthResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Health); err != nil {
		return &data, err
	}

	return &data, nil
}

// Indices executes a /_cat/indices request with the optional CatIndicesReq
func (c catClient) Indices(ctx context.Context, req *CatIndicesReq) (*CatIndicesResp, error) {
	if req == nil {
		req = &CatIndicesReq{}
	}

	var (
		data CatIndicesResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Indices); err != nil {
		return &data, err
	}

	return &data, nil
}

// Master executes a /_cat/master request with the optional CatMasterReq
func (c catClient) Master(ctx context.Context, req *CatMasterReq) (*CatMasterResp, error) {
	if req == nil {
		req = &CatMasterReq{}
	}

	var (
		data CatMasterResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Master); err != nil {
		return &data, err
	}

	return &data, nil
}

// NodeAttrs executes a /_cat/nodeattrs request with the optional CatNodeAttrsReq
func (c catClient) NodeAttrs(ctx context.Context, req *CatNodeAttrsReq) (*CatNodeAttrsResp, error) {
	if req == nil {
		req = &CatNodeAttrsReq{}
	}

	var (
		data CatNodeAttrsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.NodeAttrs); err != nil {
		return &data, err
	}

	return &data, nil
}

// Nodes executes a /_cat/nodes request with the optional CatNodesReq
func (c catClient) Nodes(ctx context.Context, req *CatNodesReq) (*CatNodesResp, error) {
	if req == nil {
		req = &CatNodesReq{}
	}

	var (
		data CatNodesResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Nodes); err != nil {
		return &data, err
	}

	return &data, nil
}

// PendingTasks executes a /_cat/pending_tasks request with the optional CatPendingTasksReq
func (c catClient) PendingTasks(ctx context.Context, req *CatPendingTasksReq) (*CatPendingTasksResp, error) {
	if req == nil {
		req = &CatPendingTasksReq{}
	}

	var (
		data CatPendingTasksResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.PendingTasks); err != nil {
		return &data, err
	}

	return &data, nil
}

// Plugins executes a /_cat/plugins request with the optional CatPluginsReq
func (c catClient) Plugins(ctx context.Context, req *CatPluginsReq) (*CatPluginsResp, error) {
	if req == nil {
		req = &CatPluginsReq{}
	}

	var (
		data CatPluginsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Plugins); err != nil {
		return &data, err
	}

	return &data, nil
}

// Recovery executes a /_cat/recovery request with the optional CatRecoveryReq
func (c catClient) Recovery(ctx context.Context, req *CatRecoveryReq) (*CatRecoveryResp, error) {
	if req == nil {
		req = &CatRecoveryReq{}
	}

	var (
		data CatRecoveryResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Recovery); err != nil {
		return &data, err
	}

	return &data, nil
}

// Repositories executes a /_cat/repositories request with the optional CatRepositoriesReq
func (c catClient) Repositories(ctx context.Context, req *CatRepositoriesReq) (*CatRepositoriesResp, error) {
	if req == nil {
		req = &CatRepositoriesReq{}
	}

	var (
		data CatRepositoriesResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Repositories); err != nil {
		return &data, err
	}

	return &data, nil
}

// Segments executes a /_cat/segments request with the optional CatSegmentsReq
func (c catClient) Segments(ctx context.Context, req *CatSegmentsReq) (*CatSegmentsResp, error) {
	if req == nil {
		req = &CatSegmentsReq{}
	}

	var (
		data CatSegmentsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Segments); err != nil {
		return &data, err
	}

	return &data, nil
}

// Shards executes a /_cat/shards request with the optional CatShardsReq
func (c catClient) Shards(ctx context.Context, req *CatShardsReq) (*CatShardsResp, error) {
	if req == nil {
		req = &CatShardsReq{}
	}

	var (
		data CatShardsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Shards); err != nil {
		return &data, err
	}

	return &data, nil
}

// Snapshots executes a /_cat/snapshots request with the required CatSnapshotsReq
func (c catClient) Snapshots(ctx context.Context, req CatSnapshotsReq) (*CatSnapshotsResp, error) {
	var (
		data CatSnapshotsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Snapshots); err != nil {
		return &data, err
	}

	return &data, nil
}

// Tasks executes a /_cat/tasks request with the optional CatTasksReq
func (c catClient) Tasks(ctx context.Context, req *CatTasksReq) (*CatTasksResp, error) {
	if req == nil {
		req = &CatTasksReq{}
	}

	var (
		data CatTasksResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Tasks); err != nil {
		return &data, err
	}

	return &data, nil
}

// Templates executes a /_cat/templates request with the optional CatTemplatesReq
func (c catClient) Templates(ctx context.Context, req *CatTemplatesReq) (*CatTemplatesResp, error) {
	if req == nil {
		req = &CatTemplatesReq{}
	}

	var (
		data CatTemplatesResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Templates); err != nil {
		return &data, err
	}

	return &data, nil
}

// ThreadPool executes a /_cat/thread_pool request with the optional CatThreadPoolReq
func (c catClient) ThreadPool(ctx context.Context, req *CatThreadPoolReq) (*CatThreadPoolResp, error) {
	if req == nil {
		req = &CatThreadPoolReq{}
	}

	var (
		data CatThreadPoolResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.ThreadPool); err != nil {
		return &data, err
	}

	return &data, nil
}
