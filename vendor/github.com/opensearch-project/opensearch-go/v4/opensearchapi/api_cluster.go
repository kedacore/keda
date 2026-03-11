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

type clusterClient struct {
	apiClient *Client
}

// AllocationExplain executes a /_cluster/allocation/explain request with the optional ClusterAllocationExplainReq
func (c clusterClient) AllocationExplain(ctx context.Context, req *ClusterAllocationExplainReq) (*ClusterAllocationExplainResp, error) {
	if req == nil {
		req = &ClusterAllocationExplainReq{}
	}

	var (
		data ClusterAllocationExplainResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Health executes a /_cluster/health request with the optional ClusterHealthReq
func (c clusterClient) Health(ctx context.Context, req *ClusterHealthReq) (*ClusterHealthResp, error) {
	if req == nil {
		req = &ClusterHealthReq{}
	}

	var (
		data ClusterHealthResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// PendingTasks executes a /_cluster/pending_tasks request with the optional ClusterPendingTasksReq
func (c clusterClient) PendingTasks(ctx context.Context, req *ClusterPendingTasksReq) (*ClusterPendingTasksResp, error) {
	if req == nil {
		req = &ClusterPendingTasksReq{}
	}

	var (
		data ClusterPendingTasksResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// GetSettings executes a /_cluster/settings request with the optional ClusterGetSettingsReq
func (c clusterClient) GetSettings(ctx context.Context, req *ClusterGetSettingsReq) (*ClusterGetSettingsResp, error) {
	if req == nil {
		req = &ClusterGetSettingsReq{}
	}

	var (
		data ClusterGetSettingsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// PutSettings executes a /_cluster/settings request with the required ClusterPutSettingsReq
func (c clusterClient) PutSettings(ctx context.Context, req ClusterPutSettingsReq) (*ClusterPutSettingsResp, error) {
	var (
		data ClusterPutSettingsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// State executes a /_cluster/state request with the optional ClusterStateReq
func (c clusterClient) State(ctx context.Context, req *ClusterStateReq) (*ClusterStateResp, error) {
	if req == nil {
		req = &ClusterStateReq{}
	}

	var (
		data ClusterStateResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Stats executes a /_cluster/stats request with the optional ClusterStatsReq
func (c clusterClient) Stats(ctx context.Context, req *ClusterStatsReq) (*ClusterStatsResp, error) {
	if req == nil {
		req = &ClusterStatsReq{}
	}

	var (
		data ClusterStatsResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// Reroute executes a /_cluster/reroute request with the required ClusterRerouteReq
func (c clusterClient) Reroute(ctx context.Context, req ClusterRerouteReq) (*ClusterRerouteResp, error) {
	var (
		data ClusterRerouteResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// PostVotingConfigExclusions executes a /_cluster/voting_config_exclusions request with the optional ClusterPostVotingConfigExclusionsReq
func (c clusterClient) PostVotingConfigExclusions(
	ctx context.Context,
	req ClusterPostVotingConfigExclusionsReq,
) (*opensearch.Response, error) {
	var (
		resp *opensearch.Response
		err  error
	)
	if resp, err = c.apiClient.do(ctx, req, nil); err != nil {
		return resp, err
	}

	return resp, nil
}

// DeleteVotingConfigExclusions executes a /_cluster/voting_config_exclusions request
// with the optional ClusterDeleteVotingConfigExclusionsReq
func (c clusterClient) DeleteVotingConfigExclusions(
	ctx context.Context,
	req ClusterDeleteVotingConfigExclusionsReq,
) (*opensearch.Response, error) {
	var (
		resp *opensearch.Response
		err  error
	)
	if resp, err = c.apiClient.do(ctx, req, nil); err != nil {
		return resp, err
	}

	return resp, nil
}

// PutDecommission executes a /_cluster/decommission/awareness request with the optional ClusterPutDecommissionReq
func (c clusterClient) PutDecommission(ctx context.Context, req ClusterPutDecommissionReq) (*ClusterPutDecommissionResp, error) {
	var (
		data ClusterPutDecommissionResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// DeleteDecommission executes a /_cluster/decommission/awareness request with the optional ClusterDeleteDecommissionReq
func (c clusterClient) DeleteDecommission(
	ctx context.Context,
	req *ClusterDeleteDecommissionReq,
) (*ClusterDeleteDecommissionResp, error) {
	if req == nil {
		req = &ClusterDeleteDecommissionReq{}
	}

	var (
		data ClusterDeleteDecommissionResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data); err != nil {
		return &data, err
	}

	return &data, nil
}

// GetDecommission executes a /_cluster/decommission/awareness request with the optional ClusterGetDecommissionReq
func (c clusterClient) GetDecommission(ctx context.Context, req ClusterGetDecommissionReq) (*ClusterGetDecommissionResp, error) {
	var (
		data ClusterGetDecommissionResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Values); err != nil {
		return &data, err
	}

	return &data, nil
}

// RemoteInfo executes a /_remote/info request with the optional ClusterRemoteInfoReq
func (c clusterClient) RemoteInfo(ctx context.Context, req *ClusterRemoteInfoReq) (*ClusterRemoteInfoResp, error) {
	if req == nil {
		req = &ClusterRemoteInfoReq{}
	}

	var (
		data ClusterRemoteInfoResp
		err  error
	)
	if data.response, err = c.apiClient.do(ctx, req, &data.Clusters); err != nil {
		return &data, err
	}

	return &data, nil
}
