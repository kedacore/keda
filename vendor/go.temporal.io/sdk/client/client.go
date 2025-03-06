// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
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

//go:generate mockgen -copyright_file ../LICENSE -package client -source client.go -destination client_mock.go

// Package client is used by external programs to communicate with Temporal service.
//
// NOTE: DO NOT USE THIS API INSIDE OF ANY WORKFLOW CODE!!!
package client

import (
	"context"
	"crypto/tls"
	"io"

	"go.temporal.io/api/cloud/cloudservice/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal"
	"go.temporal.io/sdk/internal/common/metrics"
)

// TaskReachability specifies which category of tasks may reach a worker on a versioned task queue.
// Used both in a reachability query and its response.
//
// Deprecated: Use [BuildIDTaskReachability]
type TaskReachability = internal.TaskReachability

const (
	// TaskReachabilityUnspecified indicates the reachability was not specified
	TaskReachabilityUnspecified = internal.TaskReachabilityUnspecified
	// TaskReachabilityNewWorkflows indicates the Build Id might be used by new workflows
	TaskReachabilityNewWorkflows = internal.TaskReachabilityNewWorkflows
	// TaskReachabilityExistingWorkflows indicates the Build Id might be used by open workflows
	// and/or closed workflows.
	TaskReachabilityExistingWorkflows = internal.TaskReachabilityExistingWorkflows
	// TaskReachabilityOpenWorkflows indicates the Build Id might be used by open workflows.
	TaskReachabilityOpenWorkflows = internal.TaskReachabilityOpenWorkflows
	// TaskReachabilityClosedWorkflows indicates the Build Id might be used by closed workflows
	TaskReachabilityClosedWorkflows = internal.TaskReachabilityClosedWorkflows
)

// TaskQueueType specifies which category of tasks are associated with a queue.
// WARNING: Worker versioning is currently experimental
type TaskQueueType = internal.TaskQueueType

const (
	// TaskQueueTypeUnspecified indicates the task queue type was not specified.
	TaskQueueTypeUnspecified = internal.TaskQueueTypeUnspecified
	// TaskQueueTypeWorkflow indicates the task queue is used for dispatching workflow tasks.
	TaskQueueTypeWorkflow = internal.TaskQueueTypeWorkflow
	// TaskQueueTypeActivity indicates the task queue is used for delivering activity tasks.
	TaskQueueTypeActivity = internal.TaskQueueTypeActivity
	// TaskQueueTypeNexus indicates the task queue is used for dispatching Nexus requests.
	TaskQueueTypeNexus = internal.TaskQueueTypeNexus
)

// BuildIDTaskReachability specifies which category of tasks may reach a versioned worker of a certain Build ID.
// Note: future activities who inherit their workflow's Build ID but not its task queue will not be
// accounted for reachability as server cannot know if they'll happen as they do not use
// assignment rules of their task queue. Same goes for Child Workflows or Continue-As-New Workflows
// who inherit the parent/previous workflow's Build ID but not its task queue. In those cases, make
// sure to query reachability for the parent/previous workflow's task queue as well.
// WARNING: Worker versioning is currently experimental
type BuildIDTaskReachability = internal.BuildIDTaskReachability

const (
	// BuildIDTaskReachabilityUnspecified indicates that task reachability was not reported.
	BuildIDTaskReachabilityUnspecified = internal.BuildIDTaskReachabilityUnspecified
	// BuildIDTaskReachabilityReachable indicates that this Build ID may be used by new workflows or activities
	// (based on versioning rules), or there are open workflows or backlogged activities assigned to it.
	BuildIDTaskReachabilityReachable = internal.BuildIDTaskReachabilityReachable
	// BuildIDTaskReachabilityClosedWorkflowsOnly specifies that this Build ID does not have open workflows
	// and is not reachable by new workflows, but MAY have closed workflows within the namespace retention period.
	// Not applicable to activity-only task queues.
	BuildIDTaskReachabilityClosedWorkflowsOnly = internal.BuildIDTaskReachabilityClosedWorkflowsOnly
	// BuildIDTaskReachabilityUnreachable indicates that this Build ID is not used for new executions, nor
	// it has been used by any existing execution within the retention period.
	BuildIDTaskReachabilityUnreachable = internal.BuildIDTaskReachabilityUnreachable
)

// WorkflowUpdateStage indicates the stage of an update request.
// NOTE: Experimental
type WorkflowUpdateStage = internal.WorkflowUpdateStage

const (
	// WorkflowUpdateStageUnspecified indicates the wait stage was not specified
	// NOTE: Experimental
	WorkflowUpdateStageUnspecified = internal.WorkflowUpdateStageUnspecified
	// WorkflowUpdateStageAdmitted indicates the update is admitted
	// NOTE: Experimental
	WorkflowUpdateStageAdmitted = internal.WorkflowUpdateStageAdmitted
	// WorkflowUpdateStageAccepted indicates the update is accepted
	// NOTE: Experimental
	WorkflowUpdateStageAccepted = internal.WorkflowUpdateStageAccepted
	// WorkflowUpdateStageCompleted indicates the update is completed
	// NOTE: Experimental
	WorkflowUpdateStageCompleted = internal.WorkflowUpdateStageCompleted
)

const (
	// DefaultHostPort is the host:port which is used if not passed with options.
	DefaultHostPort = internal.LocalHostPort

	// DefaultNamespace is the namespace name which is used if not passed with options.
	DefaultNamespace = internal.DefaultNamespace

	// QueryTypeStackTrace is the build in query type for Client.QueryWorkflow() call. Use this query type to get the call
	// stack of the workflow. The result will be a string encoded in the converter.EncodedValue.
	QueryTypeStackTrace string = internal.QueryTypeStackTrace

	// QueryTypeOpenSessions is the build in query type for Client.QueryWorkflow() call. Use this query type to get all open
	// sessions in the workflow. The result will be a list of SessionInfo encoded in the converter.EncodedValue.
	QueryTypeOpenSessions string = internal.QueryTypeOpenSessions

	// UnversionedBuildID is a stand-in for a Build Id for unversioned Workers.
	// WARNING: Worker versioning is currently experimental
	UnversionedBuildID string = internal.UnversionedBuildID
)

