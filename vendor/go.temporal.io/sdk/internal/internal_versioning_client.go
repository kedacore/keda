// The MIT License
//
// Copyright (c) 2024 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package internal

import (
	"errors"
	"time"

	"go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"

	enumspb "go.temporal.io/api/enums/v1"
)

// TaskQueueType specifies which category of tasks are associated with a queue.
type TaskQueueType int

const (
	// TaskQueueTypeUnspecified indicates the task queue type was not specified.
	TaskQueueTypeUnspecified = iota
	// TaskQueueTypeWorkflow indicates the task queue is used for dispatching workflow tasks.
	TaskQueueTypeWorkflow
	// TaskQueueTypeActivity indicates the task queue is used for delivering activity tasks.
	TaskQueueTypeActivity
	// TaskQueueTypeNexus indicates the task queue is used for dispatching Nexus requests.
	TaskQueueTypeNexus
)

// BuildIDTaskReachability specifies which category of tasks may reach a versioned worker of a certain Build ID.
//
// Note: future activities who inherit their workflow's Build ID but not its task queue will not be
// accounted for reachability as server cannot know if they'll happen as they do not use
// assignment rules of their task queue. Same goes for Child Workflows or Continue-As-New Workflows
// who inherit the parent/previous workflow's Build ID but not its task queue. In those cases, make
// sure to query reachability for the parent/previous workflow's task queue as well.
type BuildIDTaskReachability int

const (
	// BuildIDTaskReachabilityUnspecified indicates that task reachability was not reported.
	BuildIDTaskReachabilityUnspecified = iota
	// BuildIDTaskReachabilityReachable indicates that this Build ID may be used by new workflows or activities
	// (based on versioning rules), or there are open workflows or backlogged activities assigned to it.
	BuildIDTaskReachabilityReachable
	// BuildIDTaskReachabilityClosedWorkflowsOnly specifies that this Build ID does not have open workflows
	// and is not reachable by new workflows, but MAY have closed workflows within the namespace retention period.
	// Not applicable to activity-only task queues.
	BuildIDTaskReachabilityClosedWorkflowsOnly
	// BuildIDTaskReachabilityUnreachable indicates that this Build ID is not used for new executions, nor
	// it has been used by any existing execution within the retention period.
	BuildIDTaskReachabilityUnreachable
)

