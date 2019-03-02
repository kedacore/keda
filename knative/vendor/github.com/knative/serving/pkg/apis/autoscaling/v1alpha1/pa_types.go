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
	"fmt"
	"strconv"
	"time"

	"github.com/knative/pkg/apis"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/serving/pkg/apis/autoscaling"
	servingv1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodAutoscaler is a Knative abstraction that encapsulates the interface by which Knative
// components instantiate autoscalers.  This definition is an abstraction that may be backed
// by multiple definitions.  For more information, see the Knative Pluggability presentation:
// https://docs.google.com/presentation/d/10KWynvAJYuOEWy69VBa6bHJVCqIsz1TNdEKosNvcpPY/edit
type PodAutoscaler struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the PodAutoscaler (from the client).
	// +optional
	Spec PodAutoscalerSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the PodAutoscaler (from the controller).
	// +optional
	Status PodAutoscalerStatus `json:"status,omitempty"`
}

// Check that PodAutoscaler can be validated, can be defaulted, and has immutable fields.
var _ apis.Validatable = (*PodAutoscaler)(nil)
var _ apis.Defaultable = (*PodAutoscaler)(nil)
var _ apis.Immutable = (*PodAutoscaler)(nil)

// Check that ConfigurationStatus may have its conditions managed.
var _ duckv1alpha1.ConditionsAccessor = (*PodAutoscalerStatus)(nil)

// PodAutoscalerSpec holds the desired state of the PodAutoscaler (from the client).
type PodAutoscalerSpec struct {
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

	// ConcurrencyModel specifies the desired concurrency model
	// (Single or Multi) for the scale target. Defaults to Multi.
	// Deprecated in favor of ContainerConcurrency.
	// +optional
	ConcurrencyModel servingv1alpha1.RevisionRequestConcurrencyModelType `json:"concurrencyModel,omitempty"`

	// ContainerConcurrency specifies the maximum allowed
	// in-flight (concurrent) requests per container of the Revision.
	// Defaults to `0` which means unlimited concurrency.
	// This field replaces ConcurrencyModel. A value of `1`
	// is equivalent to `Single` and `0` is equivalent to `Multi`.
	// +optional
	ContainerConcurrency servingv1alpha1.RevisionContainerConcurrencyType `json:"containerConcurrency,omitempty"`

	// ScaleTargetRef defines the /scale-able resource that this PodAutoscaler
	// is responsible for quickly right-sizing.
	ScaleTargetRef autoscalingv1.CrossVersionObjectReference `json:"scaleTargetRef"`

	// ServiceName holds the name of a core Kubernetes Service resource that
	// load balances over the pods referenced by the ScaleTargetRef.
	ServiceName string `json:"serviceName"`
}

const (
	// PodAutoscalerConditionReady is set when the revision is starting to materialize
	// runtime resources, and becomes true when those resources are ready.
	PodAutoscalerConditionReady = duckv1alpha1.ConditionReady
	// PodAutoscalerConditionActive is set when the PodAutoscaler's ScaleTargetRef is receiving traffic.
	PodAutoscalerConditionActive duckv1alpha1.ConditionType = "Active"
)

var podCondSet = duckv1alpha1.NewLivingConditionSet(PodAutoscalerConditionActive)

// PodAutoscalerStatus communicates the observed state of the PodAutoscaler (from the controller).
type PodAutoscalerStatus struct {
	// Conditions communicates information about ongoing/complete
	// reconciliation processes that bring the "spec" inline with the observed
	// state of the world.
	// +optional
	Conditions duckv1alpha1.Conditions `json:"conditions,omitempty"`

	// ObservedGeneration is the 'Generation' of the PodAutoscaler that
	// was last processed by the controller. The observed generation is updated
	// even if the controller failed to process the spec.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodAutoscalerList is a list of PodAutoscaler resources
type PodAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PodAutoscaler `json:"items"`
}

func (pa *PodAutoscaler) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("PodAutoscaler")
}

func (pa *PodAutoscaler) Class() string {
	if c, ok := pa.Annotations[autoscaling.ClassAnnotationKey]; ok {
		return c
	}
	// Default to "kpa" class for backward compatibility.
	return autoscaling.KPA
}

func (pa *PodAutoscaler) annotationInt32(key string) int32 {
	if s, ok := pa.Annotations[key]; ok {
		// no error check: relying on validation
		i, _ := strconv.ParseInt(s, 10, 32)
		if i < 0 {
			return 0
		}
		return int32(i)
	}
	return 0
}

