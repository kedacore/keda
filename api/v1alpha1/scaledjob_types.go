package v1alpha1

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=scaledjobs,scope=Namespaced,shortName=sj
// +kubebuilder:printcolumn:name="Triggers",type="string",JSONPath=".spec.triggers[*].type"
// +kubebuilder:printcolumn:name="Authentication",type="string",JSONPath=".spec.triggers[*].authenticationRef.name"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[?(@.type==\"Active\")].status"
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
	EnvSourceContainerName string `json:"envSourceContainerName,omitempty"`
	// +optional
	MaxReplicaCount *int32          `json:"maxReplicaCount,omitempty"`
	Triggers        []ScaleTriggers `json:"triggers"`
}

// ScaledJobStatus defines the observed state of ScaledJob
// +optional
type ScaledJobStatus struct {
	// +optional
	LastActiveTime *metav1.Time `json:"lastActiveTime,omitempty"`
	// +optional
	Conditions Conditions `json:"conditions,omitempty"`
}

// ScaledJobList contains a list of ScaledJob
// +kubebuilder:object:root=true
type ScaledJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScaledJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScaledJob{}, &ScaledJobList{})
}
