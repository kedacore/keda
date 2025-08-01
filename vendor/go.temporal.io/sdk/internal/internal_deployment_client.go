package internal

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/deployment/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
)

type (
	// deploymentClient is the client for managing deployments.
	deploymentClient struct {
		workflowClient *WorkflowClient
	}

	// deploymentListIteratorImpl is the implementation of [DeploymentListIterator].
	// Adapted from [scheduleListIteratorImpl].
	deploymentListIteratorImpl struct {
		// nextDeploymentIndex - Local index to cached deployments.
		nextDeploymentIndex int

		// err - Error from getting the last page of deployments.
		err error

		// response - Last page of deployments from server.
		response *workflowservice.ListDeploymentsResponse

		// paginate - Function to get the next page of deployment from server.
		paginate func(nexttoken []byte) (*workflowservice.ListDeploymentsResponse, error)
	}
)

func (iter *deploymentListIteratorImpl) HasNext() bool {
	if iter.err == nil &&
		(iter.response == nil ||
			(iter.nextDeploymentIndex >= len(iter.response.Deployments) && len(iter.response.NextPageToken) > 0)) {
		iter.response, iter.err = iter.paginate(iter.response.GetNextPageToken())
		iter.nextDeploymentIndex = 0
	}

	return iter.nextDeploymentIndex < len(iter.response.GetDeployments()) || iter.err != nil
}

func (iter *deploymentListIteratorImpl) Next() (*DeploymentListEntry, error) {
	if !iter.HasNext() {
		panic("DeploymentListIterator Next() called without checking HasNext()")
	} else if iter.err != nil {
		return nil, iter.err
	}
	deployment := iter.response.Deployments[iter.nextDeploymentIndex]
	iter.nextDeploymentIndex++
	return deploymentListEntryFromProto(deployment), nil
}

func deploymentFromProto(deployment *deployment.Deployment) Deployment {
	return Deployment{
		SeriesName: deployment.GetSeriesName(),
		BuildID:    deployment.GetBuildId(),
	}
}

func deploymentToProto(deploymentID Deployment) *deployment.Deployment {
	return &deployment.Deployment{
		SeriesName: deploymentID.SeriesName,
		BuildId:    deploymentID.BuildID,
	}
}

func deploymentListEntryFromProto(deployment *deployment.DeploymentListInfo) *DeploymentListEntry {
	return &DeploymentListEntry{
		Deployment: deploymentFromProto(deployment.GetDeployment()),
		CreateTime: deployment.GetCreateTime().AsTime(),
		IsCurrent:  deployment.GetIsCurrent(),
	}
}

func deploymentTaskQueuesInfoFromProto(tqsInfo []*deployment.DeploymentInfo_TaskQueueInfo) []DeploymentTaskQueueInfo {
	result := []DeploymentTaskQueueInfo{}
	for _, info := range tqsInfo {
		result = append(result, DeploymentTaskQueueInfo{
			Name:            info.GetName(),
			Type:            TaskQueueType(info.GetType()),
			FirstPollerTime: info.GetFirstPollerTime().AsTime(),
		})
	}
	return result
}

func deploymentInfoFromProto(deploymentInfo *deployment.DeploymentInfo) DeploymentInfo {
	return DeploymentInfo{
		Deployment:      deploymentFromProto(deploymentInfo.GetDeployment()),
		CreateTime:      deploymentInfo.GetCreateTime().AsTime(),
		IsCurrent:       deploymentInfo.GetIsCurrent(),
		TaskQueuesInfos: deploymentTaskQueuesInfoFromProto(deploymentInfo.GetTaskQueueInfos()),
		Metadata:        deploymentInfo.GetMetadata(),
	}
}

func deploymentDescriptionFromProto(deploymentInfo *deployment.DeploymentInfo) DeploymentDescription {
	return DeploymentDescription{
		DeploymentInfo: deploymentInfoFromProto(deploymentInfo),
	}
}

func deploymentReachabilityInfoFromProto(response *workflowservice.GetDeploymentReachabilityResponse) DeploymentReachabilityInfo {
	return DeploymentReachabilityInfo{
		DeploymentInfo: deploymentInfoFromProto(response.GetDeploymentInfo()),
		Reachability:   DeploymentReachability(response.GetReachability()),
		LastUpdateTime: response.GetLastUpdateTime().AsTime(),
	}
}

func deploymentGetCurrentResponseFromProto(deploymentInfo *deployment.DeploymentInfo) DeploymentGetCurrentResponse {
	return DeploymentGetCurrentResponse{
		DeploymentInfo: deploymentInfoFromProto(deploymentInfo),
	}
}

func deploymentMetadataUpdateToProto(dc converter.DataConverter, update DeploymentMetadataUpdate) *deployment.UpdateDeploymentMetadata {
	upsertEntries := make(map[string]*common.Payload)

	for k, v := range update.UpsertEntries {
		if enc, ok := v.(*common.Payload); ok {
			upsertEntries[k] = enc
		} else {
			dataConverter := dc
			if dataConverter == nil {
				dataConverter = converter.GetDefaultDataConverter()
			}
			metadataBytes, err := dataConverter.ToPayload(v)
			if err != nil {
				panic(fmt.Sprintf("encode deployment metadata error: %v", err.Error()))
			}
			upsertEntries[k] = metadataBytes
		}
	}

	return &deployment.UpdateDeploymentMetadata{
		UpsertEntries: upsertEntries,
		RemoveEntries: update.RemoveEntries,
	}
}

