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
	ScaleTargetRef  ObjectReference `json:"scaleTargetRef"`
	PollingInterval *int32          `json:"pollingInterval"`
	CooldownPeriod  *int32          `json:"cooldownPeriod"`
	MinReplicaCount *int32          `json:"minReplicaCount"`
	MaxReplicaCount *int32          `json:"maxReplicaCount"`
	Triggers        []ScaleTriggers `json:"triggers"`
}

// ObjectReference holds the a reference to the deployment this
// ScaledObject applies
type ObjectReference struct {
	DeploymentName string `json:"deploymentName"`
	ContainerName  string `json:"containerName"`
}

type ScaleTriggers struct {
	Type           string                     `json:"type"`
	Name           string                     `json:"name"`
	Metadata       map[string]string          `json:"metadata"`
	Authentication ScaledObjectAuthentication `json:"authentication"`
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

// ScaledObjectAuthentication defines how the trigger can authenticate
type ScaledObjectAuthentication struct {
	// +optional
	AzureMSI bool `json:"azureMsi"`

	// +optional
	SecretRef []AuthenticationSecretRef `json:"secretRef"`

	// +optional
	Env []AuthenticationEnvironment `json:"env"`
}

// AuthenticationSecretRef is used to authenticate using a reference to a secret
type AuthenticationSecretRef struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`
	Key       string `json:"key"`

	// +optional
	Namespace string `json:"namespace"`
}

// AuthenticationEnvironment is used to authenticate using environment variables
// in the destination deployment spec
type AuthenticationEnvironment struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`

	// +optional
	ContainerName string `json:"containerName"`
}
