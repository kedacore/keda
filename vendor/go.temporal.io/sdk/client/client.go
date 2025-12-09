//go:generate mockgen -copyright_file ../LICENSE -package client -source client.go -destination client_mock.go

// Package client is used by external programs to communicate with Temporal service.
//
// NOTE: DO NOT USE THIS API INSIDE OF ANY WORKFLOW CODE!!!
package client

import (
	"context"
	"crypto/tls"
	"io"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal"
	"go.temporal.io/sdk/internal/common/metrics"
)

// DeploymentReachability specifies which category of tasks may reach a worker
// associated with a deployment, simplifying safe decommission.
//
// Deprecated: Use [WorkerDeploymentVersionDrainageStatus]
type DeploymentReachability = internal.DeploymentReachability

const (
	// DeploymentReachabilityUnspecified - Reachability level not specified.
	//
	// Deprecated: Use [WorkerDeploymentVersionDrainageStatus]
	DeploymentReachabilityUnspecified = internal.DeploymentReachabilityUnspecified

	// DeploymentReachabilityReachable - The deployment is reachable by new
	// and/or open workflows. The deployment cannot be decommissioned safely.
	//
	// Deprecated: Use [WorkerDeploymentVersionDrainageStatus]
	DeploymentReachabilityReachable = internal.DeploymentReachabilityReachable

	// DeploymentReachabilityClosedWorkflows - The deployment is not reachable
	// by new or open workflows, but might be still needed by
	// Queries sent to closed workflows. The deployment can be decommissioned
	// safely if user does not query closed workflows.
	//
	// Deprecated: Use [WorkerDeploymentVersionDrainageStatus]
	DeploymentReachabilityClosedWorkflows = internal.DeploymentReachabilityClosedWorkflows

	// DeploymentReachabilityUnreachable - The deployment is not reachable by
	// any workflow because all the workflows who needed this
	// deployment are out of the retention period. The deployment can be
	// decommissioned safely.
	//
	// Deprecated: Use [WorkerDeploymentVersionDrainageStatus]
	DeploymentReachabilityUnreachable = internal.DeploymentReachabilityUnreachable
)

// WorkerDeploymentVersionDrainageStatus specifies the drainage status for a Worker
// Deployment Version enabling users to decide when they can safely decommission this
// Version.
//
// NOTE: Experimental
type WorkerDeploymentVersionDrainageStatus = internal.WorkerDeploymentVersionDrainageStatus

const (
	// WorkerDeploymentVersionDrainageStatusUnspecified - Drainage status not specified.
	//
	// NOTE: Experimental
	WorkerDeploymentVersionDrainageStatusUnspecified = internal.WorkerDeploymentVersionDrainageStatusUnspecified

	// WorkerDeploymentVersionDrainageStatusDraining - The Worker Deployment Version is not
	// used by new workflows, but it is still used by open pinned workflows.
	// This Version cannot be decommissioned safely.
	//
	// NOTE: Experimental
	WorkerDeploymentVersionDrainageStatusDraining = internal.WorkerDeploymentVersionDrainageStatusDraining

	// WorkerDeploymentVersionDrainageStatusDrained - The Worker Deployment Version is not
	// used by new or open workflows, but it might still be needed to execute
	// Queries sent to closed workflows. This Version can be decommissioned safely if the user
	// does not expect to query closed workflows. In some cases this requires waiting for some
	// time after it is drained to guarantee no pending queries.
	//
	// NOTE: Experimental
	WorkerDeploymentVersionDrainageStatusDrained = internal.WorkerDeploymentVersionDrainageStatusDrained
)

// WorkerVersioningMode specifies whether the workflows processed by this
// worker use the worker's Version. The Temporal Server will use this worker's
// choice when dispatching tasks to it.
//
// NOTE: Experimental
type WorkerVersioningMode = internal.WorkerVersioningMode