type (
	// Options are optional parameters for Client creation.
	Options = internal.ClientOptions

	// CloudOperationsClientOptions are parameters for CloudOperationsClient creation.
	//
	// WARNING: Cloud operations client is currently experimental.
	CloudOperationsClientOptions = internal.CloudOperationsClientOptions

	// ConnectionOptions are optional parameters that can be specified in ClientOptions
	ConnectionOptions = internal.ConnectionOptions

	// Credentials are optional credentials that can be specified in ClientOptions.
	Credentials = internal.Credentials

	// StartWorkflowOptions configuration parameters for starting a workflow execution.
	StartWorkflowOptions = internal.StartWorkflowOptions

	// WithStartWorkflowOperation defines how to start a workflow when using UpdateWithStartWorkflow.
	// See [Client.NewWithStartWorkflowOperation] and [Client.UpdateWithStartWorkflow].
	// NOTE: Experimental
	WithStartWorkflowOperation = internal.WithStartWorkflowOperation

	// HistoryEventIterator is a iterator which can return history events.
	HistoryEventIterator = internal.HistoryEventIterator

	// WorkflowRun represents a started non child workflow.
	WorkflowRun = internal.WorkflowRun

	// WorkflowRunGetOptions are options for WorkflowRun.GetWithOptions.
	WorkflowRunGetOptions = internal.WorkflowRunGetOptions

	// QueryWorkflowWithOptionsRequest defines the request to QueryWorkflowWithOptions.
	QueryWorkflowWithOptionsRequest = internal.QueryWorkflowWithOptionsRequest

	// QueryWorkflowWithOptionsResponse defines the response to QueryWorkflowWithOptions.
	QueryWorkflowWithOptionsResponse = internal.QueryWorkflowWithOptionsResponse

	// CheckHealthRequest is a request for Client.CheckHealth.
	CheckHealthRequest = internal.CheckHealthRequest

	// CheckHealthResponse is a response for Client.CheckHealth.
	CheckHealthResponse = internal.CheckHealthResponse

	// ScheduleRange represents a set of integer values.
	ScheduleRange = internal.ScheduleRange

	// ScheduleCalendarSpec is an event specification relative to the calendar.
	ScheduleCalendarSpec = internal.ScheduleCalendarSpec

	// ScheduleIntervalSpec describes periods a schedules action should occur.
	ScheduleIntervalSpec = internal.ScheduleIntervalSpec

	// ScheduleSpec describes when a schedules action should occur.
	ScheduleSpec = internal.ScheduleSpec

	// SchedulePolicies describes the current polcies of a schedule.
	SchedulePolicies = internal.SchedulePolicies

	// ScheduleState describes the current state of a schedule.
	ScheduleState = internal.ScheduleState

	// ScheduleBackfill desribes a time periods and policy and takes Actions as if that time passed by right now, all at once.
	ScheduleBackfill = internal.ScheduleBackfill

	// ScheduleAction is the interface for all actions a schedule can take.
	ScheduleAction = internal.ScheduleAction

	// ScheduleWorkflowAction is the implementation of ScheduleAction to start a workflow.
	ScheduleWorkflowAction = internal.ScheduleWorkflowAction

	// ScheduleOptions configuration parameters for creating a schedule.
	ScheduleOptions = internal.ScheduleOptions

	// ScheduleClient is the interface with the server to create and get handles to schedules.
	ScheduleClient = internal.ScheduleClient

	// ScheduleListOptions are configuration parameters for listing schedules.
	ScheduleListOptions = internal.ScheduleListOptions

	// ScheduleListIterator is a iterator which can return created schedules.
	ScheduleListIterator = internal.ScheduleListIterator

	// ScheduleListEntry is a result from ScheduleListEntry.
	ScheduleListEntry = internal.ScheduleListEntry

	// ScheduleUpdateOptions are configuration parameters for updating a schedule.
	ScheduleUpdateOptions = internal.ScheduleUpdateOptions

	// ScheduleHandle represents a created schedule.
	ScheduleHandle = internal.ScheduleHandle

	// ScheduleActionResult describes when a schedule action took place.
	ScheduleActionResult = internal.ScheduleActionResult

	// ScheduleWorkflowExecution contains details on a workflows execution stared by a schedule.
	ScheduleWorkflowExecution = internal.ScheduleWorkflowExecution

	// ScheduleInfo describes other information about a schedule.
	ScheduleInfo = internal.ScheduleInfo

	// ScheduleDescription describes the current Schedule details from ScheduleHandle.Describe.
	ScheduleDescription = internal.ScheduleDescription

	// Schedule describes a created schedule.
	Schedule = internal.Schedule

	// ScheduleUpdate describes the desired new schedule from ScheduleHandle.Update.
	ScheduleUpdate = internal.ScheduleUpdate

	// ScheduleUpdateInput describes the current state of the schedule to be updated.
	ScheduleUpdateInput = internal.ScheduleUpdateInput

	// ScheduleTriggerOptions configure the parameters for triggering a schedule.
	ScheduleTriggerOptions = internal.ScheduleTriggerOptions

	// SchedulePauseOptions configure the parameters for pausing a schedule.
	SchedulePauseOptions = internal.SchedulePauseOptions

	// ScheduleUnpauseOptions configure the parameters for unpausing a schedule.
	ScheduleUnpauseOptions = internal.ScheduleUnpauseOptions

	// ScheduleBackfillOptions configure the parameters for backfilling a schedule.
	ScheduleBackfillOptions = internal.ScheduleBackfillOptions

	// UpdateWorkflowOptions encapsulates the parameters for
	// sending an update to a workflow execution.
	// NOTE: Experimental
	UpdateWorkflowOptions = internal.UpdateWorkflowOptions

	// UpdateWithStartWorkflowOptions encapsulates the parameters used by UpdateWithStartWorkflow.
	// See [Client.UpdateWithStartWorkflow] and [Client.NewWithStartWorkflowOperation].
	// NOTE: Experimental
	UpdateWithStartWorkflowOptions = internal.UpdateWithStartWorkflowOptions

	// WorkflowUpdateHandle represents a running or completed workflow
	// execution update and gives the holder access to the outcome of the same.
	// NOTE: Experimental
	WorkflowUpdateHandle = internal.WorkflowUpdateHandle

	// GetWorkflowUpdateHandleOptions encapsulates the parameters needed to unambiguously
	// refer to a Workflow Update
	// NOTE: Experimental
	GetWorkflowUpdateHandleOptions = internal.GetWorkflowUpdateHandleOptions

	// UpdateWorkerBuildIdCompatibilityOptions is the input to Client.UpdateWorkerBuildIdCompatibility.
	//
	// Deprecated: Use [UpdateWorkerVersioningRulesOptions] with the new worker versioning api.
	UpdateWorkerBuildIdCompatibilityOptions = internal.UpdateWorkerBuildIdCompatibilityOptions

	// GetWorkerBuildIdCompatibilityOptions is the input to Client.GetWorkerBuildIdCompatibility.
	//
	// Deprecated: Use [GetWorkerVersioningOptions] with the new worker versioning api.
	GetWorkerBuildIdCompatibilityOptions = internal.GetWorkerBuildIdCompatibilityOptions

	// WorkerBuildIDVersionSets is the response for Client.GetWorkerBuildIdCompatibility.
	//
	// Deprecated: Replaced by the new worker versioning api.
	WorkerBuildIDVersionSets = internal.WorkerBuildIDVersionSets

	// BuildIDOpAddNewIDInNewDefaultSet is an operation for UpdateWorkerBuildIdCompatibilityOptions
	// to add a new BuildID in a new default set.
	//
	// Deprecated: Replaced by the new worker versioning api.
	BuildIDOpAddNewIDInNewDefaultSet = internal.BuildIDOpAddNewIDInNewDefaultSet

	// BuildIDOpAddNewCompatibleVersion is an operation for UpdateWorkerBuildIdCompatibilityOptions
	// to add a new BuildID to an existing compatible set.
	//
	// Deprecated: Replaced by the new worker versioning api.
	BuildIDOpAddNewCompatibleVersion = internal.BuildIDOpAddNewCompatibleVersion

	// BuildIDOpPromoteSet is an operation for UpdateWorkerBuildIdCompatibilityOptions to promote a
	// set to be the default set by targeting an existing BuildID.
	//
	// Deprecated: Replaced by the new worker versioning api.
	BuildIDOpPromoteSet = internal.BuildIDOpPromoteSet

	// BuildIDOpPromoteIDWithinSet is an operation for UpdateWorkerBuildIdCompatibilityOptions to
	// promote a BuildID within a set to be the default.
	//
	// Deprecated: Replaced by the new worker versioning api.
	BuildIDOpPromoteIDWithinSet = internal.BuildIDOpPromoteIDWithinSet

	// GetWorkerTaskReachabilityOptions is the input to Client.GetWorkerTaskReachability.
	//
	// Deprecated: Use [DescribeTaskQueueEnhancedOptions] with the new worker versioning api.
	GetWorkerTaskReachabilityOptions = internal.GetWorkerTaskReachabilityOptions

	// WorkerTaskReachability is the response for Client.GetWorkerTaskReachability.
	//
	// Deprecated: Replaced by the new worker versioning api.
	WorkerTaskReachability = internal.WorkerTaskReachability

	// BuildIDReachability describes the reachability of a buildID
	//
	// Deprecated: Replaced by the new worker versioning api.
	BuildIDReachability = internal.BuildIDReachability

	// TaskQueueReachability Describes how the Build ID may be reachable from the task queue.
	//
	// Deprecated: Replaced by the new worker versioning api.
	TaskQueueReachability = internal.TaskQueueReachability

	// DescribeTaskQueueEnhancedOptions is the input to [Client.DescribeTaskQueueEnhanced].
	DescribeTaskQueueEnhancedOptions = internal.DescribeTaskQueueEnhancedOptions

	// TaskQueueVersionSelection is a task queue filter based on versioning.
	// It is an optional component of [Client.DescribeTaskQueueEnhancedOptions].
	// WARNING: Worker versioning is currently experimental.
	TaskQueueVersionSelection = internal.TaskQueueVersionSelection

	// TaskQueueDescription is the response to [Client.DescribeTaskQueueEnhanced].
	TaskQueueDescription = internal.TaskQueueDescription

	// TaskQueueVersionInfo includes task queue information per Build ID.
	// It is part of [Client.TaskQueueDescription].
	TaskQueueVersionInfo = internal.TaskQueueVersionInfo

	// TaskQueueTypeInfo specifies task queue information per task type and Build ID.
	// It is included in [Client.TaskQueueVersionInfo].
	TaskQueueTypeInfo = internal.TaskQueueTypeInfo

	// TaskQueuePollerInfo provides information about a worker/client polling a task queue.
	// It is used by [Client.TaskQueueTypeInfo].
	TaskQueuePollerInfo = internal.TaskQueuePollerInfo

	// TaskQueueStats contains statistics about task queue backlog and activity.
	//
	// For workflow task queue type, this result is partial because tasks sent to sticky queues are not included. Read
	// comments above each metric to understand the impact of sticky queue exclusion on that metric accuracy.
	TaskQueueStats = internal.TaskQueueStats

	// WorkerVersionCapabilities includes a worker's build identifier
	// and whether it is choosing to use the versioning feature.
	// It is an optional component of [Client.TaskQueuePollerInfo].
	// WARNING: Worker versioning is currently experimental.
	WorkerVersionCapabilities = internal.WorkerVersionCapabilities

	// UpdateWorkerVersioningRulesOptions is the input to [Client.UpdateWorkerVersioningRules].
	// WARNING: Worker versioning is currently experimental.
	UpdateWorkerVersioningRulesOptions = internal.UpdateWorkerVersioningRulesOptions

	// VersioningConflictToken is a conflict token to serialize calls to Client.UpdateWorkerVersioningRules.
	// An update with an old token fails with `serviceerror.FailedPrecondition`.
	// The current token can be obtained with [GetWorkerVersioningRules],
	// or returned by a successful [UpdateWorkerVersioningRules].
	// WARNING: Worker versioning is currently experimental.
	VersioningConflictToken = internal.VersioningConflictToken

	// VersioningRampByPercentage is a VersionRamp that sends a proportion of the traffic
	// to the target Build ID.
	// WARNING: Worker versioning is currently experimental.
	VersioningRampByPercentage = internal.VersioningRampByPercentage

	// VersioningAssignmentRule is a BuildID  assigment rule for a task queue.
	// Assignment rules only affect new workflows.
	// WARNING: Worker versioning is currently experimental.
	VersioningAssignmentRule = internal.VersioningAssignmentRule

	// VersioningAssignmentRuleWithTimestamp contains an assignment rule annotated
	// by the server with its creation time.
	// WARNING: Worker versioning is currently experimental.
	VersioningAssignmentRuleWithTimestamp = internal.VersioningAssignmentRuleWithTimestamp

	// VersioningAssignmentRule is a BuildID redirect rule for a task queue.
	// It changes the behavior of currently running workflows and new ones.
	// WARNING: Worker versioning is currently experimental.
	VersioningRedirectRule = internal.VersioningRedirectRule

	// VersioningRedirectRuleWithTimestamp contains a redirect rule annotated
	// by the server with its creation time.
	// WARNING: Worker versioning is currently experimental.
	VersioningRedirectRuleWithTimestamp = internal.VersioningRedirectRuleWithTimestamp

	// VersioningOperationInsertAssignmentRule is an operation for UpdateWorkerVersioningRulesOptions
	// that inserts the rule to the list of assignment rules for this Task Queue.
	// The rules are evaluated in order, starting from index 0. The first
	// applicable rule will be applied and the rest will be ignored.
	// By default, the new rule is inserted at the beginning of the list
	// (index 0). If the given index is too larger the rule will be
	// inserted at the end of the list.
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationInsertAssignmentRule = internal.VersioningOperationInsertAssignmentRule

	// VersioningOperationReplaceAssignmentRule is an operation for UpdateWorkerVersioningRulesOptions
	// that replaces the assignment rule at a given index. By default presence of one
	// unconditional rule, i.e., no hint filter or ramp, is enforced, otherwise
	// the delete operation will be rejected. Set `force` to true to
	// bypass this validation.
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationReplaceAssignmentRule = internal.VersioningOperationReplaceAssignmentRule

	// VersioningOperationDeleteAssignmentRule is an operation for UpdateWorkerVersioningRulesOptions
	// that deletes the assignment rule at a given index. By default presence of one
	// unconditional rule, i.e., no hint filter or ramp, is enforced, otherwise
	// the delete operation will be rejected. Set `force` to true to
	// bypass this validation.
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationDeleteAssignmentRule = internal.VersioningOperationDeleteAssignmentRule

	// VersioningOperationAddRedirectRule is an operation for UpdateWorkerVersioningRulesOptions
	// that adds the rule to the list of redirect rules for this Task Queue. There
	// can be at most one redirect rule for each distinct Source BuildID.
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationAddRedirectRule = internal.VersioningOperationAddRedirectRule

	// VersioningOperationReplaceRedirectRule is an operation for UpdateWorkerVersioningRulesOptions
	// that replaces the routing rule with the given source BuildID.
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationReplaceRedirectRule = internal.VersioningOperationReplaceRedirectRule

	// VersioningOperationDeleteRedirectRule is an operation for UpdateWorkerVersioningRulesOptions
	// that deletes the routing rule with the given source Build ID.
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationDeleteRedirectRule = internal.VersioningOperationDeleteRedirectRule

	// VersioningOperationCommitBuildID is an operation for UpdateWorkerVersioningRulesOptions
	// that completes  the rollout of a BuildID and cleanup unnecessary rules possibly
	// created during a gradual rollout. Specifically, this command will make the following changes
	// atomically:
	//  1. Adds an assignment rule (with full ramp) for the target Build ID at
	//     the end of the list.
	//  2. Removes all previously added assignment rules to the given target
	//     Build ID (if any).
	//  3. Removes any fully-ramped assignment rule for other Build IDs.
	//
	// To prevent committing invalid Build IDs, we reject the request if no
	// pollers have been seen recently for this Build ID. Use the `force`
	// option to disable this validation.
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationCommitBuildID = internal.VersioningOperationCommitBuildID

	// GetWorkerVersioningOptions is the input to [Client.GetWorkerVersioningRules].
	// WARNING: Worker versioning is currently experimental.
	GetWorkerVersioningOptions = internal.GetWorkerVersioningOptions

	// WorkerVersioningRules is the response for [Client.GetWorkerVersioningRules].
	// WARNING: Worker versioning is currently experimental.
	WorkerVersioningRules = internal.WorkerVersioningRules

	// WorkflowUpdateServiceTimeoutOrCanceledError is an error that occurs when an update call times out or is cancelled.
	//
	// Note, this is not related to any general concept of timing out or cancelling a running update, this is only related to the client call itself.
	// NOTE: Experimental
	WorkflowUpdateServiceTimeoutOrCanceledError = internal.WorkflowUpdateServiceTimeoutOrCanceledError

	// Client is the client for starting and getting information about a workflow executions as well as
	// completing activities asynchronously.
	Client interface {
		// ExecuteWorkflow starts a workflow execution and returns a WorkflowRun instance or error
		//
		// This can be used to start a workflow using a function reference or workflow type name.
		// Either by
		//     ExecuteWorkflow(ctx, options, "workflowTypeName", arg1, arg2, arg3)
		//     or
		//     ExecuteWorkflow(ctx, options, workflowExecuteFn, arg1, arg2, arg3)
		// The errors it can return:
		//  - serviceerror.NamespaceNotFound, if namespace does not exist
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//
		// WorkflowRun has 3 methods:
		//  - GetWorkflowID() string: which return the started workflow ID
		//  - GetRunID() string: which return the first started workflow run ID (please see below)
		//  - Get(ctx context.Context, valuePtr interface{}) error: which will fill the workflow
		//    execution result to valuePtr, if workflow execution is a success, or return corresponding
		//    error. This is a blocking API.
		// NOTE: if the started workflow returns ContinueAsNewError during the workflow execution, the
		// returned result of GetRunID() will be the started workflow run ID, not the new run ID caused by ContinueAsNewError,
		// however, Get(ctx context.Context, valuePtr interface{}) will return result from the run which did not return ContinueAsNewError.
		// Say ExecuteWorkflow started a workflow, in its first run, has run ID "run ID 1", and returned ContinueAsNewError,
		// the second run has run ID "run ID 2" and return some result other than ContinueAsNewError:
		// GetRunID() will always return "run ID 1" and  Get(ctx context.Context, valuePtr interface{}) will return the result of second run.
		// NOTE: DO NOT USE THIS API INSIDE A WORKFLOW, USE workflow.ExecuteChildWorkflow instead
		ExecuteWorkflow(ctx context.Context, options StartWorkflowOptions, workflow interface{}, args ...interface{}) (WorkflowRun, error)

		// GetWorkflow retrieves a workflow execution and return a WorkflowRun instance (described above)
		// - workflow ID of the workflow.
		// - runID can be default(empty string). if empty string then it will pick the last running execution of that workflow ID.
		//
		// WorkflowRun has 2 methods:
		//  - GetRunID() string: which return the first started workflow run ID (please see below)
		//  - Get(ctx context.Context, valuePtr interface{}) error: which will fill the workflow
		//    execution result to valuePtr, if workflow execution is a success, or return corresponding
		//    error. This is a blocking API.
		// If workflow not found, the Get() will return serviceerror.NotFound.
		// NOTE: if the started workflow return ContinueAsNewError during the workflow execution, the
		// return result of GetRunID() will be the started workflow run ID, not the new run ID caused by ContinueAsNewError,
		// however, Get(ctx context.Context, valuePtr interface{}) will return result from the run which did not return ContinueAsNewError.
		// Say ExecuteWorkflow started a workflow, in its first run, has run ID "run ID 1", and returned ContinueAsNewError,
		// the second run has run ID "run ID 2" and return some result other than ContinueAsNewError:
		// GetRunID() will always return "run ID 1" and  Get(ctx context.Context, valuePtr interface{}) will return the result of second run.
		GetWorkflow(ctx context.Context, workflowID string, runID string) WorkflowRun

		// SignalWorkflow sends a signals to a workflow in execution
		// - workflow ID of the workflow.
		// - runID can be default(empty string). if empty string then it will pick the running execution of that workflow ID.
		// - signalName name to identify the signal.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error

		// SignalWithStartWorkflow sends a signal to a running workflow.
		// If the workflow is not running or not found, it starts the workflow and then sends the signal in transaction.
		// - workflowID, signalName, signalArg are same as SignalWorkflow's parameters
		// - options, workflow, workflowArgs are same as StartWorkflow's parameters
		// - the workflowID parameter is used instead of options.ID. If the latter is present, it must match the workflowID.
		// Note: options.WorkflowIDReusePolicy is default to AllowDuplicate in this API.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		SignalWithStartWorkflow(ctx context.Context, workflowID string, signalName string, signalArg interface{},
			options StartWorkflowOptions, workflow interface{}, workflowArgs ...interface{}) (WorkflowRun, error)

		// NewWithStartWorkflowOperation returns a WithStartWorkflowOperation for use with UpdateWithStartWorkflow.
		// See [Client.UpdateWithStartWorkflow].
		// NOTE: Experimental
		NewWithStartWorkflowOperation(options StartWorkflowOptions, workflow interface{}, args ...interface{}) WithStartWorkflowOperation

		// CancelWorkflow request cancellation of a workflow in execution. Cancellation request closes the channel
		// returned by the workflow.Context.Done() of the workflow that is target of the request.
		// - workflow ID of the workflow.
		// - runID can be default(empty string). if empty string then it will pick the currently running execution of that workflow ID.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		CancelWorkflow(ctx context.Context, workflowID string, runID string) error

		// TerminateWorkflow terminates a workflow execution. Terminate stops a workflow execution immediately without
		// letting the workflow to perform any cleanup
		// workflowID is required, other parameters are optional.
		// - workflow ID of the workflow.
		// - runID can be default(empty string). if empty string then it will pick the running execution of that workflow ID.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		TerminateWorkflow(ctx context.Context, workflowID string, runID string, reason string, details ...interface{}) error

		// GetWorkflowHistory gets history events of a particular workflow
		// - workflow ID of the workflow.
		// - runID can be default(empty string). if empty string then it will pick the last running execution of that workflow ID.
		// - whether use long poll for tracking new events: when the workflow is running, there can be new events generated during iteration
		//    of HistoryEventIterator, if isLongPoll == true, then iterator will do long poll, tracking new history event, i.e. the iteration
		//   will not be finished until workflow is finished; if isLongPoll == false, then iterator will only return current history events.
		// - whether return all history events or just the last event, which contains the workflow execution end result
		// Example:-
		//  To iterate all events,
		//     iter := GetWorkflowHistory(ctx, workflowID, runID, isLongPoll, filterType)
		//    events := []*shared.HistoryEvent{}
		//    for iter.HasNext() {
		//      event, err := iter.Next()
		//      if err != nil {
		//        return err
		//      }
		//      events = append(events, event)
		//    }
		GetWorkflowHistory(ctx context.Context, workflowID string, runID string, isLongPoll bool, filterType enumspb.HistoryEventFilterType) HistoryEventIterator

		// CompleteActivity reports activity completed.
		// activity Execute method can return activity.ErrResultPending to
		// indicate the activity is not completed when it's Execute method returns. In that case, this CompleteActivity() method
		// should be called when that activity is completed with the actual result and error. If err is nil, activity task
		// completed event will be reported; if err is CanceledError, activity task canceled event will be reported; otherwise,
		// activity task failed event will be reported.
		// An activity implementation should use GetActivityInfo(ctx).TaskToken function to get task token to use for completion.
		// Example:-
		//  To complete with a result.
		//    CompleteActivity(token, "Done", nil)
		//  To fail the activity with an error.
		//      CompleteActivity(token, nil, temporal.NewApplicationError("reason", details)
		// The activity can fail with below errors ApplicationError, TimeoutError, CanceledError.
		CompleteActivity(ctx context.Context, taskToken []byte, result interface{}, err error) error

		// CompleteActivityByID reports activity completed.
		// Similar to CompleteActivity, but may save user from keeping taskToken info.
		// activity Execute method can return activity.ErrResultPending to
		// indicate the activity is not completed when it's Execute method returns. In that case, this CompleteActivityById() method
		// should be called when that activity is completed with the actual result and error. If err is nil, activity task
		// completed event will be reported; if err is CanceledError, activity task canceled event will be reported; otherwise,
		// activity task failed event will be reported.
		// An activity implementation should use activityID provided in ActivityOption to use for completion.
		// namespace name, workflowID, activityID are required, runID is optional.
		// The errors it can return:
		//  - ApplicationError
		//  - TimeoutError
		//  - CanceledError
		CompleteActivityByID(ctx context.Context, namespace, workflowID, runID, activityID string, result interface{}, err error) error

		// RecordActivityHeartbeat records heartbeat for an activity.
		// taskToken - is the value of the binary "TaskToken" field of the "ActivityInfo" struct retrieved inside the activity.
		// details - is the progress you want to record along with heart beat for this activity.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		RecordActivityHeartbeat(ctx context.Context, taskToken []byte, details ...interface{}) error

		// RecordActivityHeartbeatByID records heartbeat for an activity.
		// details - is the progress you want to record along with heart beat for this activity.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		RecordActivityHeartbeatByID(ctx context.Context, namespace, workflowID, runID, activityID string, details ...interface{}) error

		// ListClosedWorkflow gets closed workflow executions based on request filters.
		// Retrieved workflow executions are sorted by close time in descending order.
		// Note: heavy usage of this API may cause huge persistence pressure.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NamespaceNotFound
		ListClosedWorkflow(ctx context.Context, request *workflowservice.ListClosedWorkflowExecutionsRequest) (*workflowservice.ListClosedWorkflowExecutionsResponse, error)

		// ListOpenWorkflow gets open workflow executions based on request filters.
		// Retrieved workflow executions are sorted by start time in descending order.
		// Note: heavy usage of this API may cause huge persistence pressure.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NamespaceNotFound
		ListOpenWorkflow(ctx context.Context, request *workflowservice.ListOpenWorkflowExecutionsRequest) (*workflowservice.ListOpenWorkflowExecutionsResponse, error)

		// ListWorkflow gets workflow executions based on query. The query is basically the SQL WHERE clause, examples:
		//  - "(WorkflowID = 'wid1' or (WorkflowType = 'type2' and WorkflowID = 'wid2'))".
		//  - "CloseTime between '2019-08-27T15:04:05+00:00' and '2019-08-28T15:04:05+00:00'".
		//  - to list only open workflow use "CloseTime is null"
		// For supported operations on different server versions see https://docs.temporal.io/visibility.
		// Retrieved workflow executions are sorted by StartTime in descending order when list open workflow,
		// and sorted by CloseTime in descending order for other queries.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		ListWorkflow(ctx context.Context, request *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error)

		// ListArchivedWorkflow gets archived workflow executions based on query. This API will return BadRequest if Temporal
		// cluster or target namespace is not configured for visibility archival or read is not enabled. The query is basically the SQL WHERE clause.
		// However, different visibility archivers have different limitations on the query. Please check the documentation of the visibility archiver used
		// by your namespace to see what kind of queries are accept and whether retrieved workflow executions are ordered or not.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		ListArchivedWorkflow(ctx context.Context, request *workflowservice.ListArchivedWorkflowExecutionsRequest) (*workflowservice.ListArchivedWorkflowExecutionsResponse, error)

		// ScanWorkflow gets workflow executions based on query. The query is basically the SQL WHERE clause
		// (see ListWorkflow for query examples).
		// For supported operations on different server versions see https://docs.temporal.io/visibility.
		// ScanWorkflow should be used when retrieving large amount of workflows and order is not needed.
		// It will use more resources than ListWorkflow, but will be several times faster
		// when retrieving millions of workflows.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		ScanWorkflow(ctx context.Context, request *workflowservice.ScanWorkflowExecutionsRequest) (*workflowservice.ScanWorkflowExecutionsResponse, error)

		// CountWorkflow gets number of workflow executions based on query. The query is basically the SQL WHERE clause
		// (see ListWorkflow for query examples).
		// For supported operations on different server versions see https://docs.temporal.io/visibility.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		CountWorkflow(ctx context.Context, request *workflowservice.CountWorkflowExecutionsRequest) (*workflowservice.CountWorkflowExecutionsResponse, error)

		// GetSearchAttributes returns valid search attributes keys and value types.
		// The search attributes can be used in query of List/Scan/Count APIs. Adding new search attributes requires temporal server
		// to update dynamic config ValidSearchAttributes.
		// NOTE: This API is not supported on Temporal Cloud.
		GetSearchAttributes(ctx context.Context) (*workflowservice.GetSearchAttributesResponse, error)

		// QueryWorkflow queries a given workflow's last execution and returns the query result synchronously. Parameter workflowID
		// and queryType are required, other parameters are optional. The workflowID and runID (optional) identify the
		// target workflow execution that this query will be send to. If runID is not specified (empty string), server will
		// use the currently running execution of that workflowID. The queryType specifies the type of query you want to
		// run. By default, temporal supports "__stack_trace" as a standard query type, which will return string value
		// representing the call stack of the target workflow. The target workflow could also setup different query handler
		// to handle custom query types.
		// See comments at workflow.SetQueryHandler(ctx Context, queryType string, handler interface{}) for more details
		// on how to setup query handler within the target workflow.
		// - workflowID is required.
		// - runID can be default(empty string). if empty string then it will pick the running execution of that workflow ID.
		// - queryType is the type of the query.
		// - args... are the optional query parameters.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NotFound
		//  - serviceerror.QueryFailed
		QueryWorkflow(ctx context.Context, workflowID string, runID string, queryType string, args ...interface{}) (converter.EncodedValue, error)

		// QueryWorkflowWithOptions queries a given workflow execution and returns the query result synchronously.
		// See QueryWorkflowWithOptionsRequest and QueryWorkflowWithOptionsResponse for more information.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NotFound
		//  - serviceerror.QueryFailed
		QueryWorkflowWithOptions(ctx context.Context, request *QueryWorkflowWithOptionsRequest) (*QueryWorkflowWithOptionsResponse, error)

		// DescribeWorkflowExecution returns information about the specified workflow execution.
		// - runID can be default(empty string). if empty string then it will pick the last running execution of that workflow ID.
		//
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NotFound
		DescribeWorkflowExecution(ctx context.Context, workflowID, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error)

		// DescribeTaskQueue returns information about the target taskqueue, right now this API returns the
		// pollers which polled this taskqueue in last few minutes.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NotFound
		DescribeTaskQueue(ctx context.Context, taskqueue string, taskqueueType enumspb.TaskQueueType) (*workflowservice.DescribeTaskQueueResponse, error)

		// DescribeTaskQueueEnhanced  returns information about the target task queue, broken down by Build Id:
		//   - List of pollers
		//   - Workflow Reachability status
		//   - Backlog info for Workflow and/or Activity tasks
		// When not supported by the server, it returns an empty [TaskQueueDescription] if there is no information
		// about the task queue, or an error when the response identifies an unsupported server.
		// Note that using a sticky queue as target is not supported.
		// Also, workflow reachability status is eventually consistent, and it could take a few minutes to update.
		// WARNING: Worker versioning is currently experimental, and requires server 1.24+
		DescribeTaskQueueEnhanced(ctx context.Context, options DescribeTaskQueueEnhancedOptions) (TaskQueueDescription, error)

		// ResetWorkflowExecution resets an existing workflow execution to WorkflowTaskFinishEventId(exclusive).
		// And it will immediately terminating the current execution instance.
		// RequestId is used to deduplicate requests. It will be autogenerated if not set.
		ResetWorkflowExecution(ctx context.Context, request *workflowservice.ResetWorkflowExecutionRequest) (*workflowservice.ResetWorkflowExecutionResponse, error)

		// UpdateWorkerBuildIdCompatibility
		// Allows you to update the worker-build-id based version sets for a particular task queue. This is used in
		// conjunction with workers who specify their build id and thus opt into the feature.
		//
		// Deprecated: Use [UpdateWorkerVersioningRules] with the versioning api.
		UpdateWorkerBuildIdCompatibility(ctx context.Context, options *UpdateWorkerBuildIdCompatibilityOptions) error

		// GetWorkerBuildIdCompatibility
		// Returns the worker-build-id based version sets for a particular task queue.
		//
		// Deprecated: Use [GetWorkerVersioningRules] with the versioning api.
		GetWorkerBuildIdCompatibility(ctx context.Context, options *GetWorkerBuildIdCompatibilityOptions) (*WorkerBuildIDVersionSets, error)

		// GetWorkerTaskReachability
		// Returns which versions are is still in use by open or closed workflows
		//
		// Deprecated: Use [DescribeTaskQueueEnhanced] with the versioning api.
		GetWorkerTaskReachability(ctx context.Context, options *GetWorkerTaskReachabilityOptions) (*WorkerTaskReachability, error)

		// UpdateWorkerVersioningRules
		// Allows updating the worker-build-id based assignment and redirect rules for a given task queue. This is used in
		// conjunction with workers who specify their build id and thus opt into the feature.
		// The errors it can return:
		//  - serviceerror.FailedPrecondition when the conflict token is invalid
		// WARNING: Worker versioning is currently experimental, and requires server 1.24+
		UpdateWorkerVersioningRules(ctx context.Context, options UpdateWorkerVersioningRulesOptions) (*WorkerVersioningRules, error)

		// GetWorkerVersioningRules
		// Returns the worker-build-id assignment and redirect rules for a task queue.
		// WARNING: Worker versioning is currently experimental, and requires server 1.24+
		GetWorkerVersioningRules(ctx context.Context, options GetWorkerVersioningOptions) (*WorkerVersioningRules, error)

		// CheckHealth performs a server health check using the gRPC health check
		// API. If the check fails, an error is returned.
		CheckHealth(ctx context.Context, request *CheckHealthRequest) (*CheckHealthResponse, error)

		// UpdateWorkflow issues an update request to the specified workflow and
		// returns a handle to the update. The call will block until the update
		// has reached the WaitForStage in the options. Note that this means
		// that the call will not return successfully until the update has been
		// delivered to a worker. Errors returned from the update handler or its
		// validator will be exposed through the return value of
		// WorkflowUpdateHandle.Get(). Errors that occur before the update is
		// delivered to the workflow (e.g. if the required workflow ID field is
		// missing from the UpdateWorkflowOptions) are returned directly from
		// this function call.
		//
		// The errors it can return:
		//  - WorkflowUpdateServiceTimeoutOrCanceledError
		// NOTE: Experimental
		UpdateWorkflow(ctx context.Context, options UpdateWorkflowOptions) (WorkflowUpdateHandle, error)

		// UpdateWithStartWorkflow issues an update-with-start request. A
		// WorkflowIDConflictPolicy must be set in the options. If the specified
		// workflow execution is not running, then a new workflow execution is
		// started and the update is sent in the first workflow task.
		// Alternatively if the specified workflow execution is running then, if
		// the WorkflowIDConflictPolicy is USE_EXISTING, the update is issued
		// against the specified workflow, and if the WorkflowIDConflictPolicy
		// is FAIL, an error is returned. The call will block until the update
		// has reached the WaitForStage in the options. Note that this means
		// that the call will not return successfully until the update has been
		// delivered to a worker.
		// NOTE: Experimental
		UpdateWithStartWorkflow(ctx context.Context, options UpdateWithStartWorkflowOptions) (WorkflowUpdateHandle, error)

		// GetWorkflowUpdateHandle creates a handle to the referenced update
		// which can be polled for an outcome. Note that runID is optional and
		// if not specified the most recent runID will be used.
		// NOTE: Experimental
		GetWorkflowUpdateHandle(ref GetWorkflowUpdateHandleOptions) WorkflowUpdateHandle

		// WorkflowService provides access to the underlying gRPC service. This should only be used for advanced use cases
		// that cannot be accomplished via other Client methods. Unlike calls to other Client methods, calls directly to the
		// service are not configured with internal semantics such as automatic retries.
		WorkflowService() workflowservice.WorkflowServiceClient

		// OperatorService creates a new operator service client with the same gRPC connection as this client.
		OperatorService() operatorservice.OperatorServiceClient

		// Schedule creates a new shedule client with the same gRPC connection as this client.
		ScheduleClient() ScheduleClient

		// Close client and clean up underlying resources.
		//
		// If this client was created via NewClientFromExisting or this client has
		// been used in that call, Close() on may not necessarily close the
		// underlying connection. Only the final close of all existing clients will
		// close the underlying connection.
		Close()
	}

	// CloudOperationsClient is the client for cloud operations.
	//
	// WARNING: Cloud operations client is currently experimental.
	CloudOperationsClient interface {
		// CloudService provides access to the underlying gRPC service.
		CloudService() cloudservice.CloudServiceClient

		// Close client and clean up underlying resources.
		Close()
	}

	// NamespaceClient is the client for managing operations on the namespace.
	// CLI, tools, ... can use this layer to manager operations on namespace.
	NamespaceClient interface {
		// Register a namespace with temporal server
		// The errors it can throw:
		//  - NamespaceAlreadyExistsError
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		Register(ctx context.Context, request *workflowservice.RegisterNamespaceRequest) error

		// Describe a namespace. The namespace has 3 part of information
		// NamespaceInfo - Which has Name, Status, Description, Owner Email
		// NamespaceConfiguration - Configuration like Workflow Execution Retention Period In Days, Whether to emit metrics.
		// ReplicationConfiguration - replication config like clusters and active cluster name
		// The errors it can throw:
		//  - serviceerror.NamespaceNotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		Describe(ctx context.Context, name string) (*workflowservice.DescribeNamespaceResponse, error)

		// Update a namespace.
		// The errors it can throw:
		//  - serviceerror.NamespaceNotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		Update(ctx context.Context, request *workflowservice.UpdateNamespaceRequest) error

		// Close client and clean up underlying resources.
		Close()
	}
)

