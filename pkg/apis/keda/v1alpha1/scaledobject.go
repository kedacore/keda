package v1alpha1

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScaledObjectScaleType (Deployment based vs. K8s Jobs)
type ScaledObjectScaleType string

const (
	ScaleTypeDeployment ScaledObjectScaleType = "deployment"
	ScaleTypeJob        ScaledObjectScaleType = "job"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScaledObject is a specification for a ScaledObject resource
type ScaledObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScaledObjectSpec   `json:"spec"`
	Status ScaledObjectStatus `json:"status"`
}

// ScaledObjectSpec is the spec for a ScaledObject resource
type ScaledObjectSpec struct {
	ScaleType       ScaledObjectScaleType `json:"scaleType"`
	ScaleTargetRef  ObjectReference       `json:"scaleTargetRef"`
	JobTargetRef    batchv1.JobSpec       `json:"jobTargetRef,omitempty"`
	PollingInterval *int32                `json:"pollingInterval,omitempty"`
	CooldownPeriod  *int32                `json:"cooldownPeriod,omitempty"`
	MinReplicaCount *int32                `json:"minReplicaCount,omitempty"`
	MaxReplicaCount *int32                `json:"maxReplicaCount,omitempty"`
	Triggers        []ScaleTriggers       `json:"triggers"`
}

// ObjectReference holds the a reference to the deployment this
// ScaledObject applies
type ObjectReference struct {
	DeploymentName string `json:"deploymentName"`
	ContainerName  string `json:"containerName"`
}

// ScaleTriggers reference the scaler that will be used
type ScaleTriggers struct {
	Type              string              `json:"type"`
	Name              string              `json:"name"`
	Metadata          map[string]string   `json:"metadata"`
	AuthenticationRef ScaledObjectAuthRef `json:"authenticationRef"`
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

// ScaledObjectAuthRef points to the TriggerAuthentication object that
// is used to authenticate the scaler with the environment
type ScaledObjectAuthRef struct {
	Name string `json:"name"`
}
