package v1alpha1

import (
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScaledObject is a specification for a ScaledObject resource
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=scaledobjects,scope=Namespaced,shortName=so
// +kubebuilder:printcolumn:name="ScaleTargetKind",type="string",JSONPath=".status.scaleTargetKind"
// +kubebuilder:printcolumn:name="ScaleTargetName",type="string",JSONPath=".spec.scaleTargetRef.name"
// +kubebuilder:printcolumn:name="Triggers",type="string",JSONPath=".spec.triggers[*].type"
// +kubebuilder:printcolumn:name="Authentication",type="string",JSONPath=".spec.triggers[*].authenticationRef.name"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[?(@.type==\"Active\")].status"
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
	ScaleTargetRef *ScaleTarget `json:"scaleTargetRef"`
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`
	// +optional
	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`
	// +optional
	MinReplicaCount *int32 `json:"minReplicaCount,omitempty"`
	// +optional
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
	// +optional
	Behavior *autoscalingv2beta2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`

	Triggers []ScaleTriggers `json:"triggers"`
}

//ScaleTarget holds the a reference to the scale target Object
// +k8s:openapi-gen=true
type ScaleTarget struct {
	Name string `json:"name"`
	// +optional
	ApiVersion string `json:"apiVersion,omitempty"`
	// +optional
	Kind string `json:"kind,omitempty"`

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
	AuthenticationRef *ScaledObjectAuthRef `json:"authenticationRef,omitempty"`
}

// ScaledObjectStatus is the status for a ScaledObject resource
// +k8s:openapi-gen=true
// +optional
type ScaledObjectStatus struct {
	// +optional
	ScaleTargetKind string `json:"scaleTargetKind,omitempty"`
	// +optional
	ScaleTargetGVKR *GroupVersionKindResource `json:"scaleTargetGVKR,omitempty"`
	// +optional
	LastActiveTime *metav1.Time `json:"lastActiveTime,omitempty"`
	// +optional
	ExternalMetricNames []string `json:"externalMetricNames,omitempty"`
	// +optional
	Conditions Conditions `json:"conditions,omitempty"`
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