// MetricsHandler is a handler for metrics emitted by the SDK. This interface is
// intentionally limited to only what the SDK needs to emit metrics and is not
// built to be a general purpose metrics abstraction for all uses.
//
// A common implementation is at
// go.temporal.io/sdk/contrib/tally.NewMetricsHandler. The MetricsNopHandler is
// a noop handler. A handler may implement "Unwrap() client.MetricsHandler" if
// it wraps a handler.
type MetricsHandler = metrics.Handler

// MetricsCounter is an ever-increasing counter.
type MetricsCounter = metrics.Counter

// MetricsGauge can be set to any float.
type MetricsGauge = metrics.Gauge

// MetricsTimer records time durations.
type MetricsTimer = metrics.Timer

// MetricsNopHandler is a noop handler that does nothing with the metrics.
var MetricsNopHandler = metrics.NopHandler

// Dial creates an instance of a workflow client. This will attempt to connect
// to the server eagerly and will return an error if the server is not
// available.
func Dial(options Options) (Client, error) {
	return DialContext(context.Background(), options)
}

// DialContext creates an instance of a workflow client. This will attempt to connect
// to the server eagerly and will return an error if the server is not
// available. Connection will respect provided context deadlines and cancellations.
func DialContext(ctx context.Context, options Options) (Client, error) {
	return internal.DialClient(ctx, options)
}

