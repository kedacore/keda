/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package traffic

import (
	"fmt"

	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// TargetError gives details about an invalid traffic target.
type TargetError interface {
	error

	// MarkBadTrafficTarget marks a RouteStatus with Condition corresponding
	// to the error case of the traffic target.
	MarkBadTrafficTarget(rs *v1alpha1.RouteStatus)

	// IsFailure returns whether a TargetError is a true failure, e.g.
	// a Configuration fails to become ready.
	IsFailure() bool
}

type missingTargetError struct {
	kind string // Kind of the traffic target, e.g. Configuration/Revision.
	name string // Name of the traffic target.
}

var _ TargetError = (*missingTargetError)(nil)

// Error implements error.
func (e *missingTargetError) Error() string {
	return fmt.Sprintf("%v %q referenced in traffic not found", e.kind, e.name)
}

// MarkBadTrafficTarget implements TargetError.
func (e *missingTargetError) MarkBadTrafficTarget(rs *v1alpha1.RouteStatus) {
	rs.MarkMissingTrafficTarget(e.kind, e.name)
}

// IsFailure implements TargetError.
func (e *missingTargetError) IsFailure() bool {
	return true
}

type unreadyConfigError struct {
	name      string // Name of the config that isn't ready.
	isFailure bool   // True iff target fails to get ready.
}

var _ TargetError = (*unreadyConfigError)(nil)

// Error implements error.
func (e *unreadyConfigError) Error() string {
	return fmt.Sprintf("Configuration '%q' not ready, isFailure=%t", e.name, e.isFailure)
}

// MarkBadTrafficTarget implements TargetError.
func (e *unreadyConfigError) MarkBadTrafficTarget(rs *v1alpha1.RouteStatus) {
	if e.IsFailure() {
		rs.MarkConfigurationFailed(e.name)
	} else {
		rs.MarkConfigurationNotReady(e.name)
	}
}

func (e *unreadyConfigError) IsFailure() bool {
	return e.isFailure
}

type unreadyRevisionError struct {
	name      string // Name of the config that isn't ready.
	isFailure bool   // True iff the Revision fails to become ready.
}

var _ TargetError = (*unreadyRevisionError)(nil)

// Error implements error.
func (e *unreadyRevisionError) Error() string {
	return fmt.Sprintf("Revision %q not ready, isFailure=%t", e.name, e.isFailure)
}

// MarkBadTrafficTarget implements TargetError.
func (e *unreadyRevisionError) MarkBadTrafficTarget(rs *v1alpha1.RouteStatus) {
	if e.IsFailure() {
		rs.MarkRevisionFailed(e.name)
	} else {
		rs.MarkRevisionNotReady(e.name)
	}
}

func (e *unreadyRevisionError) IsFailure() bool {
	return e.isFailure
}

// errUnreadyConfiguration returns a TargetError for a Configuration that is not ready.
func errUnreadyConfiguration(config *v1alpha1.Configuration) TargetError {
	status := corev1.ConditionUnknown
	if c := config.Status.GetCondition(v1alpha1.ConfigurationConditionReady); c != nil {
		status = c.Status
	}
	return &unreadyConfigError{
		name:      config.Name,
		isFailure: status == corev1.ConditionFalse,
	}
}

// errUnreadyRevision returns a TargetError for a Revision that is not ready.
func errUnreadyRevision(rev *v1alpha1.Revision) TargetError {
	status := corev1.ConditionUnknown
	if c := rev.Status.GetCondition(v1alpha1.RevisionConditionReady); c != nil {
		status = c.Status
	}
	return &unreadyRevisionError{
		name:      rev.Name,
		isFailure: status == corev1.ConditionFalse,
	}
}

// errMissingConfiguration returns a TargetError for a Configuration what does not exist.
func errMissingConfiguration(name string) TargetError {
	return &missingTargetError{
		kind: "Configuration",
		name: name,
	}
}

// errMissingRevision returns a TargetError for a Revision that does not exist.
func errMissingRevision(name string) TargetError {
	return &missingTargetError{
		kind: "Revision",
		name: name,
	}
}