const (
	// WorkerVersioningModeUnspecified - Versioning mode not reported.
	//
	// NOTE: Experimental
	WorkerVersioningModeUnspecified = internal.WorkerVersioningModeUnspecified

	// WorkerVersioningModeUnversioned - Workers with this mode are not
	// distinguished from each other for task routing, even if they
	// have different versions.
	//
	// NOTE: Experimental
	WorkerVersioningModeUnversioned = internal.WorkerVersioningModeUnversioned

	// WorkerVersioningModeVersioned - Workers with this mode are part of a
	// Worker Deployment Version which is a combination of a deployment name
	// and a build id.
	//
	// Each Deployment Version is distinguished from other Versions for task
	// routing, and users can configure the Temporal Server to send tasks to a
	// particular Version.
	//
	// NOTE: Experimental
	WorkerVersioningModeVersioned = internal.WorkerVersioningModeVersioned
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
//
// NOTE: future activities who inherit their workflow's Build ID but not its task queue will not be
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
type WorkflowUpdateStage = internal.WorkflowUpdateStage

const (
	// WorkflowUpdateStageUnspecified indicates the wait stage was not specified
	WorkflowUpdateStageUnspecified = internal.WorkflowUpdateStageUnspecified
	// WorkflowUpdateStageAdmitted indicates the update is admitted
	WorkflowUpdateStageAdmitted = internal.WorkflowUpdateStageAdmitted
	// WorkflowUpdateStageAccepted indicates the update is accepted
	WorkflowUpdateStageAccepted = internal.WorkflowUpdateStageAccepted
	// WorkflowUpdateStageCompleted indicates the update is completed
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

	// ConnectionOptions are optional parameters that can be specified in ClientOptions
	ConnectionOptions = internal.ConnectionOptions

	// Credentials are optional credentials that can be specified in ClientOptions.
	Credentials = internal.Credentials

	// StartWorkflowOptions configuration parameters for starting a workflow execution.
	StartWorkflowOptions = internal.StartWorkflowOptions

	// WithStartWorkflowOperation defines how to start a workflow when using UpdateWithStartWorkflow.
	// See [client.Client.NewWithStartWorkflowOperation] and [client.Client.UpdateWithStartWorkflow].
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

	// WorkflowExecutionDescription defines the response to DescribeWorkflow.
	WorkflowExecutionDescription = internal.WorkflowExecutionDescription

	// WorkflowExecutionMetadata defines common workflow information across multiple calls.
	WorkflowExecutionMetadata = internal.WorkflowExecutionMetadata

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
	UpdateWorkflowOptions = internal.UpdateWorkflowOptions

	// UpdateWithStartWorkflowOptions encapsulates the parameters used by UpdateWithStartWorkflow.
	// See [client.Client.UpdateWithStartWorkflow] and [client.Client.NewWithStartWorkflowOperation].
	UpdateWithStartWorkflowOptions = internal.UpdateWithStartWorkflowOptions

	// WorkerDeploymentDescribeOptions provides options for [WorkerDeploymentHandle.Describe].
	//
	// NOTE: Experimental
	WorkerDeploymentDescribeOptions = internal.WorkerDeploymentDescribeOptions

	// WorkerDeploymentVersionSummary provides a brief description of a Version.
	//
	// NOTE: Experimental
	WorkerDeploymentVersionSummary = internal.WorkerDeploymentVersionSummary

	// WorkerDeploymentInfo provides information about a Worker Deployment.
	//
	// NOTE: Experimental
	WorkerDeploymentInfo = internal.WorkerDeploymentInfo

	// WorkerDeploymentDescribeResponse is the response type for [WorkerDeploymentHandle.Describe].
	//
	// NOTE: Experimental
	WorkerDeploymentDescribeResponse = internal.WorkerDeploymentDescribeResponse

	// WorkerDeploymentSetCurrentVersionOptions provides options for
	// [WorkerDeploymentHandle.SetCurrentVersion].
	//
	// NOTE: Experimental
	WorkerDeploymentSetCurrentVersionOptions = internal.WorkerDeploymentSetCurrentVersionOptions

	// WorkerDeploymentSetCurrentVersionResponse is the response for
	// [WorkerDeploymentHandle.SetCurrentVersion].
	//
	// NOTE: Experimental
	WorkerDeploymentSetCurrentVersionResponse = internal.WorkerDeploymentSetCurrentVersionResponse

	// WorkerDeploymentSetRampingVersionOptions provides options for
	// [WorkerDeploymentHandle.SetRampingVersion].
	//
	// NOTE: Experimental
	WorkerDeploymentSetRampingVersionOptions = internal.WorkerDeploymentSetRampingVersionOptions

	// WorkerDeploymentSetRampingVersionResponse is the response for
	// [WorkerDeploymentHandle.SetRampingVersion].
	//
	// NOTE: Experimental
	WorkerDeploymentSetRampingVersionResponse = internal.WorkerDeploymentSetRampingVersionResponse

	// WorkerDeploymentSetManagerIdentityOptions provides options for
	// [WorkerDeploymentHandle.SetManagerIdentity].
	//
	// NOTE: Experimental
	WorkerDeploymentSetManagerIdentityOptions = internal.WorkerDeploymentSetManagerIdentityOptions

	// WorkerDeploymentSetManagerIdentityResponse is the response for
	// [WorkerDeploymentHandle.SetManagerIdentity].
	//
	// NOTE: Experimental
	WorkerDeploymentSetManagerIdentityResponse = internal.WorkerDeploymentSetManagerIdentityResponse

	// WorkerDeploymentDescribeVersionOptions provides options for
	// [WorkerDeploymentHandle.DescribeVersion].
	//
	// NOTE: Experimental
	WorkerDeploymentDescribeVersionOptions = internal.WorkerDeploymentDescribeVersionOptions

	// WorkerDeploymentTaskQueueInfo describes properties of the Task Queues involved
	// in a Deployment Version.
	//
	// NOTE: Experimental
	WorkerDeploymentTaskQueueInfo = internal.WorkerDeploymentTaskQueueInfo

	// WorkerDeploymentVersionDrainageInfo describes drainage properties of a Deployment Version.
	// This enables users to safely decide when they can decommission a Version.
	//
	// NOTE: Experimental
	WorkerDeploymentVersionDrainageInfo = internal.WorkerDeploymentVersionDrainageInfo

	// WorkerDeploymentVersionInfo provides information about a Worker Deployment Version.
	//
	// NOTE: Experimental
	WorkerDeploymentVersionInfo = internal.WorkerDeploymentVersionInfo

	// WorkerDeploymentVersionDescription is the response for
	// [WorkerDeploymentHandle.DescribeVersion].
	//
	// NOTE: Experimental
	WorkerDeploymentVersionDescription = internal.WorkerDeploymentVersionDescription

	// WorkerDeploymentDeleteVersionOptions provides options for
	// [WorkerDeploymentHandle.DeleteVersion].
	//
	// NOTE: Experimental
	WorkerDeploymentDeleteVersionOptions = internal.WorkerDeploymentDeleteVersionOptions

	// WorkerDeploymentDeleteVersionResponse is the response for
	// [WorkerDeploymentHandle.DeleteVersion].
	//
	// NOTE: Experimental
	WorkerDeploymentDeleteVersionResponse = internal.WorkerDeploymentDeleteVersionResponse

	// WorkerDeploymentMetadataUpdate modifies user-defined metadata entries that describe
	// a Version.
	//
	// NOTE: Experimental
	WorkerDeploymentMetadataUpdate = internal.WorkerDeploymentMetadataUpdate

	// WorkerDeploymentUpdateVersionMetadataOptions provides options for
	// [WorkerDeploymentHandle.UpdateVersionMetadata].
	//
	// NOTE: Experimental
	WorkerDeploymentUpdateVersionMetadataOptions = internal.WorkerDeploymentUpdateVersionMetadataOptions

	// WorkerDeploymentUpdateVersionMetadataResponse is the response for
	// [WorkerDeploymentHandle.UpdateVersionMetadata].
	//
	// NOTE: Experimental
	WorkerDeploymentUpdateVersionMetadataResponse = internal.WorkerDeploymentUpdateVersionMetadataResponse

	// WorkerDeploymentHandle is a handle to a Worker Deployment.
	//
	// NOTE: Experimental
	WorkerDeploymentHandle = internal.WorkerDeploymentHandle

	// DeploymentListOptions are the parameters for configuring listing Worker Deployments.
	//
	// NOTE: Experimental
	WorkerDeploymentListOptions = internal.WorkerDeploymentListOptions

	// WorkerDeploymentRoutingConfig describes when new or existing Workflow Tasks are
	// executed with this Worker Deployment.
	//
	// NOTE: Experimental
	WorkerDeploymentRoutingConfig = internal.WorkerDeploymentRoutingConfig

	// WorkerDeploymentListEntry is a subset of fields from [WorkerDeploymentInfo].
	//
	// NOTE: Experimental
	WorkerDeploymentListEntry = internal.WorkerDeploymentListEntry

	// WorkerDeploymentListIterator is an iterator for deployments.
	//
	// NOTE: Experimental
	WorkerDeploymentListIterator = internal.WorkerDeploymentListIterator

	// WorkerDeploymentDeleteOptions provides options for [WorkerDeploymentClient.Delete].
	//
	// NOTE: Experimental
	WorkerDeploymentDeleteOptions = internal.WorkerDeploymentDeleteOptions

	// WorkerDeploymentDeleteResponse is the response for [WorkerDeploymentClient.Delete].
	//
	// NOTE: Experimental
	WorkerDeploymentDeleteResponse = internal.WorkerDeploymentDeleteResponse

	// WorkerDeploymentClient is the client that manages Worker Deployments.
	//
	// NOTE: Experimental
	WorkerDeploymentClient = internal.WorkerDeploymentClient

	// Deployment identifies a set of workers. This identifier combines
	// the deployment series name with their Build ID.
	//
	// Deprecated: Use the new Worker Deployment API
	Deployment = internal.Deployment

	// DeploymentTaskQueueInfo describes properties of the Task Queues involved
	// in a deployment.
	//
	// Deprecated: Use [WorkerDeploymentTaskQueueInfo]
	DeploymentTaskQueueInfo = internal.DeploymentTaskQueueInfo

	// DeploymentInfo holds information associated with
	// workers in this deployment.
	// Workers can poll multiple task queues in a single deployment,
	// which are listed in this message.
	//
	// Deprecated: Use [WorkerDeploymentInfo]
	DeploymentInfo = internal.DeploymentInfo

	// DeploymentListEntry is a subset of fields from DeploymentInfo.
	//
	// Deprecated: Use [WorkerDeploymentListEntry]
	DeploymentListEntry = internal.DeploymentListEntry

	// DeploymentListIterator is an iterator for deployments.
	//
	// Deprecated: Use [WorkerDeploymentListIterator]
	DeploymentListIterator = internal.DeploymentListIterator

	// DeploymentListOptions are the parameters for configuring listing deployments.
	//
	// Deprecated: Use [WorkerDeploymentListOptions]
	DeploymentListOptions = internal.DeploymentListOptions

	// DeploymentReachabilityInfo extends DeploymentInfo with reachability information.
	//
	// Deprecated: Use [WorkerDeploymentVersionDrainageInfo]
	DeploymentReachabilityInfo = internal.DeploymentReachabilityInfo

	// DeploymentMetadataUpdate modifies user-defined metadata entries that describe
	// a deployment.
	//
	// Deprecated: Use [WorkerDeploymentMetadataUpdate]
	DeploymentMetadataUpdate = internal.DeploymentMetadataUpdate

	// DeploymentDescribeOptions provides options for [DeploymentClient.Describe].
	//
	// Deprecated: Use [WorkerDeploymentDescribeOptions]
	DeploymentDescribeOptions = internal.DeploymentDescribeOptions

	// DeploymentDescription is the response type for [DeploymentClient.Describe].
	//
	// Deprecated: Use [WorkerDeploymentDescribeResponse]
	DeploymentDescription = internal.DeploymentDescription

	// DeploymentGetReachabilityOptions provides options for [DeploymentClient.GetReachability].
	//
	// Deprecated: Use [WorkerDeploymentDescribeResponse]
	DeploymentGetReachabilityOptions = internal.DeploymentGetReachabilityOptions

	// DeploymentGetCurrentOptions provides options for [DeploymentClient.GetCurrent].
	//
	// Deprecated: Use [WorkerDeploymentDescribeOptions]
	DeploymentGetCurrentOptions = internal.DeploymentGetCurrentOptions

	// DeploymentGetCurrentResponse is the response type for [DeploymentClient.GetCurrent].
	//
	// Deprecated: Use [WorkerDeploymentDescribeResponse]
	DeploymentGetCurrentResponse = internal.DeploymentGetCurrentResponse

	// DeploymentSetCurrentOptions provides options for [DeploymentClient.SetCurrent].
	//
	// Deprecated: Use [WorkerDeploymentSetCurrentVersionOptions]
	DeploymentSetCurrentOptions = internal.DeploymentSetCurrentOptions

	// DeploymentSetCurrentResponse is the response type for [DeploymentClient.SetCurrent].
	//
	// Deprecated: Use [WorkerDeploymentSetCurrentVersionResponse]
	DeploymentSetCurrentResponse = internal.DeploymentSetCurrentResponse

	// DeploymentClient is the server interface to manage deployments.
	//
	// Deprecated: Use [WorkerDeploymentClient]
	DeploymentClient = internal.DeploymentClient

	// UpdateWorkflowExecutionOptionsRequest is a request for [client.Client.UpdateWorkflowExecutionOptions].
	//
	// NOTE: Experimental
	UpdateWorkflowExecutionOptionsRequest = internal.UpdateWorkflowExecutionOptionsRequest

	// WorkflowExecutionOptions contains a set of properties of an existing workflow
	// that can be overriden using [client.Client.UpdateWorkflowExecutionOptions].
	//
	// NOTE: Experimental
	WorkflowExecutionOptions = internal.WorkflowExecutionOptions

	// WorkflowExecutionOptionsChanges describes changes to [WorkflowExecutionOptions]
	// in the [client.Client.UpdateWorkflowExecutionOptions] API.
	//
	// NOTE: Experimental
	WorkflowExecutionOptionsChanges = internal.WorkflowExecutionOptionsChanges

	// VersioningOverrideChange sets or removes a versioning override when used with
	// [WorkflowExecutionOptionsChanges].
	//
	// NOTE: Experimental
	VersioningOverrideChange = internal.VersioningOverrideChange

	// VersioningOverride is a property in [WorkflowExecutionOptions] that changes the versioning
	// configuration of a specific workflow execution.
	//
	// If set, it takes precedence over the Versioning Behavior provided with workflow type
	// registration, or default worker options.
	//
	// NOTE: Experimental
	VersioningOverride = internal.VersioningOverride

	// PinnedVersioningOverride means the workflow will be pinned to a specific deployment version.
	//
	// NOTE: Experimental
	PinnedVersioningOverride = internal.PinnedVersioningOverride

	// AutoUpgradeVersioningOverride means the workflow will auto-upgrade to the current deployment
	// version on the next workflow task.
	//
	// NOTE: Experimental
	AutoUpgradeVersioningOverride = internal.AutoUpgradeVersioningOverride

	// WorkflowUpdateHandle represents a running or completed workflow
	// execution update and gives the holder access to the outcome of the same.
	WorkflowUpdateHandle = internal.WorkflowUpdateHandle

	// GetWorkflowUpdateHandleOptions encapsulates the parameters needed to unambiguously
	// refer to a Workflow Update
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

	// DescribeTaskQueueEnhancedOptions is the input to [client.Client.DescribeTaskQueueEnhanced].
	DescribeTaskQueueEnhancedOptions = internal.DescribeTaskQueueEnhancedOptions

	// TaskQueueVersionSelection is a task queue filter based on versioning.
	// It is an optional component of [DescribeTaskQueueEnhancedOptions].
	// WARNING: Worker versioning is currently experimental.
	TaskQueueVersionSelection = internal.TaskQueueVersionSelection

	// TaskQueueDescription is the response to [client.Client.DescribeTaskQueueEnhanced].
	TaskQueueDescription = internal.TaskQueueDescription

	// TaskQueueVersionInfo includes task queue information per Build ID.
	// It is part of [TaskQueueDescription].
	//
	// Deprecated: Use [TaskQueueVersioningInfo]
	TaskQueueVersionInfo = internal.TaskQueueVersionInfo

	// TaskQueueVersioningInfo provides worker deployment configuration for this
	// task queue.
	// It is part of [Client.TaskQueueDescription].
	//
	// NOTE: Experimental
	TaskQueueVersioningInfo = internal.TaskQueueVersioningInfo

	// TaskQueueTypeInfo specifies task queue information per task type and Build ID.
	// It is included in [TaskQueueVersionInfo].
	TaskQueueTypeInfo = internal.TaskQueueTypeInfo

	// TaskQueuePollerInfo provides information about a worker/client polling a task queue.
	// It is used by [TaskQueueTypeInfo].
	TaskQueuePollerInfo = internal.TaskQueuePollerInfo

	// TaskQueueStats contains statistics about task queue backlog and activity.
	//
	// For workflow task queue type, this result is partial because tasks sent to sticky queues are not included. Read
	// comments above each metric to understand the impact of sticky queue exclusion on that metric accuracy.
	TaskQueueStats = internal.TaskQueueStats

	// WorkerVersionCapabilities includes a worker's build identifier
	// and whether it is choosing to use the versioning feature.
	// It is an optional component of [TaskQueuePollerInfo].
	// WARNING: Worker versioning is currently experimental.
	WorkerVersionCapabilities = internal.WorkerVersionCapabilities

	// UpdateWorkerVersioningRulesOptions is the input to [client.Client.UpdateWorkerVersioningRules].
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	UpdateWorkerVersioningRulesOptions = internal.UpdateWorkerVersioningRulesOptions //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningConflictToken is a conflict token to serialize calls to [client.Client.UpdateWorkerVersioningRules].
	// An update with an old token fails with `serviceerror.FailedPrecondition`.
	// The current token can be obtained with [client.Client.GetWorkerVersioningRules],
	// or returned by a successful [client.Client.UpdateWorkerVersioningRules].
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningConflictToken = internal.VersioningConflictToken //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningRampByPercentage is a VersionRamp that sends a proportion of the traffic
	// to the target Build ID.
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningRampByPercentage = internal.VersioningRampByPercentage //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningAssignmentRule is a BuildID  assigment rule for a task queue.
	// Assignment rules only affect new workflows.
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningAssignmentRule = internal.VersioningAssignmentRule //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningAssignmentRuleWithTimestamp contains an assignment rule annotated
	// by the server with its creation time.
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningAssignmentRuleWithTimestamp = internal.VersioningAssignmentRuleWithTimestamp //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningAssignmentRule is a BuildID redirect rule for a task queue.
	// It changes the behavior of currently running workflows and new ones.
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningRedirectRule = internal.VersioningRedirectRule //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningRedirectRuleWithTimestamp contains a redirect rule annotated
	// by the server with its creation time.
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningRedirectRuleWithTimestamp = internal.VersioningRedirectRuleWithTimestamp //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningOperationInsertAssignmentRule is an operation for UpdateWorkerVersioningRulesOptions
	// that inserts the rule to the list of assignment rules for this Task Queue.
	// The rules are evaluated in order, starting from index 0. The first
	// applicable rule will be applied and the rest will be ignored.
	// By default, the new rule is inserted at the beginning of the list
	// (index 0). If the given index is too larger the rule will be
	// inserted at the end of the list.
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationInsertAssignmentRule = internal.VersioningOperationInsertAssignmentRule //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningOperationReplaceAssignmentRule is an operation for UpdateWorkerVersioningRulesOptions
	// that replaces the assignment rule at a given index. By default presence of one
	// unconditional rule, i.e., no hint filter or ramp, is enforced, otherwise
	// the delete operation will be rejected. Set `force` to true to
	// bypass this validation.
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationReplaceAssignmentRule = internal.VersioningOperationReplaceAssignmentRule //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningOperationDeleteAssignmentRule is an operation for UpdateWorkerVersioningRulesOptions
	// that deletes the assignment rule at a given index. By default presence of one
	// unconditional rule, i.e., no hint filter or ramp, is enforced, otherwise
	// the delete operation will be rejected. Set `force` to true to
	// bypass this validation.
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationDeleteAssignmentRule = internal.VersioningOperationDeleteAssignmentRule //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningOperationAddRedirectRule is an operation for UpdateWorkerVersioningRulesOptions
	// that adds the rule to the list of redirect rules for this Task Queue. There
	// can be at most one redirect rule for each distinct Source BuildID.
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationAddRedirectRule = internal.VersioningOperationAddRedirectRule //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningOperationReplaceRedirectRule is an operation for UpdateWorkerVersioningRulesOptions
	// that replaces the routing rule with the given source BuildID.
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationReplaceRedirectRule = internal.VersioningOperationReplaceRedirectRule //lint:ignore SA1019 transitioning to Worker Deployments

	// VersioningOperationDeleteRedirectRule is an operation for UpdateWorkerVersioningRulesOptions
	// that deletes the routing rule with the given source Build ID.
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationDeleteRedirectRule = internal.VersioningOperationDeleteRedirectRule //lint:ignore SA1019 transitioning to Worker Deployments

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
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	VersioningOperationCommitBuildID = internal.VersioningOperationCommitBuildID //lint:ignore SA1019 transitioning to Worker Deployments

	// GetWorkerVersioningOptions is the input to [client.Client.GetWorkerVersioningRules].
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	GetWorkerVersioningOptions = internal.GetWorkerVersioningOptions //lint:ignore SA1019 transitioning to Worker Deployments

	// WorkerVersioningRules is the response for [client.Client.GetWorkerVersioningRules].
	//
	// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
	//
	// WARNING: Worker versioning is currently experimental.
	WorkerVersioningRules = internal.WorkerVersioningRules //lint:ignore SA1019 transitioning to Worker Deployments

	// WorkflowUpdateServiceTimeoutOrCanceledError is an error that occurs when an update call times out or is cancelled.
	//
	// Note, this is not related to any general concept of timing out or cancelling a running update, this is only related to the client call itself.
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
		//  - serviceerror.WorkflowExecutionAlreadyStarted, when WorkflowExecutionErrorWhenAlreadyStarted is specified
		//
		// WorkflowRun has 3 methods:
		//  - GetWorkflowID() string: which return the started workflow ID
		//  - GetRunID() string: which return the first started workflow run ID (please see below)
		//  - Get(ctx context.Context, valuePtr interface{}) error: which will fill the workflow
		//    execution result to valuePtr, if workflow execution is a success, or return corresponding
		//    error. This is a blocking API.
		//
		// NOTE: If the started workflow returns ContinueAsNewError during the workflow execution, the
		// returned result of GetRunID() will be the started workflow run ID, not the new run ID caused by ContinueAsNewError.
		// However, Get(ctx context.Context, valuePtr interface{}) will return result from the run which did not return ContinueAsNewError.
		// Say ExecuteWorkflow started a workflow, in its first run, has run ID "run ID 1", and returned ContinueAsNewError,
		// the second run has run ID "run ID 2" and return some result other than ContinueAsNewError:
		// GetRunID() will always return "run ID 1" and  Get(ctx context.Context, valuePtr interface{}) will return the result of second run.
		//
		// NOTE: DO NOT USE THIS API INSIDE A WORKFLOW, USE workflow.ExecuteChildWorkflow instead
		ExecuteWorkflow(ctx context.Context, options StartWorkflowOptions, workflow interface{}, args ...interface{}) (WorkflowRun, error)

		// GetWorkflow retrieves a workflow execution and return a WorkflowRun instance (described above)
		//  - workflow ID of the workflow.
		//  - runID can be default(empty string). if empty string then it will pick the last running execution of that workflow ID.
		//
		// WorkflowRun has 2 methods:
		//  - GetRunID() string: which return the first started workflow run ID (please see below)
		//  - Get(ctx context.Context, valuePtr interface{}) error: which will fill the workflow
		//    execution result to valuePtr, if workflow execution is a success, or return corresponding
		//    error. This is a blocking API.
		// If workflow not found, the Get() will return serviceerror.NotFound.
		//
		// NOTE: if the started workflow return ContinueAsNewError during the workflow execution, the
		// return result of GetRunID() will be the started workflow run ID, not the new run ID caused by ContinueAsNewError,
		// however, Get(ctx context.Context, valuePtr interface{}) will return result from the run which did not return ContinueAsNewError.
		// Say ExecuteWorkflow started a workflow, in its first run, has run ID "run ID 1", and returned ContinueAsNewError,
		// the second run has run ID "run ID 2" and return some result other than ContinueAsNewError:
		// GetRunID() will always return "run ID 1" and  Get(ctx context.Context, valuePtr interface{}) will return the result of second run.
		GetWorkflow(ctx context.Context, workflowID string, runID string) WorkflowRun

		// SignalWorkflow sends a signals to a workflow in execution
		//  - workflow ID of the workflow.
		//  - runID can be default(empty string). if empty string then it will pick the running execution of that workflow ID.
		//  - signalName name to identify the signal.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error

		// SignalWithStartWorkflow sends a signal to a running workflow.
		// If the workflow is not running or not found, it starts the workflow and then sends the signal in transaction.
		//  - workflowID, signalName, signalArg are same as SignalWorkflow's parameters
		//  - options, workflow, workflowArgs are same as StartWorkflow's parameters
		//  - the workflowID parameter is used instead of options.ID. If the latter is present, it must match the workflowID.
		//
		// NOTE: options.WorkflowIDReusePolicy is default to AllowDuplicate in this API.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		SignalWithStartWorkflow(ctx context.Context, workflowID string, signalName string, signalArg interface{},
			options StartWorkflowOptions, workflow interface{}, workflowArgs ...interface{}) (WorkflowRun, error)

		// NewWithStartWorkflowOperation returns a WithStartWorkflowOperation for use with UpdateWithStartWorkflow.
		// See [client.Client.UpdateWithStartWorkflow].
		NewWithStartWorkflowOperation(options StartWorkflowOptions, workflow interface{}, args ...interface{}) WithStartWorkflowOperation

		// CancelWorkflow request cancellation of a workflow in execution. Cancellation request closes the channel
		// returned by the workflow.Context.Done() of the workflow that is target of the request.
		//  - workflow ID of the workflow.
		//  - runID can be default(empty string). if empty string then it will pick the currently running execution of that workflow ID.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		CancelWorkflow(ctx context.Context, workflowID string, runID string) error

		// TerminateWorkflow terminates a workflow execution. Terminate stops a workflow execution immediately without
		// letting the workflow to perform any cleanup
		// workflowID is required, other parameters are optional.
		//  - workflow ID of the workflow.
		//  - runID can be default(empty string). if empty string then it will pick the running execution of that workflow ID.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		TerminateWorkflow(ctx context.Context, workflowID string, runID string, reason string, details ...interface{}) error

		// GetWorkflowHistory gets history events of a particular workflow
		//  - workflow ID of the workflow.
		//  - runID can be default(empty string). if empty string then it will pick the last running execution of that workflow ID.
		//  - whether use long poll for tracking new events: when the workflow is running, there can be new events generated during iteration
		//    of HistoryEventIterator, if isLongPoll == true, then iterator will do long poll, tracking new history event, i.e. the iteration
		//   will not be finished until workflow is finished; if isLongPoll == false, then iterator will only return current history events.
		//  - whether return all history events or just the last event, which contains the workflow execution end result
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
		// details - is the progress you want to record along with heart beat for this activity. If the activity is canceled,
		// the error returned will be a CanceledError. If the activity is paused by the server, the error returned will be a
		// ErrActivityPaused. If the activity is reset by the server, the error returned will be a ErrActivityReset.
		// Otherwise the errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		RecordActivityHeartbeat(ctx context.Context, taskToken []byte, details ...interface{}) error

		// RecordActivityHeartbeatByID records heartbeat for an activity.
		// details - is the progress you want to record along with heart beat for this activity. If the activity is canceled,
		// the error returned will be a CanceledError. If the activity is paused by the server, the error returned will be a
		// ErrActivityPaused. If the activity is reset by the server, the error returned will be a ErrActivityReset.
		// The errors it can return:
		//  - serviceerror.NotFound
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		RecordActivityHeartbeatByID(ctx context.Context, namespace, workflowID, runID, activityID string, details ...interface{}) error

		// ListClosedWorkflow gets closed workflow executions based on request filters.
		// Retrieved workflow executions are sorted by close time in descending order.
		//
		// NOTE: heavy usage of this API may cause huge persistence pressure.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NamespaceNotFound
		ListClosedWorkflow(ctx context.Context, request *workflowservice.ListClosedWorkflowExecutionsRequest) (*workflowservice.ListClosedWorkflowExecutionsResponse, error)

		// ListOpenWorkflow gets open workflow executions based on request filters.
		// Retrieved workflow executions are sorted by start time in descending order.
		//
		// NOTE: heavy usage of this API may cause huge persistence pressure.
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NamespaceNotFound
		ListOpenWorkflow(ctx context.Context, request *workflowservice.ListOpenWorkflowExecutionsRequest) (*workflowservice.ListOpenWorkflowExecutionsResponse, error)

		// ListWorkflow gets workflow executions based on query. The query is basically the SQL WHERE clause, examples:
		//  - "(WorkflowId = 'wid1' or (WorkflowType = 'type2' and WorkflowId = 'wid2'))".
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
		//
		// Deprecated: Use ListWorkflow instead.
		ScanWorkflow(ctx context.Context, request *workflowservice.ScanWorkflowExecutionsRequest) (*workflowservice.ScanWorkflowExecutionsResponse, error) //lint:ignore SA1019 the server API was deprecated.

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
		//
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
		//  - workflowID is required.
		//  - runID can be default(empty string). if empty string then it will pick the running execution of that workflow ID.
		//  - queryType is the type of the query.
		//  - args... are the optional query parameters.
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
		//  - runID can be default(empty string). if empty string then it will pick the last running execution of that workflow ID.
		//
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NotFound
		DescribeWorkflowExecution(ctx context.Context, workflowID, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error)

		// DescribeWorkflow returns information about the specified workflow execution.
		//  - runID can be default(empty string). if empty string then it will pick the last running execution of that workflow ID.
		//
		// The errors it can return:
		//  - serviceerror.InvalidArgument
		//  - serviceerror.Internal
		//  - serviceerror.Unavailable
		//  - serviceerror.NotFound
		DescribeWorkflow(ctx context.Context, workflowID, runID string) (*WorkflowExecutionDescription, error)

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
		// Deprecated: Use [client.Client.UpdateWorkerVersioningRules] with the versioning api.
		UpdateWorkerBuildIdCompatibility(ctx context.Context, options *UpdateWorkerBuildIdCompatibilityOptions) error

		// GetWorkerBuildIdCompatibility
		// Returns the worker-build-id based version sets for a particular task queue.
		//
		// Deprecated: Use [client.Client.GetWorkerVersioningRules] with the versioning api.
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
		//
		// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
		//
		// WARNING: Worker versioning is currently experimental, and requires server 1.24+
		UpdateWorkerVersioningRules(ctx context.Context, options UpdateWorkerVersioningRulesOptions) (*WorkerVersioningRules, error)

		// GetWorkerVersioningRules
		// Returns the worker-build-id assignment and redirect rules for a task queue.
		//
		// Deprecated: Build-id based versioning is deprecated in favor of worker deployment based versioning and will be removed soon.
		//
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
		UpdateWorkflow(ctx context.Context, options UpdateWorkflowOptions) (WorkflowUpdateHandle, error)

		// UpdateWorkflowExecutionOptions partially overrides the [WorkflowExecutionOptions] of an existing workflow execution
		// and returns the new [WorkflowExecutionOptions] after applying the changes.
		// It is intended for building tools that can selectively apply ad-hoc workflow configuration changes.
		// Use [DescribeWorkflowExecution] to get similar information without modifying options.
		//
		// NOTE: Experimental
		UpdateWorkflowExecutionOptions(ctx context.Context, options UpdateWorkflowExecutionOptionsRequest) (WorkflowExecutionOptions, error)

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
		UpdateWithStartWorkflow(ctx context.Context, options UpdateWithStartWorkflowOptions) (WorkflowUpdateHandle, error)

		// GetWorkflowUpdateHandle creates a handle to the referenced update
		// which can be polled for an outcome. Note that runID is optional and
		// if not specified the most recent runID will be used.
		GetWorkflowUpdateHandle(ref GetWorkflowUpdateHandleOptions) WorkflowUpdateHandle

		// WorkflowService provides access to the underlying gRPC service. This should only be used for advanced use cases
		// that cannot be accomplished via other Client methods. Unlike calls to other Client methods, calls directly to the
		// service are not configured with internal semantics such as automatic retries.
		WorkflowService() workflowservice.WorkflowServiceClient

		// OperatorService creates a new operator service client with the same gRPC connection as this client.
		OperatorService() operatorservice.OperatorServiceClient

		// Schedule creates a new shedule client with the same gRPC connection as this client.
		ScheduleClient() ScheduleClient

		// DeploymentClient create a new deployment client with the same gRPC connection as this client.
		//
		// Deprecated: use [WorkerDeploymentClient]
		DeploymentClient() DeploymentClient

		// WorkerDeploymentClient create a new worker deployment client with the same gRPC connections as this client.
		//
		// NOTE: Experimental
		WorkerDeploymentClient() WorkerDeploymentClient

		// Close client and clean up underlying resources.
		//
		// If this client was created via NewClientFromExisting or this client has
		// been used in that call, Close() on may not necessarily close the
		// underlying connection. Only the final close of all existing clients will
		// close the underlying connection.
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

// NewNamespaceClient creates an instance of a namespace client, to manage
// lifecycle of namespaces. This will not attempt to connect to the server
// eagerly and therefore may not fail for an unreachable server until a call is
// made.
func NewNamespaceClient(options Options) (NamespaceClient, error) {
	return internal.NewNamespaceClient(options)
}

// make sure if new methods are added to internal.Client they are also added to public Client.
var (
	_ Client                   = internal.Client(nil)
	_ internal.Client          = Client(nil)
	_ NamespaceClient          = internal.NamespaceClient(nil)
	_ internal.NamespaceClient = NamespaceClient(nil)
)

// NewValue creates a new [converter.EncodedValue] which can be used to decode binary data returned by Temporal.  For example:
// User had Activity.RecordHeartbeat(ctx, "my-heartbeat") and then got response from calling [client.Client.DescribeWorkflowExecution].
// The response contains binary field PendingActivityInfo.HeartbeatDetails,
// which can be decoded by using:
//
//	var result string // This need to be same type as the one passed to RecordHeartbeat
//	NewValue(data).Get(&result)
func NewValue(data *commonpb.Payloads) converter.EncodedValue {
	return internal.NewValue(data)
}

// NewValues creates a new [converter.EncodedValues] which can be used to decode binary data returned by Temporal. For example:
// User has Activity.RecordHeartbeat(ctx, "my-heartbeat", 123) and then got a response from calling [client.Client.DescribeWorkflowExecution].
// The response contains the binary field PendingActivityInfo.HeartbeatDetails,
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