func (dc *deploymentClient) List(ctx context.Context, options DeploymentListOptions) (DeploymentListIterator, error) {
	paginate := func(nextToken []byte) (*workflowservice.ListDeploymentsResponse, error) {
		if err := dc.workflowClient.ensureInitialized(ctx); err != nil {
			return nil, err
		}
		if dc.workflowClient.namespace == "" {
			return nil, errors.New("missing namespace argument")
		}
		grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
		defer cancel()
		request := &workflowservice.ListDeploymentsRequest{
			Namespace:     dc.workflowClient.namespace,
			PageSize:      int32(options.PageSize),
			NextPageToken: nextToken,
			SeriesName:    options.SeriesName,
		}

		return dc.workflowClient.workflowService.ListDeployments(grpcCtx, request)
	}

	return &deploymentListIteratorImpl{
		paginate: paginate,
	}, nil
}

func validateDeployment(deployment Deployment) error {
	if deployment.BuildID == "" {
		return errors.New("missing build id in deployment argument")
	}

	if deployment.SeriesName == "" {
		return errors.New("missing series name in deployment argument")
	}

	return nil
}

func (dc *deploymentClient) Describe(ctx context.Context, options DeploymentDescribeOptions) (DeploymentDescription, error) {
	if err := dc.workflowClient.ensureInitialized(ctx); err != nil {
		return DeploymentDescription{}, err
	}
	if dc.workflowClient.namespace == "" {
		return DeploymentDescription{}, errors.New("missing namespace argument")
	}
	if err := validateDeployment(options.Deployment); err != nil {
		return DeploymentDescription{}, err
	}

	request := &workflowservice.DescribeDeploymentRequest{
		Namespace:  dc.workflowClient.namespace,
		Deployment: deploymentToProto(options.Deployment),
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	resp, err := dc.workflowClient.workflowService.DescribeDeployment(grpcCtx, request)
	if err != nil {
		return DeploymentDescription{}, err
	}

	return deploymentDescriptionFromProto(resp.GetDeploymentInfo()), nil
}

func (dc *deploymentClient) GetReachability(ctx context.Context, options DeploymentGetReachabilityOptions) (DeploymentReachabilityInfo, error) {
	if err := dc.workflowClient.ensureInitialized(ctx); err != nil {
		return DeploymentReachabilityInfo{}, err
	}
	if dc.workflowClient.namespace == "" {
		return DeploymentReachabilityInfo{}, errors.New("missing namespace argument")
	}
	if err := validateDeployment(options.Deployment); err != nil {
		return DeploymentReachabilityInfo{}, err
	}

	request := &workflowservice.GetDeploymentReachabilityRequest{
		Namespace:  dc.workflowClient.namespace,
		Deployment: deploymentToProto(options.Deployment),
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	resp, err := dc.workflowClient.workflowService.GetDeploymentReachability(grpcCtx, request)
	if err != nil {
		return DeploymentReachabilityInfo{}, err
	}

	return deploymentReachabilityInfoFromProto(resp), nil
}

func (dc *deploymentClient) GetCurrent(ctx context.Context, options DeploymentGetCurrentOptions) (DeploymentGetCurrentResponse, error) {
	if err := dc.workflowClient.ensureInitialized(ctx); err != nil {
		return DeploymentGetCurrentResponse{}, err
	}
	if dc.workflowClient.namespace == "" {
		return DeploymentGetCurrentResponse{}, errors.New("missing namespace argument")
	}
	if options.SeriesName == "" {
		return DeploymentGetCurrentResponse{}, errors.New("missing series name argument")
	}

	request := &workflowservice.GetCurrentDeploymentRequest{
		Namespace:  dc.workflowClient.namespace,
		SeriesName: options.SeriesName,
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	resp, err := dc.workflowClient.workflowService.GetCurrentDeployment(grpcCtx, request)
	if err != nil {
		return DeploymentGetCurrentResponse{}, err
	}
	return deploymentGetCurrentResponseFromProto(resp.GetCurrentDeploymentInfo()), nil
}

func (dc *deploymentClient) SetCurrent(ctx context.Context, options DeploymentSetCurrentOptions) (DeploymentSetCurrentResponse, error) {
	if err := dc.workflowClient.ensureInitialized(ctx); err != nil {
		return DeploymentSetCurrentResponse{}, err
	}
	if dc.workflowClient.namespace == "" {
		return DeploymentSetCurrentResponse{}, errors.New("missing namespace argument")
	}
	if err := validateDeployment(options.Deployment); err != nil {
		return DeploymentSetCurrentResponse{}, err
	}

	request := &workflowservice.SetCurrentDeploymentRequest{
		Namespace:      dc.workflowClient.namespace,
		Deployment:     deploymentToProto(options.Deployment),
		Identity:       dc.workflowClient.identity,
		UpdateMetadata: deploymentMetadataUpdateToProto(dc.workflowClient.dataConverter, options.MetadataUpdate),
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	resp, err := dc.workflowClient.workflowService.SetCurrentDeployment(grpcCtx, request)
	if err != nil {
		return DeploymentSetCurrentResponse{}, err
	}

	return DeploymentSetCurrentResponse{
		Current:  deploymentInfoFromProto(resp.GetCurrentDeploymentInfo()),
		Previous: deploymentInfoFromProto(resp.GetPreviousDeploymentInfo()),
	}, nil
}
