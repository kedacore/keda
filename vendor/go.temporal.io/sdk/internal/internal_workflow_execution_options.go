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
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type (
	// UpdateWorkflowExecutionOptionsRequest is a request for [Client.UpdateWorkflowExecutionOptions].
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
	// NOTE: Experimental
	WorkflowExecutionOptions struct {
		// If set, it takes precedence over the Versioning Behavior provided with code annotations.
		VersioningOverride VersioningOverride
	}

	// WorkflowExecutionOptionsChanges describes changes to the options of a workflow execution in [WorkflowExecutionOptions].
	// An entry with a `nil` pointer means do not change.
	// An entry with a pointer to an empty value means delete the entry, i.e., the empty value is a tombstone.
	// An entry with a pointer to a non-empty value means replace the entry, i.e., there is no deep merging.
	// NOTE: Experimental
	WorkflowExecutionOptionsChanges struct {
		VersioningOverride *VersioningOverride
	}

	// VersioningOverride changes the versioning configuration of a specific workflow execution.
	// If set, it takes precedence over the Versioning Behavior provided with workflow type registration or
	// default worker options.
	// To remove the override, the [UpdateWorkflowExecutionOptionsRequest] should include a pointer to
	// an empty [VersioningOverride] value in [WorkflowExecutionOptionsChanges].
	// See [WorkflowExecutionOptionsChanges] for details.
	// NOTE: Experimental
	VersioningOverride struct {
		// Behavior - The new Versioning Behavior. This field is required.
		Behavior VersioningBehavior
		// Identifies the Build ID and Deployment Series Name to pin the workflow to. Ignored when Behavior is not
		// [VersioningBehaviorPinned].
		//
		// Deprecated: Use [PinnedVersion]
		Deployment Deployment
		// PinnedVersion - Identifies the Worker Deployment Version to pin the workflow to, using the format
		// "<deployment_name>.<build_id>".
		// Required if behavior is [VersioningBehaviorPinned]. Must be absent if behavior is not [VersioningBehaviorPinned].
		PinnedVersion string
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

func workerDeploymentToProto(d Deployment) *deploymentpb.Deployment {
	// Server 1.26.2 requires a nil Deployment pointer, and not just a pointer to an empty Deployment,
	// to indicate that there is no Deployment.
	// It is a server error to override versioning behavior to AutoUpgrade while providing a Deployment,
	// and we need to replace it by nil. See https://github.com/temporalio/sdk-go/issues/1764.
	//
	// Future server versions may relax this requirement.
	if (Deployment{}) == d {
		return nil
	}
	return &deploymentpb.Deployment{
		SeriesName: d.SeriesName,
		BuildId:    d.BuildID,
	}
}

func versioningOverrideToProto(versioningOverride VersioningOverride) *workflowpb.VersioningOverride {
	if (VersioningOverride{}) == versioningOverride {
		return nil
	}
	return &workflowpb.VersioningOverride{
		Behavior:      versioningBehaviorToProto(versioningOverride.Behavior),
		Deployment:    workerDeploymentToProto(versioningOverride.Deployment),
		PinnedVersion: versioningOverride.PinnedVersion,
	}
}

func versioningOverrideFromProto(versioningOverride *workflowpb.VersioningOverride) VersioningOverride {
	if versioningOverride == nil {
		return VersioningOverride{}
	}

	return VersioningOverride{
		Behavior: VersioningBehavior(versioningOverride.GetBehavior()),
		Deployment: Deployment{
			//lint:ignore SA1019 ignore deprecated versioning APIs
			SeriesName: versioningOverride.GetDeployment().GetSeriesName(),
			//lint:ignore SA1019 ignore deprecated versioning APIs
			BuildID: versioningOverride.GetDeployment().GetBuildId(),
		},
		PinnedVersion: versioningOverride.GetPinnedVersion(),
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
		options.VersioningOverride = *changes.VersioningOverride
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