// NewLazyClient creates an instance of a workflow client. Unlike Dial, this
// will not eagerly connect to the server.
func NewLazyClient(options Options) (Client, error) {
	return internal.NewLazyClient(options)
}

// NewClient creates an instance of a workflow client. This will attempt to
// connect to the server eagerly and will return an error if the server is not
// available.
//
// Deprecated: Use Dial or NewLazyClient instead.
func NewClient(options Options) (Client, error) {
	return internal.NewClient(context.Background(), options)
}

// NewClientFromExisting creates a new client using the same connection as the
// existing client. This means all options.ConnectionOptions are ignored and
// options.HostPort is ignored. The existing client must have been created from
// this package and cannot be wrapped. Currently, this always attempts an eager
// connection even if the existing client was created with NewLazyClient and has
// not made any calls yet.
//
// Close() on the resulting client may not necessarily close the underlying
// connection if there are any other clients using the connection. All clients
// associated with the existing client must call Close() and only the last one
// actually performs the connection close.
func NewClientFromExisting(existingClient Client, options Options) (Client, error) {
	return NewClientFromExistingWithContext(context.Background(), existingClient, options)
}

// NewClientFromExistingWithContext creates a new client using the same connection as the
// existing client. This means all options.ConnectionOptions are ignored and
// options.HostPort is ignored. The existing client must have been created from
// this package and cannot be wrapped. Currently, this always attempts an eager
// connection even if the existing client was created with NewLazyClient and has
// not made any calls yet.
//
// Close() on the resulting client may not necessarily close the underlying
// connection if there are any other clients using the connection. All clients
// associated with the existing client must call Close() and only the last one
// actually performs the connection close.
func NewClientFromExistingWithContext(ctx context.Context, existingClient Client, options Options) (Client, error) {
	return internal.NewClientFromExisting(ctx, existingClient, options)
}

