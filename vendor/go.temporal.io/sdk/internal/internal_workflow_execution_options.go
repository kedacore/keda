package internal

import (
	"errors"
	"fmt"

	deploymentpb "go.temporal.io/api/deployment/v1"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type (
	// UpdateWorkflowExecutionOptionsRequest is a request for [Client.UpdateWorkflowExecutionOptions].
	//
	// NOTE: Experimental
	UpdateWorkflowExecutionOptionsRequest struct {
		// ID of the workflow.
		WorkflowId string
		// Running execution for a workflow ID. If empty string then it will pick the last running execution.
		RunId string
		// WorkflowExecutionOptionsChanges specifies changes to the options of a workflow execution.
		WorkflowExecutionOptionsChanges WorkflowExecutionOptionsChanges
	}

	// WorkflowExecutionOptions describes options for a workflow execution.
	//
	// NOTE: Experimental
	WorkflowExecutionOptions struct {
		// If set, it takes precedence over the Versioning Behavior provided with code annotations.
		VersioningOverride VersioningOverride
	}

	// WorkflowExecutionOptionsChanges describes changes to the options of a workflow execution in
	// [WorkflowExecutionOptions]. An entry with a `nil` pointer means do not change.
	//
	// NOTE: Experimental
	WorkflowExecutionOptionsChanges struct {
		// If non-nil, change the versioning override.
		VersioningOverride *VersioningOverrideChange
	}

	// VersioningOverrideChange sets or removes a versioning override when used with
	// [WorkflowExecutionOptionsChanges].
	//
	// NOTE: Experimental
	VersioningOverrideChange struct {
		// Set the override entry if non-nil. If nil, remove any previously set override.
		Value VersioningOverride
	}

	// VersioningOverride changes the versioning configuration of a specific workflow execution.
	// If set, it takes precedence over the Versioning Behavior provided with workflow type
	// registration or default worker options.
	//
	// To remove the override, the [UpdateWorkflowExecutionOptionsRequest] should include a pointer
	// to an empty [VersioningOverride] value in [WorkflowExecutionOptionsChanges]. See
	// [WorkflowExecutionOptionsChanges] for details.
	//
	// NOTE: Experimental
	VersioningOverride interface {
		behavior() VersioningBehavior
	}

	// PinnedVersioningOverride means the workflow will be pinned to a specific deployment version.
	//
	// NOTE: Experimental
	PinnedVersioningOverride struct {
		Version WorkerDeploymentVersion
	}

	// AutoUpgradeVersioningOverride means the workflow will auto-upgrade to the current deployment
	// version on the next workflow task.
	//
	// NOTE: Experimental
	AutoUpgradeVersioningOverride struct {
	}

	// OnConflictOptions specifies the actions to be taken when using the workflow ID conflict policy
	// USE_EXISTING.
	//
	// NOTE: Experimental
	OnConflictOptions struct {
		AttachRequestID           bool
		AttachCompletionCallbacks bool
		AttachLinks               bool
	}
)

func (*PinnedVersioningOverride) behavior() VersioningBehavior {
	return VersioningBehaviorPinned
}

func (*AutoUpgradeVersioningOverride) behavior() VersioningBehavior {
	return VersioningBehaviorAutoUpgrade
}

// Mapping WorkflowExecutionOptions field names to proto ones.
var workflowExecutionOptionsMap map[string]string = map[string]string{
	"VersioningOverride": "versioning_override",
}

func generateWorkflowExecutionOptionsPaths(mask []string) []string {
	var result []string
	for _, field := range mask {
		val, ok := workflowExecutionOptionsMap[field]
		if !ok {
			panic(fmt.Sprintf("invalid UpdatedFields entry %s not a field in WorkflowExecutionOptions", field))
		}
		result = append(result, val)
	}
	return result
}

func workflowExecutionOptionsMaskToProto(mask []string) *fieldmaskpb.FieldMask {
	paths := generateWorkflowExecutionOptionsPaths(mask)
	var workflowExecutionOptions *workflowpb.WorkflowExecutionOptions
	protoMask, err := fieldmaskpb.New(workflowExecutionOptions, paths...)
	if err != nil {
		panic("invalid field mask for WorkflowExecutionOptions")
	}
	return protoMask
}

func versioningOverrideToProto(versioningOverride VersioningOverride) *workflowpb.VersioningOverride {
	if versioningOverride == nil {
		return nil
	}
	behavior := versioningOverride.behavior()
	switch v := versioningOverride.(type) {
	case *PinnedVersioningOverride:
		return &workflowpb.VersioningOverride{
			Behavior:      versioningBehaviorToProto(behavior),
			PinnedVersion: v.Version.toCanonicalString(),
			Deployment: &deploymentpb.Deployment{
				SeriesName: v.Version.DeploymentName,
				BuildId:    v.Version.BuildID,
			},
			Override: &workflowpb.VersioningOverride_Pinned{
				Pinned: &workflowpb.VersioningOverride_PinnedOverride{
					Behavior: workflowpb.VersioningOverride_PINNED_OVERRIDE_BEHAVIOR_PINNED,
					Version:  v.Version.toProto(),
				},
			},
		}
	case *AutoUpgradeVersioningOverride:
		return &workflowpb.VersioningOverride{
			Behavior: versioningBehaviorToProto(behavior),
			Override: &workflowpb.VersioningOverride_AutoUpgrade{AutoUpgrade: true},
		}
	default:
		return nil
	}
}

func versioningOverrideFromProto(versioningOverride *workflowpb.VersioningOverride) VersioningOverride {
	if versioningOverride == nil {
		return nil
	}

	if versioningOverride.Override != nil {
		switch ot := versioningOverride.Override.(type) {
		case *workflowpb.VersioningOverride_AutoUpgrade:
			return &AutoUpgradeVersioningOverride{}
		case *workflowpb.VersioningOverride_Pinned:
			return &PinnedVersioningOverride{
				Version: workerDeploymentVersionFromProto(ot.Pinned.Version),
			}
		}
	}

	//lint:ignore SA1019 ignore deprecated versioning APIs
	behavior := versioningOverride.GetBehavior()
	switch behavior {
	case enumspb.VERSIONING_BEHAVIOR_AUTO_UPGRADE:
		return &AutoUpgradeVersioningOverride{}
	case enumspb.VERSIONING_BEHAVIOR_PINNED:
		//lint:ignore SA1019 ignore deprecated versioning APIs
		if versioningOverride.PinnedVersion != "" {
			return &PinnedVersioningOverride{
				//lint:ignore SA1019 ignore deprecated versioning APIs
				Version: *workerDeploymentVersionFromString(versioningOverride.PinnedVersion),
			}
		}
		return &PinnedVersioningOverride{
			Version: WorkerDeploymentVersion{
				//lint:ignore SA1019 ignore deprecated versioning APIs
				DeploymentName: versioningOverride.GetDeployment().SeriesName,
				//lint:ignore SA1019 ignore deprecated versioning APIs
				BuildID: versioningOverride.GetDeployment().BuildId,
			},
		}
	default:
		return nil
	}
}

func workflowExecutionOptionsToProto(options WorkflowExecutionOptions) *workflowpb.WorkflowExecutionOptions {
	return &workflowpb.WorkflowExecutionOptions{
		VersioningOverride: versioningOverrideToProto(options.VersioningOverride),
	}
}

func workflowExecutionOptionsChangesToProto(changes WorkflowExecutionOptionsChanges) (*workflowpb.WorkflowExecutionOptions, *fieldmaskpb.FieldMask) {
	mask := []string{}
	options := WorkflowExecutionOptions{}
	if changes.VersioningOverride != nil {
		mask = append(mask, "VersioningOverride")
		options.VersioningOverride = changes.VersioningOverride.Value
	}
	return workflowExecutionOptionsToProto(options), workflowExecutionOptionsMaskToProto(mask)
}

func workflowExecutionOptionsFromProtoUpdateResponse(response *workflowservice.UpdateWorkflowExecutionOptionsResponse) WorkflowExecutionOptions {
	if response == nil {
		return WorkflowExecutionOptions{}
	}

	versioningOverride := response.GetWorkflowExecutionOptions().GetVersioningOverride()

	return WorkflowExecutionOptions{
		VersioningOverride: versioningOverrideFromProto(versioningOverride),
	}
}

func (r *UpdateWorkflowExecutionOptionsRequest) validateAndConvertToProto(namespace string) (*workflowservice.UpdateWorkflowExecutionOptionsRequest, error) {
	if namespace == "" {
		return nil, errors.New("missing namespace argument")
	}

	if r.WorkflowId == "" {
		return nil, errors.New("missing workflow id argument")
	}

	if r.WorkflowExecutionOptionsChanges.VersioningOverride == nil {
		return nil, errors.New("update with no changes")
	}

	workflowExecutionOptions, updateMask := workflowExecutionOptionsChangesToProto(r.WorkflowExecutionOptionsChanges)

	requestMsg := &workflowservice.UpdateWorkflowExecutionOptionsRequest{
		Namespace: namespace,
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: r.WorkflowId,
			RunId:      r.RunId,
		},
		WorkflowExecutionOptions: workflowExecutionOptions,
		UpdateMask:               updateMask,
	}

	return requestMsg, nil
}

func (o *OnConflictOptions) ToProto() *workflowpb.OnConflictOptions {
	if o == nil {
		return nil
	}
	return &workflowpb.OnConflictOptions{
		AttachRequestId:           o.AttachRequestID,
		AttachCompletionCallbacks: o.AttachCompletionCallbacks,
		AttachLinks:               o.AttachLinks,
	}
}
