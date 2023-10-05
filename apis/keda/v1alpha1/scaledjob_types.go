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
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultScaledJobMaxReplicaCount = 100
	defaultScaledJobMinReplicaCount = 0
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=scaledjobs,scope=Namespaced,shortName=sj
// +kubebuilder:printcolumn:name="Min",type="integer",JSONPath=".spec.minReplicaCount"
// +kubebuilder:printcolumn:name="Max",type="integer",JSONPath=".spec.maxReplicaCount"
// +kubebuilder:printcolumn:name="Triggers",type="string",JSONPath=".spec.triggers[*].type"
// +kubebuilder:printcolumn:name="Authentication",type="string",JSONPath=".spec.triggers[*].authenticationRef.name"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[?(@.type==\"Active\")].status"
// +kubebuilder:printcolumn:name="Paused",type="string",JSONPath=".status.conditions[?(@.type==\"Paused\")].status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ScaledJob is the Schema for the scaledjobs API
type ScaledJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScaledJobSpec   `json:"spec,omitempty"`
	Status ScaledJobStatus `json:"status,omitempty"`
}

// ScaledJobSpec defines the desired state of ScaledJob
type ScaledJobSpec struct {
	JobTargetRef *batchv1.JobSpec `json:"jobTargetRef"`
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`
	// +optional
	RolloutStrategy string `json:"rolloutStrategy,omitempty"`
	// +optional
	Rollout Rollout `json:"rollout,omitempty"`
	// +optional
	EnvSourceContainerName string `json:"envSourceContainerName,omitempty"`
	// +optional
	MinReplicaCount *int32 `json:"minReplicaCount,omitempty"`
	// +optional
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
	// +optional
	ScalingStrategy ScalingStrategy `json:"scalingStrategy,omitempty"`
	Triggers        []ScaleTriggers `json:"triggers"`
}

// ScaledJobStatus defines the observed state of ScaledJob
// +optional
type ScaledJobStatus struct {
	// +optional
	LastActiveTime *metav1.Time `json:"lastActiveTime,omitempty"`
	// +optional
	Conditions Conditions `json:"conditions,omitempty"`
	// +optional
	Paused string `json:"Paused,omitempty"`
}

// ScaledJobList contains a list of ScaledJob
// +kubebuilder:object:root=true
type ScaledJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScaledJob `json:"items"`
}

// ScalingStrategy defines the strategy of Scaling
// +optional
type ScalingStrategy struct {
	// +optional
	Strategy string `json:"strategy,omitempty"`
	// +optional
	CustomScalingQueueLengthDeduction *int32 `json:"customScalingQueueLengthDeduction,omitempty"`
	// +optional
	CustomScalingRunningJobPercentage string `json:"customScalingRunningJobPercentage,omitempty"`
	// +optional
	PendingPodConditions []string `json:"pendingPodConditions,omitempty"`
	// +optional
	MultipleScalersCalculation string `json:"multipleScalersCalculation,omitempty"`
}

// Rollout defines the strategy for job rollouts
// +optional
type Rollout struct {
	// +optional
	Strategy string `json:"strategy,omitempty"`
	// +optional
	PropagationPolicy string `json:"propagationPolicy,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ScaledJob{}, &ScaledJobList{})
}

// MaxReplicaCount returns MaxReplicaCount
func (s ScaledJob) MaxReplicaCount() int64 {
	if s.Spec.MaxReplicaCount != nil {
		if s.Spec.MinReplicaCount != nil && *s.Spec.MinReplicaCount > *s.Spec.MaxReplicaCount {
			return int64(*s.Spec.MaxReplicaCount)
		}
		return int64(*s.Spec.MaxReplicaCount) - s.MinReplicaCount()
	}

	return defaultScaledJobMaxReplicaCount
}

// MinReplicaCount returns MinReplicaCount
func (s ScaledJob) MinReplicaCount() int64 {
	if s.Spec.MinReplicaCount != nil {
		if s.Spec.MaxReplicaCount != nil &&
			*s.Spec.MinReplicaCount > *s.Spec.MaxReplicaCount {
			return int64(*s.Spec.MaxReplicaCount)
		}
		return int64(*s.Spec.MinReplicaCount)
	}
	return defaultScaledJobMinReplicaCount
}

func (s *ScaledJob) GenerateIdentifier() string {
	return GenerateIdentifier("ScaledJob", s.Namespace, s.Name)
}