// DialCloudOperationsClient creates a cloud client to perform cloud-management
// operations. Users should provide Credentials in the options.
//
// WARNING: Cloud operations client is currently experimental.
func DialCloudOperationsClient(ctx context.Context, options CloudOperationsClientOptions) (CloudOperationsClient, error) {
	return internal.DialCloudOperationsClient(ctx, options)
}

// NewNamespaceClient creates an instance of a namespace client, to manage
// lifecycle of namespaces. This will not attempt to connect to the server
// eagerly and therefore may not fail for an unreachable server until a call is
// made.
func NewNamespaceClient(options Options) (NamespaceClient, error) {
	return internal.NewNamespaceClient(options)
}

// make sure if new methods are added to internal.Client they are also added to public Client.
var (
	_ Client                         = internal.Client(nil)
	_ internal.Client                = Client(nil)
	_ CloudOperationsClient          = internal.CloudOperationsClient(nil)
	_ internal.CloudOperationsClient = CloudOperationsClient(nil)
	_ NamespaceClient                = internal.NamespaceClient(nil)
	_ internal.NamespaceClient       = NamespaceClient(nil)
)

// NewValue creates a new [converter.EncodedValue] which can be used to decode binary data returned by Temporal.  For example:
// User had Activity.RecordHeartbeat(ctx, "my-heartbeat") and then got response from calling Client.DescribeWorkflowExecution.
// The response contains binary field PendingActivityInfo.HeartbeatDetails,
// which can be decoded by using:
//
//	var result string // This need to be same type as the one passed to RecordHeartbeat
//	NewValue(data).Get(&result)
func NewValue(data *commonpb.Payloads) converter.EncodedValue {
	return internal.NewValue(data)
}

