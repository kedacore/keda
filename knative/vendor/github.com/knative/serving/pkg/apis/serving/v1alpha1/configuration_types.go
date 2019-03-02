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
	"github.com/knative/pkg/apis"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/kmeta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration represents the "floating HEAD" of a linear history of Revisions,
// and optionally how the containers those revisions reference are built.
// Users create new Revisions by updating the Configuration's spec.
// The "latest created" revision's name is available under status, as is the
// "latest ready" revision's name.
// See also: https://github.com/knative/serving/blob/master/docs/spec/overview.md#configuration
type Configuration struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Configuration (from the client).
	// +optional
	Spec ConfigurationSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the Configuration (from the controller).
	// +optional
	Status ConfigurationStatus `json:"status,omitempty"`
}

// Check that Configuration may be validated and defaulted.
var _ apis.Validatable = (*Configuration)(nil)
var _ apis.Defaultable = (*Configuration)(nil)

// Check that we can create OwnerReferences to a Configuration.
var _ kmeta.OwnerRefable = (*Configuration)(nil)

// Check that ConfigurationStatus may have its conditions managed.
var _ duckv1alpha1.ConditionsAccessor = (*ConfigurationStatus)(nil)

// ConfigurationSpec holds the desired state of the Configuration (from the client).
type ConfigurationSpec struct {
	// DeprecatedGeneration was used prior in Kubernetes versions <1.11
	// when metadata.generation was not being incremented by the api server
	//
	// This property will be dropped in future Knative releases and should
	// not be used - use metadata.generation
	//
	// Tracking issue: https://github.com/knative/serving/issues/643
	//
	// +optional
	DeprecatedGeneration int64 `json:"generation,omitempty"`

	// Build optionally holds the specification for the build to
	// perform to produce the Revision's container image.
	// +optional
	Build *RawExtension `json:"build,omitempty"`

	// RevisionTemplate holds the latest specification for the Revision to
	// be stamped out. If a Build specification is provided, then the
	// RevisionTemplate's BuildName field will be populated with the name of
	// the Build object created to produce the container for the Revision.
	// +optional
	RevisionTemplate RevisionTemplateSpec `json:"revisionTemplate"`
}

const (
	// ConfigurationConditionReady is set when the configuration's latest
	// underlying revision has reported readiness.
	ConfigurationConditionReady = duckv1alpha1.ConditionReady
)

var confCondSet = duckv1alpha1.NewLivingConditionSet()

// ConfigurationStatus communicates the observed state of the Configuration (from the controller).
type ConfigurationStatus struct {
	// Conditions communicates information about ongoing/complete
	// reconciliation processes that bring the "spec" inline with the observed
	// state of the world.
	// +optional
	Conditions duckv1alpha1.Conditions `json:"conditions,omitempty"`

	// LatestReadyRevisionName holds the name of the latest Revision stamped out
	// from this Configuration that has had its "Ready" condition become "True".
	// +optional
	LatestReadyRevisionName string `json:"latestReadyRevisionName,omitempty"`

	// LatestCreatedRevisionName is the last revision that was created from this
	// Configuration. It might not be ready yet, for that use LatestReadyRevisionName.
	// +optional
	LatestCreatedRevisionName string `json:"latestCreatedRevisionName,omitempty"`

	// ObservedGeneration is the 'Generation' of the Configuration that
	// was last processed by the controller. The observed generation is updated
	// even if the controller failed to process the spec and create the Revision.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfigurationList is a list of Configuration resources
type ConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Configuration `json:"items"`
}

func (r *Configuration) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Configuration")
}

// IsReady looks at the conditions to see if they are happy.
func (cs *ConfigurationStatus) IsReady() bool {
	return confCondSet.Manage(cs).IsHappy()
}

// IsLatestReadyRevisionNameUpToDate returns true if the Configuration is ready
// and LatestCreateRevisionName is equal to LatestReadyRevisionName. Otherwise
// it returns false.
func (cs *ConfigurationStatus) IsLatestReadyRevisionNameUpToDate() bool {
	return cs.IsReady() &&
		cs.LatestCreatedRevisionName == cs.LatestReadyRevisionName
}

func (cs *ConfigurationStatus) GetCondition(t duckv1alpha1.ConditionType) *duckv1alpha1.Condition {
	return confCondSet.Manage(cs).GetCondition(t)
}

func (cs *ConfigurationStatus) InitializeConditions() {
	confCondSet.Manage(cs).InitializeConditions()
}

func (cs *ConfigurationStatus) SetLatestCreatedRevisionName(name string) {
	cs.LatestCreatedRevisionName = name
	if cs.LatestReadyRevisionName != name {
		confCondSet.Manage(cs).MarkUnknown(
			ConfigurationConditionReady,
			"",
			"")
	}
}

func (cs *ConfigurationStatus) SetLatestReadyRevisionName(name string) {
	cs.LatestReadyRevisionName = name
	confCondSet.Manage(cs).MarkTrue(ConfigurationConditionReady)
}

func (cs *ConfigurationStatus) MarkLatestCreatedFailed(name, message string) {
	confCondSet.Manage(cs).MarkFalse(
		ConfigurationConditionReady,
		"RevisionFailed",
		"Revision %q failed with message: %s.", name, message)
}

func (cs *ConfigurationStatus) MarkRevisionCreationFailed(message string) {
	confCondSet.Manage(cs).MarkFalse(
		ConfigurationConditionReady,
		"RevisionFailed",
		"Revision creation failed with message: %s.", message)
}

func (cs *ConfigurationStatus) MarkLatestReadyDeleted() {
	confCondSet.Manage(cs).MarkFalse(
		ConfigurationConditionReady,
		"RevisionDeleted",
		"Revision %q was deleted.", cs.LatestReadyRevisionName)
}

// GetConditions returns the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (cs *ConfigurationStatus) GetConditions() duckv1alpha1.Conditions {
	return cs.Conditions
}

// SetConditions sets the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (cs *ConfigurationStatus) SetConditions(conditions duckv1alpha1.Conditions) {
	cs.Conditions = conditions
}
