package internal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/deployment/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// A reserved identifier of unversioned workers.
const WorkerDeploymentUnversioned = "__unversioned__"

// A reserved separator for Worker Deployment Versions.
const WorkerDeploymentVersionSeparator = "."

var errBuildIdCantBeEmpty = fmt.Errorf("BuildID cannot be empty")

// safeAsTime ensures that a nil proto timestamp makes `IsZero()` true.
func safeAsTime(timestamp *timestamppb.Timestamp) time.Time {
	if timestamp == nil {
		return time.Time{}
	} else {
		return timestamp.AsTime()
	}
}

type (
	// WorkerDeploymentClient is the client for managing worker deployments.
	workerDeploymentClient struct {
		workflowClient *WorkflowClient
	}

	// workerDeploymentHandleImpl is the implementation of [WorkerDeploymentHandle]
	workerDeploymentHandleImpl struct {
		Name           string
		workflowClient *WorkflowClient
	}

	// workerDeploymentListIteratorImpl is the implementation of [WorkerDeploymentListIterator].
	// Adapted from [scheduleListIteratorImpl].
	workerDeploymentListIteratorImpl struct {
		// nextWorkerDeploymentIndex - Local index to cached deployments.
		nextWorkerDeploymentIndex int

		// err - Error from getting the last page of deployments.
		err error

		// response - Last page of deployments from server.
		response *workflowservice.ListWorkerDeploymentsResponse

		// paginate - Function to get the next page of deployment from server.
		paginate func(nexttoken []byte) (*workflowservice.ListWorkerDeploymentsResponse, error)
	}
)

func (iter *workerDeploymentListIteratorImpl) HasNext() bool {
	if iter.err == nil &&
		(iter.response == nil ||
			(iter.nextWorkerDeploymentIndex >= len(iter.response.WorkerDeployments) && len(iter.response.NextPageToken) > 0)) {
		iter.response, iter.err = iter.paginate(iter.response.GetNextPageToken())
		iter.nextWorkerDeploymentIndex = 0
	}

	return iter.nextWorkerDeploymentIndex < len(iter.response.GetWorkerDeployments()) || iter.err != nil
}

func (iter *workerDeploymentListIteratorImpl) Next() (*WorkerDeploymentListEntry, error) {
	if !iter.HasNext() {
		panic("WorkerDeploymentListIterator Next() called without checking HasNext()")
	} else if iter.err != nil {
		return nil, iter.err
	}
	deployment := iter.response.WorkerDeployments[iter.nextWorkerDeploymentIndex]
	iter.nextWorkerDeploymentIndex++
	return workerDeploymentListEntryFromProto(deployment), nil
}

func workerDeploymentRoutingConfigFromProto(routingConfig *deployment.RoutingConfig) WorkerDeploymentRoutingConfig {
	if routingConfig == nil {
		return WorkerDeploymentRoutingConfig{}
	}

	return WorkerDeploymentRoutingConfig{
		CurrentVersion: workerDeploymentVersionFromProtoOrString(
			//lint:ignore SA1019 ignore deprecated versioning APIs
			routingConfig.CurrentDeploymentVersion, routingConfig.CurrentVersion),
		RampingVersion: workerDeploymentVersionFromProtoOrString(
			//lint:ignore SA1019 ignore deprecated versioning APIs
			routingConfig.RampingDeploymentVersion, routingConfig.RampingVersion),
		RampingVersionPercentage:            routingConfig.GetRampingVersionPercentage(),
		CurrentVersionChangedTime:           safeAsTime(routingConfig.GetCurrentVersionChangedTime()),
		RampingVersionChangedTime:           safeAsTime(routingConfig.GetRampingVersionChangedTime()),
		RampingVersionPercentageChangedTime: safeAsTime(routingConfig.GetRampingVersionPercentageChangedTime()),
	}
}

func workerDeploymentListEntryFromProto(summary *workflowservice.ListWorkerDeploymentsResponse_WorkerDeploymentSummary) *WorkerDeploymentListEntry {
	return &WorkerDeploymentListEntry{
		Name:          summary.GetName(),
		CreateTime:    safeAsTime(summary.GetCreateTime()),
		RoutingConfig: workerDeploymentRoutingConfigFromProto(summary.GetRoutingConfig()),
	}
}