// ScaleBounds returns scale bounds annotations values as a tuple:
// `(min, max int32)`. The value of 0 for any of min or max means the bound is
// not set
func (pa *PodAutoscaler) ScaleBounds() (min, max int32) {
	min = pa.annotationInt32(autoscaling.MinScaleAnnotationKey)
	max = pa.annotationInt32(autoscaling.MaxScaleAnnotationKey)
	return
}

// Target returns the target annotation value or false if not present.
func (pa *PodAutoscaler) Target() (target int32, ok bool) {
	if s, ok := pa.Annotations[autoscaling.TargetAnnotationKey]; ok {
		if i, err := strconv.Atoi(s); err == nil {
			if i < 1 {
				return 0, false
			}
			return int32(i), true
		}
	}
	return 0, false
}

// IsReady looks at the conditions and if the Status has a condition
// PodAutoscalerConditionReady returns true if ConditionStatus is True
func (rs *PodAutoscalerStatus) IsReady() bool {
	return podCondSet.Manage(rs).IsHappy()
}

// IsActivating assumes the pod autoscaler is Activating if it is neither
// Active nor Inactive
func (rs *PodAutoscalerStatus) IsActivating() bool {
	cond := rs.GetCondition(PodAutoscalerConditionActive)

	return cond != nil && cond.Status == corev1.ConditionUnknown
}

func (rs *PodAutoscalerStatus) GetCondition(t duckv1alpha1.ConditionType) *duckv1alpha1.Condition {
	return podCondSet.Manage(rs).GetCondition(t)
}

func (rs *PodAutoscalerStatus) InitializeConditions() {
	podCondSet.Manage(rs).InitializeConditions()
}

func (rs *PodAutoscalerStatus) MarkActive() {
	podCondSet.Manage(rs).MarkTrue(PodAutoscalerConditionActive)
}

func (rs *PodAutoscalerStatus) MarkActivating(reason, message string) {
	podCondSet.Manage(rs).MarkUnknown(PodAutoscalerConditionActive, reason, message)
}

func (rs *PodAutoscalerStatus) MarkInactive(reason, message string) {
	podCondSet.Manage(rs).MarkFalse(PodAutoscalerConditionActive, reason, message)
}

// MarkResourceNotOwned changes the "Active" condition to false to reflect that the
// resource of the given kind and name has already been created, and we do not own it.
func (rs *PodAutoscalerStatus) MarkResourceNotOwned(kind, name string) {
	rs.MarkInactive("NotOwned",
		fmt.Sprintf("There is an existing %s %q that we do not own.", kind, name))
}

// MarkResourceFailedCreation changes the "Active" condition to false to reflect that a
// critical resource of the given kind and name was unable to be created.
func (rs *PodAutoscalerStatus) MarkResourceFailedCreation(kind, name string) {
	rs.MarkInactive("FailedCreate",
		fmt.Sprintf("Failed to create %s %q.", kind, name))
}

// CanScaleToZero checks whether the pod autoscaler has been in an inactive state
// for at least the specified grace period.
func (rs *PodAutoscalerStatus) CanScaleToZero(gracePeriod time.Duration) bool {
	if cond := rs.GetCondition(PodAutoscalerConditionActive); cond != nil {
		switch cond.Status {
		case corev1.ConditionFalse:
			// Check that this PodAutoscaler has been inactive for
			// at least the grace period.
			return time.Now().After(cond.LastTransitionTime.Inner.Add(gracePeriod))
		}
	}
	return false
}

// CanMarkInactive checks whether the pod autoscaler has been in an active state
// for at least the specified idle period.
func (rs *PodAutoscalerStatus) CanMarkInactive(idlePeriod time.Duration) bool {
	if cond := rs.GetCondition(PodAutoscalerConditionActive); cond != nil {
		switch cond.Status {
		case corev1.ConditionTrue:
			// Check that this PodAutoscaler has been active for
			// at least the grace period.
			return time.Now().After(cond.LastTransitionTime.Inner.Add(idlePeriod))
		}
	}
	return false
}

// GetConditions returns the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (rs *PodAutoscalerStatus) GetConditions() duckv1alpha1.Conditions {
	return rs.Conditions
}

// SetConditions sets the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (rs *PodAutoscalerStatus) SetConditions(conditions duckv1alpha1.Conditions) {
	rs.Conditions = conditions
}
