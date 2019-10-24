package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerAuthentication defines how a trigger can authenticate
type TriggerAuthentication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TriggerAuthenticationSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerAuthenticationList is a list of TriggerAuthentication resources
type TriggerAuthenticationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []TriggerAuthentication `json:"items"`
}

// TriggerAuthenticationSpec defines the various ways to authenticate
type TriggerAuthenticationSpec struct {
	// +optional
	PodIdentity AuthPodIdentity `json:"podIdentity"`

	// +optional
	SecretTargetRef []AuthSecretTargetRef `json:"secretTargetRef"`

	// +optional
	Env []AuthEnvironment `json:"env"`
}

// PodIdentityProvider contains the list of providers
type PodIdentityProvider string

const (
	PodIdentityProviderNone   PodIdentityProvider = "none"
	PodIdentityProviderAzure                      = "azure"
	PodIdentityProviderGCP                        = "gcp"
	PodIdentityProviderSpiffe                     = "spiffe"
)

// AuthPodIdentity allows users to select the platform native identity
// mechanism
type AuthPodIdentity struct {
	Provider PodIdentityProvider `json:"provider"`
}

// AuthSecretTargetRef is used to authenticate using a reference to a secret
type AuthSecretTargetRef struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`
	Key       string `json:"key"`
}

// AuthEnvironment is used to authenticate using environment variables
// in the destination deployment spec
type AuthEnvironment struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`

	// +optional
	ContainerName string `json:"containerName"`
}