type (
	// TaskQueueVersionSelection is a task queue filter based on versioning.
	// It is an optional component of [DescribeTaskQueueEnhancedOptions].
	TaskQueueVersionSelection struct {
		// Include specific Build IDs.
		BuildIDs []string
		// Include the unversioned queue.
		Unversioned bool
		// Include all active versions. A version is considered active if, in the last few minutes,
		// it has had new tasks or polls, or it has been the subject of certain task queue API calls.
		AllActive bool
	}

	// DescribeTaskQueueEnhancedOptions is the input to [Client.DescribeTaskQueueEnhanced].
	DescribeTaskQueueEnhancedOptions struct {
		// Name of the task queue. Sticky queues are not supported.
		TaskQueue string
		// An optional queue selector based on versioning. If not provided,
		// the result for the default Build ID will be returned. The default
		// Build ID is the one mentioned in the first unconditional Assignment Rule.
		// If there is no default Build ID, the result for the
		// unversioned queue will be returned.
		Versions *TaskQueueVersionSelection
		// Task queue types to report info about. If not specified, all types are considered.
		TaskQueueTypes []TaskQueueType
		// Include list of pollers for requested task queue types and versions.
		ReportPollers bool
		// Include task reachability for the requested versions and all task types
		// (task reachability is not reported per task type).
		ReportTaskReachability bool
		// Include task queue stats for requested task queue types and versions.
		ReportStats bool
	}

	// WorkerVersionCapabilities includes a worker's build identifier
	// and whether it is choosing to use the versioning feature.
	// It is an optional component of [TaskQueuePollerInfo].
	WorkerVersionCapabilities struct {
		// Build ID of the worker.
		BuildID string
		// Whether the worker is using the versioning feature.
		UseVersioning bool
		// An identifier to group task queues based on Build ID.
		DeploymentSeriesName string
	}

	// TaskQueuePollerInfo provides information about a worker/client polling a task queue.
	// It is used by [TaskQueueTypeInfo].
	TaskQueuePollerInfo struct {
		// Time of the last poll. A value of zero means it was not set.
		LastAccessTime time.Time
		// The identity of the worker/client who is polling this task queue.
		Identity string
		// Polling rate. A value of zero means it was not set.
		RatePerSecond float64
		// Optional poller versioning capabilities. Available when a worker has opted into the worker versioning feature.
		WorkerVersionCapabilities *WorkerVersionCapabilities
	}

	// TaskQueueStats contains statistics about task queue backlog and activity.
	//
	// For workflow task queue type, this result is partial because tasks sent to sticky queues are not included. Read
	// comments above each metric to understand the impact of sticky queue exclusion on that metric accuracy.
	TaskQueueStats struct {
		// The approximate number of tasks backlogged in this task queue. May count expired tasks but eventually
		// converges to the right value. Can be relied upon for scaling decisions.
		//
		// Special note for workflow task queue type: this metric does not count sticky queue tasks. However, because
		// those tasks only remain valid for a few seconds, the inaccuracy becomes less significant as the backlog size
		// grows.
		ApproximateBacklogCount int64
		// Approximate age of the oldest task in the backlog based on the creation time of the task at the head of
		// the queue. Can be relied upon for scaling decisions.
		//
		// Special note for workflow task queue type: this metric does not count sticky queue tasks. However, because
		// those tasks only remain valid for a few seconds, they should not affect the result when backlog is older than
		// few seconds.
		ApproximateBacklogAge time.Duration
		// Approximate *net* tasks per second added to the backlog, averaging the last 30 seconds. This is calculated as
		// `TasksAddRate - TasksDispatchRate`.
		// A positive value of `X` means the backlog is growing by about `X` tasks per second. A negative `-X` value means the
		// backlog is shrinking by about `X` tasks per second.
		//
		// Special note for workflow task queue type: this metric does not count sticky queue tasks. However, because
		// those tasks only remain valid for a few seconds, the inaccuracy becomes less significant as the backlog size
		// or age grow.
		BacklogIncreaseRate float32
		// Approximate tasks per second added to the task queue, averaging the last 30 seconds. This includes both
		// backlogged and sync-matched tasks, but excludes the Eagerly dispatched workflow and activity tasks (see
		// documentation for `client.StartWorkflowOptions.EnableEagerStart` and `worker.Options.DisableEagerActivities`.)
		//
		// The difference between `TasksAddRate` and `TasksDispatchRate` is a reliable metric for the rate at which
		// backlog grows/shrinks. See `BacklogIncreaseRate`.
		//
		// Special note for workflow task queue type: this metric does not count sticky queue tasks. Hence, the reported
		// value may be significantly lower than the actual number of workflow tasks added. Note that typically, only
		// the first workflow task of each workflow goes to a normal queue, and the rest workflow tasks go to the sticky
		// queue associated with a specific worker instance. Activity tasks always go to normal queues.
		TasksAddRate float32
		// Approximate tasks per second dispatched to workers, averaging the last 30 seconds. This includes both
		// backlogged and sync-matched tasks, but excludes the Eagerly dispatched workflow and activity tasks (see
		// documentation for `client.StartWorkflowOptions.EnableEagerStart` and `worker.Options.DisableEagerActivities`.)
		//
		// The difference between `TasksAddRate` and `TasksDispatchRate` is a reliable metric for the rate at which
		// backlog grows/shrinks. See `BacklogIncreaseRate`.
		//
		// Special note for workflow task queue type: this metric does not count sticky queue tasks. Hence, the reported
		// value may be significantly lower than the actual number of workflow tasks dispatched. Note that typically, only
		// the first workflow task of each workflow goes to a normal queue, and the rest workflow tasks go to the sticky
		// queue associated with a specific worker instance. Activity tasks always go to normal queues.
		TasksDispatchRate float32
	}

	// TaskQueueTypeInfo specifies task queue information per task type and Build ID.
	// It is included in [TaskQueueVersionInfo].
	TaskQueueTypeInfo struct {
		// Poller details for this task queue category.
		Pollers []TaskQueuePollerInfo
		Stats   *TaskQueueStats
	}

	// TaskQueueVersionInfo includes task queue information per Build ID.
	// It is part of [TaskQueueDescription].
	TaskQueueVersionInfo struct {
		// Task queue info per task type.
		TypesInfo map[TaskQueueType]TaskQueueTypeInfo
		// The category of tasks that may reach a versioned worker of a certain Build ID.
		TaskReachability BuildIDTaskReachability
	}

	// TaskQueueDescription is the response to [Client.DescribeTaskQueueEnhanced].
	TaskQueueDescription struct {
		// Task queue information for each Build ID. Empty string as key value means unversioned.
		VersionsInfo map[string]TaskQueueVersionInfo
	}
)

