// The MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
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

	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
)

// A stand-in for a Build Id for unversioned Workers.
const UnversionedBuildID = ""

// VersioningIntent indicates whether the user intends certain commands to be run on
// a compatible worker build ID version or not.
type VersioningIntent int

const (
	// VersioningIntentUnspecified indicates that the SDK should choose the most sensible default
	// behavior for the type of command, accounting for whether the command will be run on the same
	// task queue as the current worker.
	VersioningIntentUnspecified VersioningIntent = iota
	// VersioningIntentCompatible indicates that the command should run on a worker with compatible
	// version if possible. It may not be possible if the target task queue does not also have
	// knowledge of the current worker's build ID.
	//
	// Deprecated: This has the same effect as [VersioningIntentInheritBuildID], use that instead.
	VersioningIntentCompatible
	// VersioningIntentDefault indicates that the command should run on the target task queue's
	// current overall-default build ID.
	//
	// Deprecated: This has the same effect as [VersioningIntentUseAssignmentRules], use that instead.
	VersioningIntentDefault
	// VersioningIntentInheritBuildID indicates the command should inherit the current Build ID of the
	// Workflow triggering it, and not use Assignment Rules. (Redirect Rules are still applicable)
	// This is the default behavior for commands running on the same Task Queue as the current worker.
	VersioningIntentInheritBuildID
	// VersioningIntentUseAssignmentRules indicates the command should use the latest Assignment Rules
	// to select a Build ID independently of the workflow triggering it.
	// This is the default behavior for commands not running on the same Task Queue as the current worker.
	VersioningIntentUseAssignmentRules
)

// TaskReachability specifies which category of tasks may reach a worker on a versioned task queue.
// Used both in a reachability query and its response.
type TaskReachability int

const (
	// TaskReachabilityUnspecified indicates the reachability was not specified
	TaskReachabilityUnspecified = iota
	// TaskReachabilityNewWorkflows indicates the Build Id might be used by new workflows
	TaskReachabilityNewWorkflows
	// TaskReachabilityExistingWorkflows indicates the Build Id might be used by open workflows
	// and/or closed workflows.
	TaskReachabilityExistingWorkflows
	// TaskReachabilityOpenWorkflows indicates the Build Id might be used by open workflows.
	TaskReachabilityOpenWorkflows
	// TaskReachabilityClosedWorkflows indicates the Build Id might be used by closed workflows
	TaskReachabilityClosedWorkflows
)

type (
	// UpdateWorkerBuildIdCompatibilityOptions is the input to
	// Client.UpdateWorkerBuildIdCompatibility.
	UpdateWorkerBuildIdCompatibilityOptions struct {
		// The task queue to update the version sets of.
		TaskQueue string
		Operation UpdateBuildIDOp
	}

	// UpdateBuildIDOp is an interface for the different operations that can be
	// performed when updating the worker build ID compatibility sets for a task queue.
	//
	// Possible operations are:
	//   - BuildIDOpAddNewIDInNewDefaultSet
	//   - BuildIDOpAddNewCompatibleVersion
	//   - BuildIDOpPromoteSet
	//   - BuildIDOpPromoteIDWithinSet
	UpdateBuildIDOp interface {
		targetedBuildId() string
	}
	BuildIDOpAddNewIDInNewDefaultSet struct {
		BuildID string
	}
	BuildIDOpAddNewCompatibleVersion struct {
		BuildID                   string
		ExistingCompatibleBuildID string
		MakeSetDefault            bool
	}
	BuildIDOpPromoteSet struct {
		BuildID string
	}
	BuildIDOpPromoteIDWithinSet struct {
		BuildID string
	}
)

