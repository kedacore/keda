package internal

import (
	"context"
	"time"

	commonpb "go.temporal.io/api/common/v1"
)

// WorkerDeploymentVersionDrainageStatus specifies the drainage status for a Worker
// Deployment Version enabling users to decide when they can safely decommission this
// Version.
//
// NOTE: Experimental
//
// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentVersionDrainageStatus]
type WorkerDeploymentVersionDrainageStatus int

const (
	// WorkerDeploymentVersionDrainageStatusUnspecified - Drainage status not specified.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentVersionDrainageStatusUnspecified]
	WorkerDeploymentVersionDrainageStatusUnspecified = iota

	// WorkerDeploymentVersionDrainageStatusDraining - The Worker Deployment Version is not
	// used by new workflows, but it is still used by open pinned workflows.
	// This Version cannot be decommissioned safely.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentVersionDrainageStatusDraining]
	WorkerDeploymentVersionDrainageStatusDraining

	// WorkerDeploymentVersionDrainageStatusDrained - The Worker Deployment Version is not
	// used by new or open workflows, but it might still be needed to execute
	// Queries sent to closed workflows. This Version can be decommissioned safely if the user
	// does not expect to query closed workflows. In some cases this requires waiting for some
	// time after it is drained to guarantee no pending queries.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentVersionDrainageStatusDrained]
	WorkerDeploymentVersionDrainageStatusDrained
)