// NewValues creates a new [converter.EncodedValues] which can be used to decode binary data returned by Temporal. For example:
// User had Activity.RecordHeartbeat(ctx, "my-heartbeat", 123) and then got response from calling Client.DescribeWorkflowExecution.
// The response contains binary field PendingActivityInfo.HeartbeatDetails,
// which can be decoded by using:
//
//	var result1 string
//	var result2 int // These need to be same type as those arguments passed to RecordHeartbeat
//	NewValues(data).Get(&result1, &result2)
func NewValues(data *commonpb.Payloads) converter.EncodedValues {
	return internal.NewValues(data)
}

// HistoryJSONOptions are options for HistoryFromJSON.
type HistoryJSONOptions struct {
	// LastEventID, if set, will only load history up to this ID (inclusive).
	LastEventID int64
}

// HistoryFromJSON deserializes history from a reader of JSON bytes. This does
// not close the reader if it is closeable.
func HistoryFromJSON(r io.Reader, options HistoryJSONOptions) (*historypb.History, error) {
	return internal.HistoryFromJSON(r, options.LastEventID)
}

// NewAPIKeyStaticCredentials creates credentials that can be provided to
// ClientOptions to use a fixed API key.
//
// This is the equivalent of providing a headers provider that sets the
// "Authorization" header with "Bearer " + the given key. This will overwrite
// any "Authorization" header that may be on the context or from existing header
// provider.
//
// Note, this uses a fixed header value for authentication. Many users that want
// to rotate this value without reconnecting should use
// [NewAPIKeyDynamicCredentials].
func NewAPIKeyStaticCredentials(apiKey string) Credentials {
	return internal.NewAPIKeyStaticCredentials(apiKey)
}