// Validates and converts the user's options into the proto request. Namespace must be attached afterward.
func (uw *UpdateWorkerBuildIdCompatibilityOptions) validateAndConvertToProto() (*workflowservice.UpdateWorkerBuildIdCompatibilityRequest, error) {
	if uw.TaskQueue == "" {
		return nil, errors.New("missing TaskQueue field")
	}
	if uw.Operation.targetedBuildId() == "" {
		return nil, errors.New("missing Operation BuildID field")
	}
	req := &workflowservice.UpdateWorkerBuildIdCompatibilityRequest{
		TaskQueue: uw.TaskQueue,
	}

	switch v := uw.Operation.(type) {
	case *BuildIDOpAddNewIDInNewDefaultSet:
		req.Operation = &workflowservice.UpdateWorkerBuildIdCompatibilityRequest_AddNewBuildIdInNewDefaultSet{
			AddNewBuildIdInNewDefaultSet: v.BuildID,
		}

	case *BuildIDOpAddNewCompatibleVersion:
		if v.ExistingCompatibleBuildID == "" {
			return nil, errors.New("missing ExistingCompatibleBuildID")
		}
		req.Operation = &workflowservice.UpdateWorkerBuildIdCompatibilityRequest_AddNewCompatibleBuildId{
			AddNewCompatibleBuildId: &workflowservice.UpdateWorkerBuildIdCompatibilityRequest_AddNewCompatibleVersion{
				NewBuildId:                v.BuildID,
				ExistingCompatibleBuildId: v.ExistingCompatibleBuildID,
				MakeSetDefault:            v.MakeSetDefault,
			},
		}
	case *BuildIDOpPromoteSet:
		req.Operation = &workflowservice.UpdateWorkerBuildIdCompatibilityRequest_PromoteSetByBuildId{
			PromoteSetByBuildId: v.BuildID,
		}
	case *BuildIDOpPromoteIDWithinSet:
		req.Operation = &workflowservice.UpdateWorkerBuildIdCompatibilityRequest_PromoteBuildIdWithinSet{
			PromoteBuildIdWithinSet: v.BuildID,
		}
	}

	return req, nil
}

type GetWorkerBuildIdCompatibilityOptions struct {
	TaskQueue string
	MaxSets   int
}

type GetWorkerTaskReachabilityOptions struct {
	// BuildIDs - The build IDs to query the reachability of. At least one build ID must be provided.
	BuildIDs []string
	// TaskQueues - The task queues with Build IDs defined on them that the request is
	// concerned with.
	// Optional: defaults to all task queues
	TaskQueues []string
	// Reachability - The reachability this request is concerned with.
	// Optional: defaults to all types of reachability
	Reachability TaskReachability
}

type WorkerTaskReachability struct {
	// BuildIDReachability - map of build IDs and their reachability information
	// May contain an entry with UnversionedBuildID for an unversioned worker
	BuildIDReachability map[string]*BuildIDReachability
}

type BuildIDReachability struct {
	// TaskQueueReachable map of task queues and their reachability information.
	TaskQueueReachable map[string]*TaskQueueReachability
	// UnretrievedTaskQueues is a list of task queues not retrieved because the server limits
	// the number that can be queried at once.
	UnretrievedTaskQueues []string
}

type TaskQueueReachability struct {
	// TaskQueueReachability for a worker in a single task queue.
	// If TaskQueueReachability is empty, this worker is considered unreachable in this task queue.
	TaskQueueReachability []TaskReachability
}

// WorkerBuildIDVersionSets is the response for Client.GetWorkerBuildIdCompatibility and represents the sets
// of worker build id based versions.
type WorkerBuildIDVersionSets struct {
	Sets []*CompatibleVersionSet
}

// Default returns the current overall default version. IE: The one that will be used to start new workflows.
// Returns the empty string if there are no versions present.
func (s *WorkerBuildIDVersionSets) Default() string {
	if len(s.Sets) == 0 {
		return ""
	}
	lastSet := s.Sets[len(s.Sets)-1]
	if len(lastSet.BuildIDs) == 0 {
		return ""
	}
	return lastSet.BuildIDs[len(lastSet.BuildIDs)-1]
}

// CompatibleVersionSet represents a set of worker build ids which are compatible with each other.
type CompatibleVersionSet struct {
	BuildIDs []string
}

func workerVersionSetsFromProtoResponse(response *workflowservice.GetWorkerBuildIdCompatibilityResponse) *WorkerBuildIDVersionSets {
	if response == nil {
		return nil
	}
	return &WorkerBuildIDVersionSets{
		Sets: workerVersionSetsFromProto(response.GetMajorVersionSets()),
	}
}

func workerVersionSetsFromProto(sets []*taskqueuepb.CompatibleVersionSet) []*CompatibleVersionSet {
	if sets == nil {
		return nil
	}
	result := make([]*CompatibleVersionSet, len(sets))
	for i, s := range sets {
		result[i] = &CompatibleVersionSet{
			BuildIDs: s.GetBuildIds(),
		}
	}
	return result
}

func workerTaskReachabilityFromProtoResponse(response *workflowservice.GetWorkerTaskReachabilityResponse) *WorkerTaskReachability {
	if response == nil {
		return nil
	}
	return &WorkerTaskReachability{
		BuildIDReachability: buildIDReachabilityFromProto(response.GetBuildIdReachability()),
	}
}

