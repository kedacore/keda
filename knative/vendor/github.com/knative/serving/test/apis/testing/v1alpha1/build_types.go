/*
Copyright 2018 The Knative Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Build is a simple test build resource.
type Build struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Build (from the client).
	// +optional
	Spec BuildSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the Build (from the controller).
	// +optional
	Status BuildStatus `json:"status,omitempty"`
}

// Check that ConfigurationStatus may have its conditions managed.
var _ duckv1alpha1.ConditionsAccessor = (*BuildStatus)(nil)

// BuildSpec holds the desired state of the Build (from the client).
type BuildSpec struct {
	Failure *FailureInfo `json:"failure,omitempty"`
}

type FailureInfo struct {
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

const (
	// BuildConditionSucceeded is set when the revision is starting to materialize
	// runtime resources, and becomes true when those resources are ready.
	BuildConditionSucceeded = duckv1alpha1.ConditionSucceeded
)

var podCondSet = duckv1alpha1.NewBatchConditionSet()

// BuildStatus communicates the observed state of the Build (from the controller).
type BuildStatus struct {
	// Conditions communicates information about ongoing/complete
	// reconciliation processes that bring the "spec" inline with the observed
	// state of the world.
	// +optional
	Conditions duckv1alpha1.Conditions `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildList is a list of Build resources
type BuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Build `json:"items"`
}

// IsReady looks at the conditions and if the Status has a condition
// BuildConditionSucceeded returns true if ConditionStatus is True
func (rs *BuildStatus) IsReady() bool {
	return podCondSet.Manage(rs).IsHappy()
}

func (rs *BuildStatus) GetCondition(t duckv1alpha1.ConditionType) *duckv1alpha1.Condition {
	return podCondSet.Manage(rs).GetCondition(t)
}

func (rs *BuildStatus) InitializeConditions() {
	podCondSet.Manage(rs).InitializeConditions()
}

func (rs *BuildStatus) MarkDone() {
	podCondSet.Manage(rs).MarkTrue(BuildConditionSucceeded)
}

func (rs *BuildStatus) MarkFailure(fi *FailureInfo) {
	podCondSet.Manage(rs).MarkFalse(BuildConditionSucceeded, fi.Reason, fi.Message)
}

// GetConditions returns the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (rs *BuildStatus) GetConditions() duckv1alpha1.Conditions {
	return rs.Conditions
}

// SetConditions sets the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (rs *BuildStatus) SetConditions(conditions duckv1alpha1.Conditions) {
	rs.Conditions = conditions
}