func workerDeploymentVersionSummariesFromProto(summaries []*deployment.WorkerDeploymentInfo_WorkerDeploymentVersionSummary) []WorkerDeploymentVersionSummary {
	result := []WorkerDeploymentVersionSummary{}
	for _, summary := range summaries {
		version := workerDeploymentVersionFromProtoOrString(
			//lint:ignore SA1019 ignore deprecated versioning APIs
			summary.DeploymentVersion, summary.Version)
		if version == nil {
			// Shouldn't receive any summary like this
			continue
		}

		result = append(result, WorkerDeploymentVersionSummary{
			Version:        *version,
			CreateTime:     safeAsTime(summary.CreateTime),
			DrainageStatus: WorkerDeploymentVersionDrainageStatus(summary.GetDrainageStatus()),
		})
	}
	return result
}

func workerDeploymentInfoFromProto(info *deployment.WorkerDeploymentInfo) WorkerDeploymentInfo {
	if info == nil {
		return WorkerDeploymentInfo{}
	}

	return WorkerDeploymentInfo{
		Name:                 info.Name,
		CreateTime:           safeAsTime(info.CreateTime),
		VersionSummaries:     workerDeploymentVersionSummariesFromProto(info.VersionSummaries),
		RoutingConfig:        workerDeploymentRoutingConfigFromProto(info.RoutingConfig),
		LastModifierIdentity: info.LastModifierIdentity,
		ManagerIdentity:      info.ManagerIdentity,
	}

}

func (h *workerDeploymentHandleImpl) validate() error {
	if h.Name == "" {
		return errors.New("missing worker deployment name in handle")
	}
	if strings.Contains(h.Name, WorkerDeploymentVersionSeparator) {
		return fmt.Errorf("worker deployment name contains reserved separator '%v'", WorkerDeploymentVersionSeparator)
	}
	if h.workflowClient.namespace == "" {
		return errors.New("missing namespace argument in handle")
	}

	return nil
}

func (h *workerDeploymentHandleImpl) buildIdToVersionStr(buildId string) string {
	if buildId == "" {
		return WorkerDeploymentUnversioned
	}
	return h.Name + WorkerDeploymentVersionSeparator + buildId
}

