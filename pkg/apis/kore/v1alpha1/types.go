package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScaledObject is a spoecification for a ScaledObject resource
type ScaledObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScaledObjectSpec   `json:"spec"`
	Status ScaledObjectStatus `json:"status"`
}

// ScaledObjectSpec is the spec for a ScaledObject resource
type ScaledObjectSpec struct {
	ScaleTargetRef ObjectReference `json:"scaleTargetRef"`
	Triggers       []ScaleTriggers `json:"triggers"`
}

type ObjectReference struct {
	DeploymentName string `json:"deploymentName"`
}

type ScaleTriggers struct {
	Type       string                     `json:"type"`
	Name       string                     `json:"name"`
	SecretRefs map[string]SecretReference `json:"secretRefs"`
	Metadata   map[string]string          `json:"metadata"`
}

// ScaledObjectStatus is the status for a ScaledObject resource
type ScaledObjectStatus struct {
	LastScaleTime   *metav1.Time      `json:"lastScaleTime,omitempty"`
	LastActiveTime  *metav1.Time      `json:"lastActiveTime,omitempty"`
	CurrentReplicas int32             `json:"currentReplicas"`
	DesiredReplicas int32             `json:"desiredReplicas"`
	CurrentMetrics  map[string]string `json:"currentMetrics"`
}

type SecretReference struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScaledObjectList is a list of ScaledObject resources
type ScaledObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ScaledObject `json:"items"`
}