type (

	// WorkerDeploymentDescribeOptions provides options for [WorkerDeploymentHandle.Describe].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentDescribeOptions]
	WorkerDeploymentDescribeOptions struct {
	}

	// WorkerDeploymentVersionSummary provides a brief description of a Version.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentVersionSummary]
	WorkerDeploymentVersionSummary struct {
		// Version - The version
		Version WorkerDeploymentVersion

		// CreateTime - When this Deployment Version was created.
		CreateTime time.Time

		// DrainageStatus - The Worker Deployment Version drainage status to guarantee safe
		// decommission of this Version.
		DrainageStatus WorkerDeploymentVersionDrainageStatus
	}

	// WorkerDeploymentInfo provides information about a Worker Deployment.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentInfo]
	WorkerDeploymentInfo struct {
		// Name - Identifies a Worker Deployment. Must be unique within the namespace. Cannot
		// contain ".", a reserved character.
		Name string

		// CreateTime - When this deployment was created.
		CreateTime time.Time

		// VersionSummaries - A brief description of the Deployment Versions that are currently
		// tracked in this Deployment.
		// A DeploymentVersion will be cleaned up automatically if all the following conditions are met:
		// - It does not receive new executions, i.e., it is not current or ramping.
		// - It has no active pollers.
		// - It is drained.
		VersionSummaries []WorkerDeploymentVersionSummary

		// RoutingConfig - When to execute new or existing Workflow Tasks with this Deployment.
		RoutingConfig WorkerDeploymentRoutingConfig

		// LastModifierIdentity - The identity of the last client that modified the
		// configuration of this Deployment.
		LastModifierIdentity string

		// ManagerIdentity - When present, clients whose identity does not match `ManagerIdentity` will not
		// be able to make changes to this Worker Deployment. They can either set their own identity as the
		// manager or unset the field to proceed. Empty by default.
		ManagerIdentity string
	}

	// WorkerDeploymentDescribeResponse is the response type for [WorkerDeploymentHandle.Describe].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentDescribeResponse]
	WorkerDeploymentDescribeResponse struct {
		// ConflictToken - Token to serialize Worker Deployment operations.
		ConflictToken []byte

		// Info - Description of this Worker Deployment.
		Info WorkerDeploymentInfo
	}

	// WorkerDeploymentSetCurrentVersionOptions provides options for
	// [WorkerDeploymentHandle.SetCurrentVersion].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentSetCurrentVersionOptions]
	WorkerDeploymentSetCurrentVersionOptions struct {
		// BuildID - A Build ID within this deployment to set as the current version. If empty, the
		// current version will target unversioned workers.
		BuildID string

		// ConflictToken - Token to serialize Worker Deployment operations. Passing a non-empty
		// conflict token will cause this request to fail with
		// `serviceerror.FailedPrecondition` if the
		// Deployment's configuration has been modified between the API call that
		// generated the token and this one.
		// The current token can be obtained with [WorkerDeploymentHandle.Describe],
		// or returned by other successful Worker Deployment operations.
		//
		// Optional: defaults to empty token, which bypasses conflict detection.
		ConflictToken []byte

		// Identity - The identity of the client who initiated this request.
		//
		// Optional: defaults to the identity of the underlying workflow client.
		Identity string

		// IgnoreMissingTaskQueues - Override protection against accidental removal of Task Queues.
		// When false this request would be rejected if not all the expected Task Queues are
		// being polled by Workers in the new Version.
		// The set of expected Task Queues contains all the Task Queues that were ever polled by
		// the existing Current Version of the Deployment, with the following exclusions:
		//   - Task Queues that are no longer used, i.e., with empty backlog and no recently added tasks.
		//   - Task Queues moved to another Worker Deployment, i.e., current in a different Deployment.
		// WARNING: setting this flag could lead to missing Task Queues polled by late starting
		// Workers.
		//
		// Optional: default to reject request when queues are missing.
		IgnoreMissingTaskQueues bool

		// AllowNoPollers - Override protection against accidentally sending tasks to a version without pollers.
		// When false this request will be rejected if no pollers have been seen for the proposed Current Version,
		// in order to protect users from routing tasks to pollers that do not exist, leading to possible timeouts.
		// Pass `true` here to bypass this protection.
		// WARNING: setting this flag could lead to tasks being sent to a version that has no pollers.
		//
		// Optional: default to reject request when version has never had pollers.
		AllowNoPollers bool
	}

	// WorkerDeploymentSetCurrentVersionResponse is the response for
	// [WorkerDeploymentHandle.SetCurrentVersion].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentSetCurrentVersionResponse]
	WorkerDeploymentSetCurrentVersionResponse struct {
		// ConflictToken - Token to serialize Worker Deployment operations.
		ConflictToken []byte

		// PreviousVersion - The Version that was current before executing this operation, if any.
        //
		// Deprecated: in favor of API idempotency. Use `Describe` before this API to get the previous
		// state. Pass the `ConflictToken` returned by `Describe` to this API to avoid race conditions.
		PreviousVersion *WorkerDeploymentVersion
	}

	// WorkerDeploymentSetRampingVersionOptions provides options for
	// [WorkerDeploymentHandle.SetRampingVersion].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentSetRampingVersionOptions]
	WorkerDeploymentSetRampingVersionOptions struct {
		// BuildID - A Build ID within this deployment to set as the ramping version. If empty, the
		// current version will target unversioned workers.
		BuildID string

		// Percentage - Ramp percentage to set. Valid range: [0,100].
		Percentage float32

		// ConflictToken - Token to serialize Worker Deployment operations. Passing a non-empty
		// conflict token will cause this request to fail with
		// `serviceerror.FailedPrecondition` if the
		// Deployment's configuration has been modified between the API call that
		// generated the token and this one.
		// The current token can be obtained with [WorkerDeploymentHandle.Describe],
		// or returned by other successful Worker Deployment operations.
		//
		// Optional: defaults to empty token, which bypasses conflict detection.
		ConflictToken []byte

		// Identity - The identity of the client who initiated this request.
		//
		// Optional: defaults to the identity of the underlying workflow client.
		Identity string

		// IgnoreMissingTaskQueues - Override protection against accidental removal of Task Queues.
		// When false this request would be rejected if not all the expected Task Queues are
		// being polled by Workers in the new Version.
		// The set of expected Task Queues contains all the Task Queues that were ever polled by
		// the existing Current Version of the Deployment, with the following exclusions:
		//   - Task Queues that are no longer used, i.e., with empty backlog and no recently added tasks.
		//   - Task Queues moved to another Worker Deployment, i.e., current in a different Deployment.
		// WARNING: setting this flag could lead to missing Task Queues polled by late starting
		// Workers.
		//
		// Optional: default to reject request when queues are missing.
		IgnoreMissingTaskQueues bool

		// AllowNoPollers - Override protection against accidentally sending tasks to a version without pollers.
		// When false this request will be rejected if no pollers have been seen for the proposed Current Version,
		// in order to protect users from routing tasks to pollers that do not exist, leading to possible timeouts.
		// Pass `true` here to bypass this protection.
		// WARNING: setting this flag could lead to tasks being sent to a version that has no pollers.
		//
		// Optional: default to reject request when version has never had pollers.
		AllowNoPollers bool
	}

	// WorkerDeploymentSetRampingVersionResponse is the response for
	// [WorkerDeploymentHandle.SetRampingVersion].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentSetRampingVersionResponse]
	WorkerDeploymentSetRampingVersionResponse struct {
		// ConflictToken - Token to serialize Worker Deployment operations.
		ConflictToken []byte

		// PreviousVersion - The Ramping Version before executing this operation, if any.
		//
		// Deprecated: in favor of API idempotency. Use `Describe` before this API to get the previous
		// state. Pass the `ConflictToken` returned by `Describe` to this API to avoid race conditions.
		PreviousVersion *WorkerDeploymentVersion

		// PreviousPercentage - The Ramping Version Percentage before executing this operation.
		//
		// Deprecated: in favor of API idempotency. Use `Describe` before this API to get the previous
		// state. Pass the `ConflictToken` returned by `Describe` to this API to avoid race conditions.
		PreviousPercentage float32
	}

	// WorkerDeploymentSetManagerIdentityOptions provides options for
	// [WorkerDeploymentHandle.SetManagerIdentity].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentSetManagerIdentityOptions]
	WorkerDeploymentSetManagerIdentityOptions struct {
		// ManagerIdentity - string to set as the Worker Deployment's ManagerIdentity.
		// An empty string will clear the ManagerIdentity field.
		// It is invalid to set Self=true and ManagerIdentity != "".
		ManagerIdentity string

		// Self - If true, set the Worker Deployment's ManagerIdentity field to the identity
		// of the user submitting this request.
		// It is invalid to set Self=true and ManagerIdentity != "".
		Self bool

		// ConflictToken - Token to serialize Worker Deployment operations. Passing a non-empty
		// conflict token will cause this request to fail with
		// `serviceerror.FailedPrecondition` if the
		// Deployment's configuration has been modified between the API call that
		// generated the token and this one.
		// The current token can be obtained with [WorkerDeploymentHandle.Describe],
		// or returned by other successful Worker Deployment operations.
		//
		// Optional: defaults to empty token, which bypasses conflict detection.
		ConflictToken []byte

		// Identity - The identity of the client who initiated this request.
		//
		// Optional: defaults to the identity of the underlying workflow client.
		Identity string
	}

	// WorkerDeploymentSetManagerIdentityResponse is the response for
	// [WorkerDeploymentHandle.SetManagerIdentity].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentSetManagerIdentityResponse]
	WorkerDeploymentSetManagerIdentityResponse struct {
		// ConflictToken - Token to serialize Worker Deployment operations.
		ConflictToken []byte

		// PreviousManagerIdentity - The Manager Identity before executing this operation, if any.
		//
		// Deprecated: in favor of API idempotency. Use `Describe` before this API to get the previous
		// state. Pass the `ConflictToken` returned by `Describe` to this API to avoid race conditions.
		PreviousManagerIdentity string
	}

	// WorkerDeploymentDescribeVersionOptions provides options for
	// [WorkerDeploymentHandle.DescribeVersion].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentDescribeVersionOptions]
	WorkerDeploymentDescribeVersionOptions struct {
		// BuildID - A Build ID within this deployment to describe.
		BuildID string
	}

	// WorkerDeploymentTaskQueueInfo describes properties of the Task Queues involved
	// in a Deployment Version.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentTaskQueueInfo]
	WorkerDeploymentTaskQueueInfo struct {
		// Name - Task queue name.
		Name string

		// Type - The type of this task queue.
		Type TaskQueueType
	}

	// WorkerDeploymentVersionDrainageInfo describes drainage properties of a Deployment Version.
	// This enables users to safely decide when they can decommission a Version.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentVersionDrainageInfo]
	WorkerDeploymentVersionDrainageInfo struct {
		// DrainageStatus - The Worker Deployment Version drainage status to guarantee safe
		// decommission of this Version.
		DrainageStatus WorkerDeploymentVersionDrainageStatus

		// LastChangedTime - Last time the drainage status changed.
		LastChangedTime time.Time

		// LastCheckedTime - Last time the system checked for drainage of this version.
		// Note that drainage values may have refresh delays up to a few minutes.
		LastCheckedTime time.Time
	}

	// WorkerDeploymentVersionInfo provides information about a Worker Deployment Version.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentVersionInfo]
	WorkerDeploymentVersionInfo struct {
		// Version - A Deployment Version identifier.
		Version WorkerDeploymentVersion

		// CreateTime - When this Deployment Version was created.
		CreateTime time.Time

		// RoutingChangedTime - Last time the routing configuration of this Version changed.
		RoutingChangedTime time.Time

		// CurrentSinceTime - The time when this Version was set to Current. Zero if not Current.
		CurrentSinceTime time.Time

		// RampingSinceTime - The time when this Version started ramping. Zero if not ramping.
		RampingSinceTime time.Time

		// RampPercentage - Ramp percentage for this Version. Valid range [0, 100].
		RampPercentage float32

		// TaskQueuesInfos - List of task queues polled by workers in this Deployment Version.
		TaskQueuesInfos []WorkerDeploymentTaskQueueInfo

		// DrainageInfo - Drainage information for a Worker Deployment Version, enabling users to
		// decide when they can safely decommission this Version.
		//
		// Optional: not present when the version is Current or Ramping.
		DrainageInfo *WorkerDeploymentVersionDrainageInfo

		// Metadata - A user-defined set of key-values attached to this Version.
		Metadata map[string]*commonpb.Payload
	}

	// WorkerDeploymentVersionDescription is the response for
	// [WorkerDeploymentHandle.DescribeVersion].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentVersionDescription]
	WorkerDeploymentVersionDescription struct {
		// Info - Information about this Version.
		Info WorkerDeploymentVersionInfo
	}

	// WorkerDeploymentDeleteVersionOptions provides options for
	// [WorkerDeploymentHandle.DeleteVersion].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentDeleteVersionOptions]
	WorkerDeploymentDeleteVersionOptions struct {
		// BuildID - A Build ID within this deployment to delete.
		BuildID string

		// SkipDrainage - Force deletion even if the Version is still draining.
		//
		// Optional: default to always drain before deletion
		SkipDrainage bool

		// Identity - The identity of the client who initiated this request.
		//
		// Optional: defaults to the identity of the underlying workflow client.
		Identity string
	}

	// WorkerDeploymentDeleteVersionResponse is the response for
	// [WorkerDeploymentHandle.DeleteVersion].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentDeleteVersionResponse]
	WorkerDeploymentDeleteVersionResponse struct {
	}

	// WorkerDeploymentMetadataUpdate modifies user-defined metadata entries that describe
	// a Version.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentMetadataUpdate]
	WorkerDeploymentMetadataUpdate struct {
		// UpsertEntries - Metadata entries inserted or modified. When values are not
		// of type *commonpb.Payload, the client data converter will be used to generate
		// payloads.
		UpsertEntries map[string]interface{}

		// RemoveEntries - List of keys to remove from the metadata.
		RemoveEntries []string
	}

	// WorkerDeploymentUpdateVersionMetadataOptions provides options for
	// [WorkerDeploymentHandle.UpdateVersionMetadata].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentUpdateVersionMetadataOptions]
	WorkerDeploymentUpdateVersionMetadataOptions struct {
		// Version - the deployment version to target.
		Version WorkerDeploymentVersion

		// MetadataUpdate - Changes to the user-defined metadata entries for this Version.
		MetadataUpdate WorkerDeploymentMetadataUpdate
	}

	// WorkerDeploymentUpdateVersionMetadataResponse is the response for
	// [WorkerDeploymentHandle.UpdateVersionMetadata].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentUpdateVersionMetadataResponse]
	WorkerDeploymentUpdateVersionMetadataResponse struct {
		// Metadata - A user-defined set of key-values after the update.
		Metadata map[string]*commonpb.Payload
	}

	// WorkerDeploymentHandle is a handle to a Worker Deployment.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentHandle]
	WorkerDeploymentHandle interface {
		// Describe returns a description of this Worker Deployment.
		//
		// NOTE: Experimental
		Describe(ctx context.Context, options WorkerDeploymentDescribeOptions) (WorkerDeploymentDescribeResponse, error)

		// SetCurrentVersion changes the Current Version for this Worker Deployment.
		//
		// It also unsets the Ramping Version when it matches the Version being set as Current.
		//
		// NOTE: Experimental
		SetCurrentVersion(ctx context.Context, options WorkerDeploymentSetCurrentVersionOptions) (WorkerDeploymentSetCurrentVersionResponse, error)

		// SetRampingVersion changes the Ramping Version of this Worker Deployment and its ramp
		// percentage.
		//
		// NOTE: Experimental
		SetRampingVersion(ctx context.Context, options WorkerDeploymentSetRampingVersionOptions) (WorkerDeploymentSetRampingVersionResponse, error)

		// SetManagerIdentity changes the Manager Identity of this Worker Deployment.
		//
		// NOTE: Experimental
		SetManagerIdentity(ctx context.Context, options WorkerDeploymentSetManagerIdentityOptions) (WorkerDeploymentSetManagerIdentityResponse, error)

		// DescribeVersion gives a description of one the Versions in this Worker Deployment.
		//
		// NOTE: Experimental
		DescribeVersion(ctx context.Context, options WorkerDeploymentDescribeVersionOptions) (WorkerDeploymentVersionDescription, error)

		// DeleteVersion manually removes a Version. This is rarely needed during normal operation
		// since unused Versions are eventually garbage collected.
		// The client can delete a Version only when all of the following conditions are met:
		//  - It is not the Current or Ramping Version for this Deployment.
		//  - It has no active pollers, i.e., none of the task queues in the Version have pollers.
		//  - It is not draining. This requirement can be ignored with the option SkipDrainage.
		//
		// NOTE: Experimental
		DeleteVersion(ctx context.Context, options WorkerDeploymentDeleteVersionOptions) (WorkerDeploymentDeleteVersionResponse, error)

		// UpdateVersionMetadata changes the metadata associated with a Worker Version in this
		// Deployment.
		//
		// NOTE: Experimental
		UpdateVersionMetadata(ctx context.Context, options WorkerDeploymentUpdateVersionMetadataOptions) (WorkerDeploymentUpdateVersionMetadataResponse, error)
	}

	// DeploymentListOptions are the parameters for configuring listing Worker Deployments.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentListOptions]
	WorkerDeploymentListOptions struct {
		// PageSize - How many results to fetch from the Server at a time.
		//
		// Optional: defaulted to 1000
		PageSize int
	}

	// WorkerDeploymentRoutingConfig describes when new or existing Workflow Tasks are
	// executed with this Worker Deployment.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentRoutingConfig]
	WorkerDeploymentRoutingConfig struct {
		// CurrentVersion - Specifies which Deployment Version should receive new workflow
		// executions and tasks of existing unversioned or AutoUpgrade workflows.
		// If nil, all unversioned workers are the target.
		CurrentVersion *WorkerDeploymentVersion

		// RampingVersion - Specifies that some traffic is being shifted from the CurrentVersion
		// to this Version. RampingVersion should always be different from CurrentVersion.
		// If nil, all unversioned workers are the target, if the percentage is nonzero.
		//
		// Note that it is possible to ramp from one Version to another Version,
		// or from unversioned workers to a particular Version, or from a particular Version to
		// unversioned workers.
		RampingVersion *WorkerDeploymentVersion

		// RampingVersionPercentage - Percentage of tasks that are routed to the RampingVersion
		// instead of the Current Version.
		// Valid range: [0, 100]. A 100% value means the RampingVersion is receiving full
		// traffic but not yet "promoted" to be the CurrentVersion, likely due to pending
		// validations. A 0% value means ramping has been paused, or there is no ramping if
		// RampingVersion is missing.
		RampingVersionPercentage float32

		// CurrentVersionChangedTime - Last time the current version was changed.
		CurrentVersionChangedTime time.Time

		// RampingVersionChangedTime - Last time the ramping version was changed. Not updated if
		// only RampingVersionPercentage changes.
		RampingVersionChangedTime time.Time

		// RampingVersionPercentageChangedTime - Last time ramping version percentage was changed.
		// If RampingVersion has changed, this is also updated, even if the percentage remains the same.
		RampingVersionPercentageChangedTime time.Time
	}

	// WorkerDeploymentListEntry is a subset of fields from [WorkerDeploymentInfo].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentListEntry]
	WorkerDeploymentListEntry struct {
		// Name - The deployment name.
		Name string

		// CreateTime - When this deployment was created.
		CreateTime time.Time

		// RoutingConfig - When to execute new or existing Workflow Tasks with this Deployment.
		RoutingConfig WorkerDeploymentRoutingConfig
	}

	// WorkerDeploymentListIterator is an iterator for deployments.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentListIterator]
	WorkerDeploymentListIterator interface {
		// HasNext - Return whether this iterator has next value.
		HasNext() bool

		// Next - Returns the next Worker Deployment and error
		Next() (*WorkerDeploymentListEntry, error)
	}

	// WorkerDeploymentDeleteOptions provides options for [WorkerDeploymentClient.Delete].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentDeleteOptions]
	WorkerDeploymentDeleteOptions struct {
		// Name - The name of the deployment to be deleted.
		Name string

		// Identity - The identity of the client who initiated this request.
		//
		// Optional: defaults to the identity of the underlying workflow client.
		Identity string
	}

	// WorkerDeploymentDeleteResponse is the response for [WorkerDeploymentClient.Delete].
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentDeleteResponse]
	WorkerDeploymentDeleteResponse struct {
	}

	// WorkerDeploymentClient is the client that manages Worker Deployments.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.WorkerDeploymentClient]
	WorkerDeploymentClient interface {
		// List returns an iterator to enumerate Worker Deployments in the client's namespace.
		//
		// NOTE: Experimental
		List(ctx context.Context, options WorkerDeploymentListOptions) (WorkerDeploymentListIterator, error)

		// GetHandle returns a handle to a Worker Deployment.
		//
		// This method does not validate the Worker Deployment Name. If there is no deployment
		// with that name in this namespace, methods like WorkerDeploymentHandle.Describe()
		// will return an error.
		//
		// NOTE: Experimental
		GetHandle(name string) WorkerDeploymentHandle

		// Delete removes the records of a Worker Deployment. A Deployment can only be
		// deleted if it has no Version in it.
		//
		// NOTE: Experimental
		Delete(ctx context.Context, options WorkerDeploymentDeleteOptions) (WorkerDeploymentDeleteResponse, error)
	}
)
