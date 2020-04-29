package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerAuthentication defines how a trigger can authenticate
// +genclient
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=triggerauthentications,scope=Namespaced,shortName=ta;triggerauth
// +kubebuilder:printcolumn:name="PodIdentity",type="string",JSONPath=".spec.podIdentity.provider"
// +kubebuilder:printcolumn:name="Secret",type="string",JSONPath=".spec.secretTargetRef[*].name"
// +kubebuilder:printcolumn:name="Env",type="string",JSONPath=".spec.env[*].name"
type TriggerAuthentication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TriggerAuthenticationSpec `json:"spec"`
}

// TriggerAuthenticationSpec defines the various ways to authenticate
// +k8s:openapi-gen=true
type TriggerAuthenticationSpec struct {
	// +optional
	PodIdentity AuthPodIdentity `json:"podIdentity"`

	// +optional
	SecretTargetRef []AuthSecretTargetRef `json:"secretTargetRef"`

	// +optional
	Env []AuthEnvironment `json:"env"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerAuthenticationList contains a list of TriggerAuthentication
type TriggerAuthenticationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []TriggerAuthentication `json:"items"`
}

// PodIdentityProvider contains the list of providers
type PodIdentityProvider string

const (
	PodIdentityProviderNone    PodIdentityProvider = "none"
	PodIdentityProviderAzure                       = "azure"
	PodIdentityProviderGCP                         = "gcp"
	PodIdentityProviderSpiffe                      = "spiffe"
	PodIdentityProviderAwsEKS                      = "aws-eks"
	PodIdentityProviderAwsKiam                     = "aws-kiam"
)

const (
	PodIdentityAnnotationEKS  = "eks.amazonaws.com/role-arn"
	PodIdentityAnnotationKiam = "iam.amazonaws.com/role"
)

// AuthPodIdentity allows users to select the platform native identity
// mechanism
// +k8s:openapi-gen=true
type AuthPodIdentity struct {
	Provider PodIdentityProvider `json:"provider"`
}

// AuthSecretTargetRef is used to authenticate using a reference to a secret
// +k8s:openapi-gen=true
type AuthSecretTargetRef struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`
	Key       string `json:"key"`
}

// AuthEnvironment is used to authenticate using environment variables
// in the destination ScaleTarget spec
// +k8s:openapi-gen=true
type AuthEnvironment struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`

	// +optional
	ContainerName string `json:"containerName"`
}

func init() {
	SchemeBuilder.Register(&TriggerAuthentication{}, &TriggerAuthenticationList{})
}