// NewAPIKeyDynamicCredentials creates credentials powered by a callback that
// is invoked on each request. The callback accepts the context that is given by
// the calling user and can return a key or an error. When error is non-nil, the
// client call is failed with that error. When string is non-empty, it is used
// as the API key. When string is empty, nothing is set/overridden.
//
// This is the equivalent of providing a headers provider that returns the
// "Authorization" header with "Bearer " + the given function result. If the
// resulting string is non-empty, it will overwrite any "Authorization" header
// that may be on the context or from existing header provider.
func NewAPIKeyDynamicCredentials(apiKeyCallback func(context.Context) (string, error)) Credentials {
	return internal.NewAPIKeyDynamicCredentials(apiKeyCallback)
}

// NewMTLSCredentials creates credentials that use TLS with the client
// certificate as the given one. If the client options do not already enable
// TLS, this enables it. If the client options' TLS configuration is present and
// already has a client certificate, client creation will fail when applying
// these credentials.
func NewMTLSCredentials(certificate tls.Certificate) Credentials {
	return internal.NewMTLSCredentials(certificate)
}

// NewWorkflowUpdateServiceTimeoutOrCanceledError creates a new WorkflowUpdateServiceTimeoutOrCanceledError.
func NewWorkflowUpdateServiceTimeoutOrCanceledError(err error) *WorkflowUpdateServiceTimeoutOrCanceledError {
	return internal.NewWorkflowUpdateServiceTimeoutOrCanceledError(err)
}
