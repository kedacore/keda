/*
Copyright 2021 The KEDA Authors

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
	"reflect"
	"strconv"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=scaledobjects,scope=Namespaced,shortName=so
// +kubebuilder:printcolumn:name="ScaleTargetKind",type="string",JSONPath=".status.scaleTargetKind"
// +kubebuilder:printcolumn:name="ScaleTargetName",type="string",JSONPath=".spec.scaleTargetRef.name"
// +kubebuilder:printcolumn:name="Min",type="integer",JSONPath=".spec.minReplicaCount"
// +kubebuilder:printcolumn:name="Max",type="integer",JSONPath=".spec.maxReplicaCount"
// +kubebuilder:printcolumn:name="Triggers",type="string",JSONPath=".spec.triggers[*].type"
// +kubebuilder:printcolumn:name="Authentication",type="string",JSONPath=".spec.triggers[*].authenticationRef.name"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[?(@.type==\"Active\")].status"
// +kubebuilder:printcolumn:name="Fallback",type="string",JSONPath=".status.conditions[?(@.type==\"Fallback\")].status"
// +kubebuilder:printcolumn:name="Paused",type="string",JSONPath=".status.conditions[?(@.type==\"Paused\")].status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ScaledObject is a specification for a ScaledObject resource
type ScaledObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ScaledObjectSpec `json:"spec"`
	// +optional
	Status ScaledObjectStatus `json:"status,omitempty"`
}

const ScaledObjectOwnerAnnotation = "scaledobject.keda.sh/name"
const ScaledObjectTransferHpaOwnershipAnnotation = "scaledobject.keda.sh/transfer-hpa-ownership"
const ValidationsHpaOwnershipAnnotation = "validations.keda.sh/hpa-ownership"
const PausedReplicasAnnotation = "autoscaling.keda.sh/paused-replicas"
const PausedAnnotation = "autoscaling.keda.sh/paused"

// HealthStatus is the status for a ScaledObject's health
type HealthStatus struct {
	// +optional
	NumberOfFailures *int32 `json:"numberOfFailures,omitempty"`
	// +optional
	Status HealthStatusType `json:"status,omitempty"`
}

// HealthStatusType is an indication of whether the health status is happy or failing
type HealthStatusType string

const (
	// HealthStatusHappy means the status of the health object is happy
	HealthStatusHappy HealthStatusType = "Happy"

	// HealthStatusFailing means the status of the health object is failing
	HealthStatusFailing HealthStatusType = "Failing"

	// Composite metric name used for scalingModifiers composite metric
	CompositeMetricName string = "composite-metric"

	defaultHPAMinReplicas int32 = 1
	defaultHPAMaxReplicas int32 = 100
)

// ScaledObjectSpec is the spec for a ScaledObject resource
type ScaledObjectSpec struct {
	ScaleTargetRef *ScaleTarget `json:"scaleTargetRef"`
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`
	// +optional
	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`
	// +optional
	IdleReplicaCount *int32 `json:"idleReplicaCount,omitempty"`
	// +optional
	MinReplicaCount *int32 `json:"minReplicaCount,omitempty"`
	// +optional
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
	// +optional
	Advanced *AdvancedConfig `json:"advanced,omitempty"`

	Triggers []ScaleTriggers `json:"triggers"`
	// +optional
	Fallback *Fallback `json:"fallback,omitempty"`
	// +optional
	InitialCooldownPeriod int32 `json:"initialCooldownPeriod,omitempty"`
}

// Fallback is the spec for fallback options
type Fallback struct {
	FailureThreshold int32 `json:"failureThreshold"`
	Replicas         int32 `json:"replicas"`
}

// AdvancedConfig specifies advance scaling options
type AdvancedConfig struct {
	// +optional
	HorizontalPodAutoscalerConfig *HorizontalPodAutoscalerConfig `json:"horizontalPodAutoscalerConfig,omitempty"`
	// +optional
	RestoreToOriginalReplicaCount bool `json:"restoreToOriginalReplicaCount,omitempty"`
	// +optional
	ScalingModifiers ScalingModifiers `json:"scalingModifiers,omitempty"`
}

// ScalingModifiers describes advanced scaling logic options like formula
type ScalingModifiers struct {
	Formula string `json:"formula,omitempty"`
	Target  string `json:"target,omitempty"`
	// +optional
	ActivationTarget string `json:"activationTarget,omitempty"`
	// +optional
	MetricType autoscalingv2.MetricTargetType `json:"metricType,omitempty"`
}

// HorizontalPodAutoscalerConfig specifies horizontal scale config
type HorizontalPodAutoscalerConfig struct {
	// +optional
	Behavior *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
	// +optional
	Name string `json:"name,omitempty"`
}

// ScaleTarget holds the reference to the scale target Object
type ScaleTarget struct {
	Name string `json:"name"`
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
	// +optional
	Kind string `json:"kind,omitempty"`
	// +optional
	EnvSourceContainerName string `json:"envSourceContainerName,omitempty"`
}

// +k8s:openapi-gen=true

// ScaledObjectStatus is the status for a ScaledObject resource
// +optional
type ScaledObjectStatus struct {
	// +optional
	ScaleTargetKind string `json:"scaleTargetKind,omitempty"`
	// +optional
	ScaleTargetGVKR *GroupVersionKindResource `json:"scaleTargetGVKR,omitempty"`
	// +optional
	OriginalReplicaCount *int32 `json:"originalReplicaCount,omitempty"`
	// +optional
	LastActiveTime *metav1.Time `json:"lastActiveTime,omitempty"`
	// +optional
	ExternalMetricNames []string `json:"externalMetricNames,omitempty"`
	// +optional
	ResourceMetricNames []string `json:"resourceMetricNames,omitempty"`
	// +optional
	CompositeScalerName string `json:"compositeScalerName,omitempty"`
	// +optional
	Conditions Conditions `json:"conditions,omitempty"`
	// +optional
	Health map[string]HealthStatus `json:"health,omitempty"`
	// +optional
	PausedReplicaCount *int32 `json:"pausedReplicaCount,omitempty"`
	// +optional
	HpaName string `json:"hpaName,omitempty"`
}

// +kubebuilder:object:root=true

// ScaledObjectList is a list of ScaledObject resources
type ScaledObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ScaledObject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScaledObject{}, &ScaledObjectList{})
}

// GenerateIdentifier returns identifier for the object in for "kind.namespace.name"
func (so *ScaledObject) GenerateIdentifier() string {
	return GenerateIdentifier("ScaledObject", so.Namespace, so.Name)
}

func (so *ScaledObject) HasPausedReplicaAnnotation() bool {
	_, pausedReplicasAnnotationFound := so.GetAnnotations()[PausedReplicasAnnotation]
	return pausedReplicasAnnotationFound
}

// HasPausedAnnotation returns whether this ScaledObject has PausedAnnotation or PausedReplicasAnnotation
func (so *ScaledObject) HasPausedAnnotation() bool {
	_, pausedAnnotationFound := so.GetAnnotations()[PausedAnnotation]
	_, pausedReplicasAnnotationFound := so.GetAnnotations()[PausedReplicasAnnotation]
	return pausedAnnotationFound || pausedReplicasAnnotationFound
}

// NeedToBePausedByAnnotation will check whether ScaledObject needs to be paused based on PausedAnnotation or PausedReplicaCount
func (so *ScaledObject) NeedToBePausedByAnnotation() bool {
	_, pausedReplicasAnnotationFound := so.GetAnnotations()[PausedReplicasAnnotation]
	if pausedReplicasAnnotationFound {
		return so.Status.PausedReplicaCount != nil
	}

	pausedAnnotationValue, pausedAnnotationFound := so.GetAnnotations()[PausedAnnotation]
	if !pausedAnnotationFound {
		return false
	}
	shouldPause, err := strconv.ParseBool(pausedAnnotationValue)
	if err != nil {
		// if annotation value is not a boolean, we assume user wants to pause the ScaledObject
		return true
	}
	return shouldPause
}

// IsUsingModifiers determines whether scalingModifiers are defined or not
func (so *ScaledObject) IsUsingModifiers() bool {
	return so.Spec.Advanced != nil && !reflect.DeepEqual(so.Spec.Advanced.ScalingModifiers, ScalingModifiers{})
}

// getHPAMinReplicas returns MinReplicas based on definition in ScaledObject or default value if not defined
func (so *ScaledObject) GetHPAMinReplicas() *int32 {
	if so.Spec.MinReplicaCount != nil && *so.Spec.MinReplicaCount > 0 {
		return so.Spec.MinReplicaCount
	}
	tmp := defaultHPAMinReplicas
	return &tmp
}

// getHPAMaxReplicas returns MaxReplicas based on definition in ScaledObject or default value if not defined
func (so *ScaledObject) GetHPAMaxReplicas() int32 {
	if so.Spec.MaxReplicaCount != nil {
		return *so.Spec.MaxReplicaCount
	}
	return defaultHPAMaxReplicas
}

// checkReplicaCountBoundsAreValid checks that Idle/Min/Max ReplicaCount defined in ScaledObject are correctly specified
// i.e. that Min is not greater than Max or Idle greater or equal to Min
func CheckReplicaCountBoundsAreValid(scaledObject *ScaledObject) error {
	min := int32(0)
	if scaledObject.Spec.MinReplicaCount != nil {
		min = *scaledObject.GetHPAMinReplicas()
	}
	max := scaledObject.GetHPAMaxReplicas()

	if min > max {
		return fmt.Errorf("MinReplicaCount=%d must be less than MaxReplicaCount=%d", min, max)
	}

	if scaledObject.Spec.IdleReplicaCount != nil && *scaledObject.Spec.IdleReplicaCount >= min {
		return fmt.Errorf("IdleReplicaCount=%d must be less than MinReplicaCount=%d", *scaledObject.Spec.IdleReplicaCount, min)
	}

	return nil
}

// CheckFallbackValid checks that the fallback supports scalers with an AverageValue metric target.
// Consequently, it does not support CPU & memory scalers, or scalers targeting a Value metric type.
func CheckFallbackValid(scaledObject *ScaledObject) error {
	if scaledObject.Spec.Fallback == nil {
		return nil
	}

	if scaledObject.Spec.Fallback.FailureThreshold < 0 || scaledObject.Spec.Fallback.Replicas < 0 {
		return fmt.Errorf("FailureThreshold=%d & Replicas=%d must both be greater than or equal to 0",
			scaledObject.Spec.Fallback.FailureThreshold, scaledObject.Spec.Fallback.Replicas)
	}

	for _, trigger := range scaledObject.Spec.Triggers {
		if trigger.Type == cpuString || trigger.Type == memoryString {
			return fmt.Errorf("type is %s, but fallback it is not supported by the CPU & memory scalers", trigger.Type)
		}
		if trigger.MetricType != autoscalingv2.AverageValueMetricType {
			return fmt.Errorf("MetricType=%s, but Fallback can only be enabled for triggers with metric of type AverageValue", trigger.MetricType)
		}
	}
	return nil
}