func (o *DescribeTaskQueueEnhancedOptions) validateAndConvertToProto(namespace string) (*workflowservice.DescribeTaskQueueRequest, error) {
	if namespace == "" {
		return nil, errors.New("missing namespace argument")
	}

	if o.TaskQueue == "" {
		return nil, errors.New("missing task queue field")
	}

	taskQueueTypes := make([]enumspb.TaskQueueType, len(o.TaskQueueTypes))
	for i, t := range o.TaskQueueTypes {
		taskQueueTypes[i] = taskQueueTypeToProto(t)
	}

	opt := &workflowservice.DescribeTaskQueueRequest{
		Namespace: namespace,
		TaskQueue: &taskqueuepb.TaskQueue{
			// Sticky queues not supported
			Name: o.TaskQueue,
		},
		ApiMode:                enumspb.DESCRIBE_TASK_QUEUE_MODE_ENHANCED,
		Versions:               taskQueueVersionSelectionToProto(o.Versions),
		TaskQueueTypes:         taskQueueTypes,
		ReportPollers:          o.ReportPollers,
		ReportTaskReachability: o.ReportTaskReachability,
		ReportStats:            o.ReportStats,
	}

	return opt, nil
}

func workerVersionCapabilitiesFromResponse(response *common.WorkerVersionCapabilities) *WorkerVersionCapabilities {
	if response == nil {
		return nil
	}

	return &WorkerVersionCapabilities{
		BuildID:              response.GetBuildId(),
		UseVersioning:        response.GetUseVersioning(),
		DeploymentSeriesName: response.GetDeploymentSeriesName(),
	}
}

func pollerInfoFromResponse(response *taskqueuepb.PollerInfo) TaskQueuePollerInfo {
	if response == nil {
		return TaskQueuePollerInfo{}
	}

	lastAccessTime := time.Time{}
	if response.GetLastAccessTime() != nil {
		lastAccessTime = response.GetLastAccessTime().AsTime()
	}

	return TaskQueuePollerInfo{
		LastAccessTime:            lastAccessTime,
		Identity:                  response.GetIdentity(),
		RatePerSecond:             response.GetRatePerSecond(),
		WorkerVersionCapabilities: workerVersionCapabilitiesFromResponse(response.GetWorkerVersionCapabilities()),
	}
}

func taskQueueTypeInfoFromResponse(response *taskqueuepb.TaskQueueTypeInfo) TaskQueueTypeInfo {
	if response == nil {
		return TaskQueueTypeInfo{}
	}

	pollers := make([]TaskQueuePollerInfo, len(response.GetPollers()))
	for i, pInfo := range response.GetPollers() {
		pollers[i] = pollerInfoFromResponse(pInfo)
	}

	return TaskQueueTypeInfo{
		Pollers: pollers,
		Stats:   statsFromResponse(response.Stats),
	}
}

func statsFromResponse(stats *taskqueuepb.TaskQueueStats) *TaskQueueStats {
	if stats == nil {
		return nil
	}

	return &TaskQueueStats{
		ApproximateBacklogCount: stats.GetApproximateBacklogCount(),
		ApproximateBacklogAge:   stats.GetApproximateBacklogAge().AsDuration(),
		TasksAddRate:            stats.TasksAddRate,
		TasksDispatchRate:       stats.TasksDispatchRate,
		BacklogIncreaseRate:     stats.TasksAddRate - stats.TasksDispatchRate,
	}
}

