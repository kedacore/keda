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

	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
)

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
	VersioningIntentCompatible
	// VersioningIntentDefault indicates that the command should run on the target task queue's
	// current overall-default build ID.
	VersioningIntentDefault
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
		ExistingCompatibleBuildId string
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
		if v.ExistingCompatibleBuildId == "" {
			return nil, errors.New("missing ExistingCompatibleBuildId")
		}
		req.Operation = &workflowservice.UpdateWorkerBuildIdCompatibilityRequest_AddNewCompatibleBuildId{
			AddNewCompatibleBuildId: &workflowservice.UpdateWorkerBuildIdCompatibilityRequest_AddNewCompatibleVersion{
				NewBuildId:                v.BuildID,
				ExistingCompatibleBuildId: v.ExistingCompatibleBuildId,
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

func (v *BuildIDOpAddNewIDInNewDefaultSet) targetedBuildId() string { return v.BuildID }
func (v *BuildIDOpAddNewCompatibleVersion) targetedBuildId() string { return v.BuildID }
func (v *BuildIDOpPromoteSet) targetedBuildId() string              { return v.BuildID }
func (v *BuildIDOpPromoteIDWithinSet) targetedBuildId() string      { return v.BuildID }

// Helper to determine if how the `UseCompatibleVersion` flag for a command should be set based on
// the user's intent and whether the target task queue matches this worker's task queue.
func determineUseCompatibleFlagForCommand(intent VersioningIntent, workerTq, TargetTq string) bool {
	useCompat := true
	if intent == VersioningIntentDefault {
		useCompat = false
	} else if intent == VersioningIntentUnspecified {
		// If the target task queue doesn't match ours, use the default version
		if workerTq != TargetTq {
			useCompat = false
		}
	}
	return useCompat
}
