package v1alpha1

import (
	core_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Scaled Object Type (Deployment based vs. K8s Jobs)
const (
	ScaleTypeDeployment string = "deployment"
	ScaleTypeJob        string = "job"
)

// ScaledObject is a specification for a ScaledObject resource
type ScaledObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScaledObjectSpec   `json:"spec"`
	Status ScaledObjectStatus `json:"status"`
}

// ScaledObjectSpec is the spec for a ScaledObject resource
type ScaledObjectSpec struct {
	ScaleType       string           `json:"scaleType"`
	ScaleTargetRef  ObjectReference  `json:"scaleTargetRef"`
	PollingInterval *int32           `json:"pollingInterval"`
	CooldownPeriod  *int32           `json:"cooldownPeriod"`
	MinReplicaCount *int32           `json:"minReplicaCount"`
	MaxReplicaCount *int32           `json:"maxReplicaCount"`
	Parallelism     *int32           `json:"parallelism,omitempty"`
	Completions     *int32           `json:"completions,omitempty"`
	ActiveDeadline  *int32           `json:"activeDeadline,omitempty"`
	BackOffLimit    *int32           `json:"backOffLimit,omitempty"`
	Triggers        []ScaleTriggers  `json:"triggers"`
	ConsumerSpec    *core_v1.PodSpec `json:"consumerSpec,omitempty"`
}

// ObjectReference holds the a reference to the deployment this
// ScaledObject applies
type ObjectReference struct {
	DeploymentName string `json:"deploymentName"`
	ContainerName  string `json:"containerName"`
}

type ScaleTriggers struct {
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}

// ScaledObjectStatus is the status for a ScaledObject resource
type ScaledObjectStatus struct {
	LastActiveTime  *metav1.Time `json:"lastActiveTime,omitempty"`
	CurrentReplicas int32        `json:"currentReplicas"`
	DesiredReplicas int32        `json:"desiredReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScaledObjectList is a list of ScaledObject resources
type ScaledObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ScaledObject `json:"items"`
}
