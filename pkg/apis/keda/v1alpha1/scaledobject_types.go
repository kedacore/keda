package v1alpha1

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScaledObjectScaleType distinguish between Deployment based and K8s Jobs
type ScaledObjectScaleType string

const (
	// ScaleTypeDeployment specifies Deployment based ScaleObject
	ScaleTypeDeployment ScaledObjectScaleType = "deployment"
	// ScaleTypeJob specifies K8s Jobs based ScaleObject
	ScaleTypeJob ScaledObjectScaleType = "job"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScaledObject is a specification for a ScaledObject resource
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=scaledobjects,scope=Namespaced
// +kubebuilder:printcolumn:name="Deployment",type="string",JSONPath=".spec.scaleTargetRef.deploymentName"
// +kubebuilder:printcolumn:name="Triggers",type="string",JSONPath=".spec.triggers[*].type"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ScaledObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ScaledObjectSpec `json:"spec"`
	// +optional
	Status ScaledObjectStatus `json:"status,omitempty"`
}

// ScaledObjectSpec is the spec for a ScaledObject resource
// +k8s:openapi-gen=true
type ScaledObjectSpec struct {
	// +optional
	ScaleType ScaledObjectScaleType `json:"scaleType,omitempty"`
	// +optional
	ScaleTargetRef ObjectReference `json:"scaleTargetRef,omitempty"`
	// +optional
	JobTargetRef *batchv1.JobSpec `json:"jobTargetRef,omitempty"`
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`
	// +optional
	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`
	// +optional
	MinReplicaCount *int32 `json:"minReplicaCount,omitempty"`
	// +optional
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
	// +listType
	Triggers []ScaleTriggers `json:"triggers"`
}

// ObjectReference holds the a reference to the deployment this
// ScaledObject applies
// +k8s:openapi-gen=true
type ObjectReference struct {
	DeploymentName string `json:"deploymentName"`
	// +optional
	ContainerName string `json:"containerName,omitempty"`
}

// ScaleTriggers reference the scaler that will be used
// +k8s:openapi-gen=true
type ScaleTriggers struct {
	Type string `json:"type"`
	// +optional
	Name     string            `json:"name,omitempty"`
	Metadata map[string]string `json:"metadata"`
	// +optional
	AuthenticationRef ScaledObjectAuthRef `json:"authenticationRef,omitempty"`
}

// ScaledObjectStatus is the status for a ScaledObject resource
// +k8s:openapi-gen=true
// +optional
type ScaledObjectStatus struct {
	// +optional
	LastActiveTime  *metav1.Time `json:"lastActiveTime,omitempty"`
	CurrentReplicas int32        `json:"currentReplicas"`
	DesiredReplicas int32        `json:"desiredReplicas"`
	// +optional
	// +listType
	ExternalMetricNames []string `json:"externalMetricNames,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScaledObjectList is a list of ScaledObject resources
type ScaledObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ScaledObject `json:"items"`
}

// ScaledObjectAuthRef points to the TriggerAuthentication object that
// is used to authenticate the scaler with the environment
// +k8s:openapi-gen=true
type ScaledObjectAuthRef struct {
	Name string `json:"name"`
}

func init() {
	SchemeBuilder.Register(&ScaledObject{}, &ScaledObjectList{})
}