func (h *workerDeploymentHandleImpl) Describe(ctx context.Context, options WorkerDeploymentDescribeOptions) (WorkerDeploymentDescribeResponse, error) {
	if err := h.validate(); err != nil {
		return WorkerDeploymentDescribeResponse{}, err
	}
	if err := h.workflowClient.ensureInitialized(ctx); err != nil {
		return WorkerDeploymentDescribeResponse{}, err
	}

	request := &workflowservice.DescribeWorkerDeploymentRequest{
		Namespace:      h.workflowClient.namespace,
		DeploymentName: h.Name,
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	resp, err := h.workflowClient.workflowService.DescribeWorkerDeployment(grpcCtx, request)
	if err != nil {
		return WorkerDeploymentDescribeResponse{}, err
	}

	return WorkerDeploymentDescribeResponse{
		ConflictToken: resp.GetConflictToken(),
		Info:          workerDeploymentInfoFromProto(resp.GetWorkerDeploymentInfo()),
	}, nil
}

func (h *workerDeploymentHandleImpl) SetCurrentVersion(ctx context.Context, options WorkerDeploymentSetCurrentVersionOptions) (WorkerDeploymentSetCurrentVersionResponse, error) {
	if err := h.validate(); err != nil {
		return WorkerDeploymentSetCurrentVersionResponse{}, err
	}
	if err := h.workflowClient.ensureInitialized(ctx); err != nil {
		return WorkerDeploymentSetCurrentVersionResponse{}, err
	}

	identity := h.workflowClient.identity
	if options.Identity != "" {
		identity = options.Identity
	}

	request := &workflowservice.SetWorkerDeploymentCurrentVersionRequest{
		Namespace:               h.workflowClient.namespace,
		DeploymentName:          h.Name,
		Version:                 h.buildIdToVersionStr(options.BuildID),
		BuildId:                 options.BuildID,
		ConflictToken:           options.ConflictToken,
		Identity:                identity,
		IgnoreMissingTaskQueues: options.IgnoreMissingTaskQueues,
		AllowNoPollers:          options.AllowNoPollers,
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	resp, err := h.workflowClient.workflowService.SetWorkerDeploymentCurrentVersion(grpcCtx, request)
	if err != nil {
		return WorkerDeploymentSetCurrentVersionResponse{}, err
	}

	return WorkerDeploymentSetCurrentVersionResponse{
		ConflictToken: resp.GetConflictToken(),
		PreviousVersion: workerDeploymentVersionFromProtoOrString(
			//lint:ignore SA1019 ignore deprecated versioning APIs
			resp.PreviousDeploymentVersion, resp.PreviousVersion),
	}, nil
}

func (h *workerDeploymentHandleImpl) SetRampingVersion(ctx context.Context, options WorkerDeploymentSetRampingVersionOptions) (WorkerDeploymentSetRampingVersionResponse, error) {
	if err := h.validate(); err != nil {
		return WorkerDeploymentSetRampingVersionResponse{}, err
	}
	if err := h.workflowClient.ensureInitialized(ctx); err != nil {
		return WorkerDeploymentSetRampingVersionResponse{}, err
	}

	identity := h.workflowClient.identity
	if options.Identity != "" {
		identity = options.Identity
	}

	request := &workflowservice.SetWorkerDeploymentRampingVersionRequest{
		Namespace:               h.workflowClient.namespace,
		DeploymentName:          h.Name,
		Version:                 h.buildIdToVersionStr(options.BuildID),
		BuildId:                 options.BuildID,
		Percentage:              options.Percentage,
		ConflictToken:           options.ConflictToken,
		Identity:                identity,
		IgnoreMissingTaskQueues: options.IgnoreMissingTaskQueues,
		AllowNoPollers:          options.AllowNoPollers,
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	resp, err := h.workflowClient.workflowService.SetWorkerDeploymentRampingVersion(grpcCtx, request)
	if err != nil {
		return WorkerDeploymentSetRampingVersionResponse{}, err
	}

	return WorkerDeploymentSetRampingVersionResponse{
		ConflictToken: resp.GetConflictToken(),
		PreviousVersion: workerDeploymentVersionFromProtoOrString(
			//lint:ignore SA1019 ignore deprecated versioning APIs
			resp.PreviousDeploymentVersion, resp.PreviousVersion),
		PreviousPercentage: resp.GetPreviousPercentage(),
	}, nil

}

func (h *workerDeploymentHandleImpl) SetManagerIdentity(ctx context.Context, options WorkerDeploymentSetManagerIdentityOptions) (WorkerDeploymentSetManagerIdentityResponse, error) {
	if err := h.validate(); err != nil {
		return WorkerDeploymentSetManagerIdentityResponse{}, err
	}
	if err := h.workflowClient.ensureInitialized(ctx); err != nil {
		return WorkerDeploymentSetManagerIdentityResponse{}, err
	}

	identity := h.workflowClient.identity
	if options.Identity != "" {
		identity = options.Identity
	}

	request := &workflowservice.SetWorkerDeploymentManagerRequest{
		Namespace:      h.workflowClient.namespace,
		DeploymentName: h.Name,
		ConflictToken:  options.ConflictToken,
		Identity:       identity,
	}
	if options.Self {
		if options.ManagerIdentity != "" {
			return WorkerDeploymentSetManagerIdentityResponse{}, fmt.Errorf("invalid input: if Self is true, ManagerIdentity must be empty but was '%s'", options.ManagerIdentity)
		}
		request.NewManagerIdentity = &workflowservice.SetWorkerDeploymentManagerRequest_Self{Self: true}
	} else {
		request.NewManagerIdentity = &workflowservice.SetWorkerDeploymentManagerRequest_ManagerIdentity{ManagerIdentity: options.ManagerIdentity}
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	resp, err := h.workflowClient.workflowService.SetWorkerDeploymentManager(grpcCtx, request)
	if err != nil {
		return WorkerDeploymentSetManagerIdentityResponse{}, err
	}

	return WorkerDeploymentSetManagerIdentityResponse{
		ConflictToken:           resp.GetConflictToken(),
		PreviousManagerIdentity: resp.GetPreviousManagerIdentity(),
	}, nil

}

func workerDeploymentTaskQueuesInfosFromProto(tqInfos []*deployment.WorkerDeploymentVersionInfo_VersionTaskQueueInfo) []WorkerDeploymentTaskQueueInfo {
	result := []WorkerDeploymentTaskQueueInfo{}
	for _, info := range tqInfos {
		result = append(result, WorkerDeploymentTaskQueueInfo{
			Name: info.GetName(),
			Type: TaskQueueType(info.GetType()),
		})
	}
	return result
}

func workerDeploymentDrainageInfoFromProto(drainageInfo *deployment.VersionDrainageInfo) *WorkerDeploymentVersionDrainageInfo {
	if drainageInfo == nil {
		return nil
	}
	return &WorkerDeploymentVersionDrainageInfo{
		DrainageStatus:  WorkerDeploymentVersionDrainageStatus(drainageInfo.Status),
		LastChangedTime: safeAsTime(drainageInfo.LastChangedTime),
		LastCheckedTime: safeAsTime(drainageInfo.LastCheckedTime),
	}
}

func workerDeploymentVersionInfoFromProto(info *deployment.WorkerDeploymentVersionInfo) WorkerDeploymentVersionInfo {
	if info == nil {
		return WorkerDeploymentVersionInfo{}
	}
	//lint:ignore SA1019 ignore deprecated versioning APIs
	version := workerDeploymentVersionFromProtoOrString(info.DeploymentVersion, info.Version)
	if version == nil {
		// Should never happen unless server is sending junk data
		version = &WorkerDeploymentVersion{}
	}
	return WorkerDeploymentVersionInfo{
		Version:            *version,
		CreateTime:         safeAsTime(info.CreateTime),
		RoutingChangedTime: safeAsTime(info.RoutingChangedTime),
		CurrentSinceTime:   safeAsTime(info.CurrentSinceTime),
		RampingSinceTime:   safeAsTime(info.RampingSinceTime),
		RampPercentage:     info.RampPercentage,
		TaskQueuesInfos:    workerDeploymentTaskQueuesInfosFromProto(info.TaskQueueInfos),
		DrainageInfo:       workerDeploymentDrainageInfoFromProto(info.DrainageInfo),
		Metadata:           info.Metadata.GetEntries(),
	}
}

func (h *workerDeploymentHandleImpl) DescribeVersion(ctx context.Context, options WorkerDeploymentDescribeVersionOptions) (WorkerDeploymentVersionDescription, error) {

	if err := h.validate(); err != nil {
		return WorkerDeploymentVersionDescription{}, err
	}
	if options.BuildID == "" {
		return WorkerDeploymentVersionDescription{}, errBuildIdCantBeEmpty
	}
	if err := h.workflowClient.ensureInitialized(ctx); err != nil {
		return WorkerDeploymentVersionDescription{}, err
	}

	request := &workflowservice.DescribeWorkerDeploymentVersionRequest{
		Namespace: h.workflowClient.namespace,
		Version:   h.buildIdToVersionStr(options.BuildID),
		DeploymentVersion: &deployment.WorkerDeploymentVersion{
			BuildId:        options.BuildID,
			DeploymentName: h.Name,
		},
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	resp, err := h.workflowClient.workflowService.DescribeWorkerDeploymentVersion(grpcCtx, request)
	if err != nil {
		return WorkerDeploymentVersionDescription{}, err
	}

	return WorkerDeploymentVersionDescription{
		Info: workerDeploymentVersionInfoFromProto(resp.GetWorkerDeploymentVersionInfo()),
	}, nil
}

func (h *workerDeploymentHandleImpl) DeleteVersion(ctx context.Context, options WorkerDeploymentDeleteVersionOptions) (WorkerDeploymentDeleteVersionResponse, error) {
	if err := h.validate(); err != nil {
		return WorkerDeploymentDeleteVersionResponse{}, err
	}
	if options.BuildID == "" {
		return WorkerDeploymentDeleteVersionResponse{}, errBuildIdCantBeEmpty
	}
	if err := h.workflowClient.ensureInitialized(ctx); err != nil {
		return WorkerDeploymentDeleteVersionResponse{}, err
	}

	identity := h.workflowClient.identity
	if options.Identity != "" {
		identity = options.Identity
	}

	request := &workflowservice.DeleteWorkerDeploymentVersionRequest{
		Namespace: h.workflowClient.namespace,
		Version:   h.buildIdToVersionStr(options.BuildID),
		DeploymentVersion: &deployment.WorkerDeploymentVersion{
			BuildId:        options.BuildID,
			DeploymentName: h.Name,
		},
		SkipDrainage: options.SkipDrainage,
		Identity:     identity,
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	_, err := h.workflowClient.workflowService.DeleteWorkerDeploymentVersion(grpcCtx, request)
	if err != nil {
		return WorkerDeploymentDeleteVersionResponse{}, err
	}

	return WorkerDeploymentDeleteVersionResponse{}, nil
}

func workerDeploymentUpsertEntriesMetadataToProto(dc converter.DataConverter, update WorkerDeploymentMetadataUpdate) map[string]*common.Payload {
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

	return upsertEntries
}

func (h *workerDeploymentHandleImpl) UpdateVersionMetadata(ctx context.Context, options WorkerDeploymentUpdateVersionMetadataOptions) (WorkerDeploymentUpdateVersionMetadataResponse, error) {
	if err := h.validate(); err != nil {
		return WorkerDeploymentUpdateVersionMetadataResponse{}, err
	}
	if options.Version.BuildID == "" {
		return WorkerDeploymentUpdateVersionMetadataResponse{}, errBuildIdCantBeEmpty
	}
	if err := h.workflowClient.ensureInitialized(ctx); err != nil {
		return WorkerDeploymentUpdateVersionMetadataResponse{}, err
	}

	request := &workflowservice.UpdateWorkerDeploymentVersionMetadataRequest{
		Namespace:         h.workflowClient.namespace,
		Version:           options.Version.toCanonicalString(),
		DeploymentVersion: options.Version.toProto(),
		UpsertEntries:     workerDeploymentUpsertEntriesMetadataToProto(h.workflowClient.dataConverter, options.MetadataUpdate),
		RemoveEntries:     options.MetadataUpdate.RemoveEntries,
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	resp, err := h.workflowClient.workflowService.UpdateWorkerDeploymentVersionMetadata(grpcCtx, request)
	if err != nil {
		return WorkerDeploymentUpdateVersionMetadataResponse{}, err
	}

	return WorkerDeploymentUpdateVersionMetadataResponse{
		Metadata: resp.GetMetadata().GetEntries(),
	}, nil
}

func (wdc *workerDeploymentClient) List(ctx context.Context, options WorkerDeploymentListOptions) (WorkerDeploymentListIterator, error) {
	paginate := func(nextToken []byte) (*workflowservice.ListWorkerDeploymentsResponse, error) {
		if err := wdc.workflowClient.ensureInitialized(ctx); err != nil {
			return nil, err
		}
		if wdc.workflowClient.namespace == "" {
			return nil, errors.New("missing namespace argument")
		}
		grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
		defer cancel()
		request := &workflowservice.ListWorkerDeploymentsRequest{
			Namespace:     wdc.workflowClient.namespace,
			PageSize:      int32(options.PageSize),
			NextPageToken: nextToken,
		}

		return wdc.workflowClient.workflowService.ListWorkerDeployments(grpcCtx, request)
	}

	return &workerDeploymentListIteratorImpl{
		paginate: paginate,
	}, nil
}

func (wdc *workerDeploymentClient) Delete(ctx context.Context, options WorkerDeploymentDeleteOptions) (WorkerDeploymentDeleteResponse, error) {
	if err := wdc.workflowClient.ensureInitialized(ctx); err != nil {
		return WorkerDeploymentDeleteResponse{}, err
	}
	if wdc.workflowClient.namespace == "" {
		return WorkerDeploymentDeleteResponse{}, errors.New("missing namespace argument")
	}
	if options.Name == "" {
		return WorkerDeploymentDeleteResponse{}, errors.New("missing worker deployment name argument")
	}

	identity := wdc.workflowClient.identity
	if options.Identity != "" {
		identity = options.Identity
	}

	request := &workflowservice.DeleteWorkerDeploymentRequest{
		Namespace:      wdc.workflowClient.namespace,
		DeploymentName: options.Name,
		Identity:       identity,
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	_, err := wdc.workflowClient.workflowService.DeleteWorkerDeployment(grpcCtx, request)
	if err != nil {
		return WorkerDeploymentDeleteResponse{}, err
	}
	return WorkerDeploymentDeleteResponse{}, nil
}

func (wdc *workerDeploymentClient) GetHandle(name string) WorkerDeploymentHandle {
	return &workerDeploymentHandleImpl{
		Name:           name,
		workflowClient: wdc.workflowClient,
	}
}