func buildIDReachabilityFromProto(sets []*taskqueuepb.BuildIdReachability) map[string]*BuildIDReachability {
	if sets == nil {
		return nil
	}
	result := make(map[string]*BuildIDReachability, len(sets))
	for _, s := range sets {
		retrievedTaskQueues, unretrievedTaskQueues := taskQueueReachabilityFromProto(s.GetTaskQueueReachability())
		result[s.GetBuildId()] = &BuildIDReachability{
			TaskQueueReachable:    retrievedTaskQueues,
			UnretrievedTaskQueues: unretrievedTaskQueues,
		}
	}
	return result
}

func taskQueueReachabilityFromProto(sets []*taskqueuepb.TaskQueueReachability) (map[string]*TaskQueueReachability, []string) {
	if sets == nil {
		return nil, nil
	}
	retrievedTaskQueues := make(map[string]*TaskQueueReachability, len(sets))
	unretrievedTaskQueues := make([]string, 0, len(sets))
	for _, s := range sets {
		reachability := make([]TaskReachability, len(s.GetReachability()))
		for i, r := range s.GetReachability() {
			reachability[i] = taskReachabilityFromProto(r)
		}
		if len(reachability) == 1 && reachability[0] == TaskReachabilityUnspecified {
			unretrievedTaskQueues = append(unretrievedTaskQueues, s.GetTaskQueue())
		} else {
			retrievedTaskQueues[s.GetTaskQueue()] = &TaskQueueReachability{
				TaskQueueReachability: reachability,
			}
		}

	}
	return retrievedTaskQueues, unretrievedTaskQueues
}

func taskReachabilityToProto(r TaskReachability) enumspb.TaskReachability {
	switch r {
	case TaskReachabilityUnspecified:
		return enumspb.TASK_REACHABILITY_UNSPECIFIED
	case TaskReachabilityNewWorkflows:
		return enumspb.TASK_REACHABILITY_NEW_WORKFLOWS
	case TaskReachabilityExistingWorkflows:
		return enumspb.TASK_REACHABILITY_EXISTING_WORKFLOWS
	case TaskReachabilityOpenWorkflows:
		return enumspb.TASK_REACHABILITY_OPEN_WORKFLOWS
	case TaskReachabilityClosedWorkflows:
		return enumspb.TASK_REACHABILITY_CLOSED_WORKFLOWS
	default:
		panic("unknown task reachability")

	}
}

func taskReachabilityFromProto(r enumspb.TaskReachability) TaskReachability {
	switch r {
	case enumspb.TASK_REACHABILITY_UNSPECIFIED:
		return TaskReachabilityUnspecified
	case enumspb.TASK_REACHABILITY_NEW_WORKFLOWS:
		return TaskReachabilityNewWorkflows
	case enumspb.TASK_REACHABILITY_EXISTING_WORKFLOWS:
		return TaskReachabilityExistingWorkflows
	case enumspb.TASK_REACHABILITY_OPEN_WORKFLOWS:
		return TaskReachabilityOpenWorkflows
	case enumspb.TASK_REACHABILITY_CLOSED_WORKFLOWS:
		return TaskReachabilityClosedWorkflows
	default:
		panic("unknown task reachability")
	}
}

func (v *BuildIDOpAddNewIDInNewDefaultSet) targetedBuildId() string { return v.BuildID }
func (v *BuildIDOpAddNewCompatibleVersion) targetedBuildId() string { return v.BuildID }
func (v *BuildIDOpPromoteSet) targetedBuildId() string              { return v.BuildID }
func (v *BuildIDOpPromoteIDWithinSet) targetedBuildId() string      { return v.BuildID }

// Helper to determine if how the `InheritBuildId` flag for a command should be set based on
// the user's intent and whether the target task queue matches this worker's task queue.
func determineInheritBuildIdFlagForCommand(intent VersioningIntent, workerTq, TargetTq string) bool {
	inheritBuildId := true
	if intent == VersioningIntentDefault || intent == VersioningIntentUseAssignmentRules {
		inheritBuildId = false
	} else if intent == VersioningIntentUnspecified {
		// If the target task queue doesn't match ours, use the default version. Empty target counts
		// as matching.
		if TargetTq != "" && workerTq != TargetTq {
			inheritBuildId = false
		}
	}
	return inheritBuildId
}