func taskQueueVersionInfoFromResponse(response *taskqueuepb.TaskQueueVersionInfo) TaskQueueVersionInfo {
	if response == nil {
		return TaskQueueVersionInfo{}
	}

	typesInfo := make(map[TaskQueueType]TaskQueueTypeInfo, len(response.GetTypesInfo()))
	for taskType, tInfo := range response.GetTypesInfo() {
		typesInfo[taskQueueTypeFromProto(enumspb.TaskQueueType(taskType))] = taskQueueTypeInfoFromResponse(tInfo)
	}

	return TaskQueueVersionInfo{
		TypesInfo:        typesInfo,
		TaskReachability: buildIDTaskReachabilityFromProto(response.GetTaskReachability()),
	}
}

func detectTaskQueueEnhancedNotSupported(response *workflowservice.DescribeTaskQueueResponse) error {
	// A server before 1.24 returns a non-enhanced proto, which only fills `pollers` and `taskQueueStatus` fields
	if len(response.GetVersionsInfo()) == 0 &&
		(len(response.GetPollers()) > 0 || response.GetTaskQueueStatus() != nil) {
		return errors.New("server does not support `DescribeTaskQueueEnhanced`")
	}
	return nil
}

func taskQueueDescriptionFromResponse(response *workflowservice.DescribeTaskQueueResponse) TaskQueueDescription {
	if response == nil {
		return TaskQueueDescription{}
	}

	versionsInfo := make(map[string]TaskQueueVersionInfo, len(response.GetVersionsInfo()))
	for buildID, vInfo := range response.GetVersionsInfo() {
		versionsInfo[buildID] = taskQueueVersionInfoFromResponse(vInfo)
	}

	return TaskQueueDescription{
		VersionsInfo: versionsInfo,
	}
}

func taskQueueVersionSelectionToProto(s *TaskQueueVersionSelection) *taskqueuepb.TaskQueueVersionSelection {
	if s == nil {
		return nil
	}

	return &taskqueuepb.TaskQueueVersionSelection{
		BuildIds:    s.BuildIDs,
		Unversioned: s.Unversioned,
		AllActive:   s.AllActive,
	}
}

func taskQueueTypeToProto(t TaskQueueType) enumspb.TaskQueueType {
	switch t {
	case TaskQueueTypeUnspecified:
		return enumspb.TASK_QUEUE_TYPE_UNSPECIFIED
	case TaskQueueTypeWorkflow:
		return enumspb.TASK_QUEUE_TYPE_WORKFLOW
	case TaskQueueTypeActivity:
		return enumspb.TASK_QUEUE_TYPE_ACTIVITY
	case TaskQueueTypeNexus:
		return enumspb.TASK_QUEUE_TYPE_NEXUS
	default:
		panic("unknown task queue type")
	}
}

func taskQueueTypeFromProto(t enumspb.TaskQueueType) TaskQueueType {
	switch t {
	case enumspb.TASK_QUEUE_TYPE_UNSPECIFIED:
		return TaskQueueTypeUnspecified
	case enumspb.TASK_QUEUE_TYPE_WORKFLOW:
		return TaskQueueTypeWorkflow
	case enumspb.TASK_QUEUE_TYPE_ACTIVITY:
		return TaskQueueTypeActivity
	case enumspb.TASK_QUEUE_TYPE_NEXUS:
		return TaskQueueTypeNexus
	default:
		panic("unknown task queue type from proto")
	}
}

func buildIDTaskReachabilityFromProto(r enumspb.BuildIdTaskReachability) BuildIDTaskReachability {
	switch r {
	case enumspb.BUILD_ID_TASK_REACHABILITY_UNSPECIFIED:
		return BuildIDTaskReachabilityUnspecified
	case enumspb.BUILD_ID_TASK_REACHABILITY_REACHABLE:
		return BuildIDTaskReachabilityReachable
	case enumspb.BUILD_ID_TASK_REACHABILITY_CLOSED_WORKFLOWS_ONLY:
		return BuildIDTaskReachabilityClosedWorkflowsOnly
	case enumspb.BUILD_ID_TASK_REACHABILITY_UNREACHABLE:
		return BuildIDTaskReachabilityUnreachable
	default:
		panic("unknown task queue reachability")
	}
}
