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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterTriggerAuthentication defines how a trigger can authenticate globally
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:path=clustertriggerauthentications,scope=Cluster,shortName=cta;clustertriggerauth
// +kubebuilder:printcolumn:name="PodIdentity",type="string",JSONPath=".spec.podIdentity.provider"
// +kubebuilder:printcolumn:name="Secret",type="string",JSONPath=".spec.secretTargetRef[*].name"
// +kubebuilder:printcolumn:name="Env",type="string",JSONPath=".spec.env[*].name"
// +kubebuilder:printcolumn:name="VaultAddress",type="string",JSONPath=".spec.hashiCorpVault.address"
type ClusterTriggerAuthentication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TriggerAuthenticationSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterTriggerAuthenticationList contains a list of ClusterTriggerAuthentication
type ClusterTriggerAuthenticationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ClusterTriggerAuthentication `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerAuthentication defines how a trigger can authenticate
// +genclient
// +kubebuilder:resource:path=triggerauthentications,scope=Namespaced,shortName=ta;triggerauth
// +kubebuilder:printcolumn:name="PodIdentity",type="string",JSONPath=".spec.podIdentity.provider"
// +kubebuilder:printcolumn:name="Secret",type="string",JSONPath=".spec.secretTargetRef[*].name"
// +kubebuilder:printcolumn:name="Env",type="string",JSONPath=".spec.env[*].name"
// +kubebuilder:printcolumn:name="VaultAddress",type="string",JSONPath=".spec.hashiCorpVault.address"
type TriggerAuthentication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TriggerAuthenticationSpec `json:"spec"`
}

// TriggerAuthenticationSpec defines the various ways to authenticate
type TriggerAuthenticationSpec struct {
	// +optional
	PodIdentity *AuthPodIdentity `json:"podIdentity,omitempty"`

	// +optional
	SecretTargetRef []AuthSecretTargetRef `json:"secretTargetRef,omitempty"`

	// +optional
	Env []AuthEnvironment `json:"env,omitempty"`

	// +optional
	HashiCorpVault *HashiCorpVault `json:"hashiCorpVault,omitempty"`

	// +optional
	AzureKeyVault *AzureKeyVault `json:"azureKeyVault,omitempty"`
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

// PodIdentityProviderNone specifies the default state when there is no Identity Provider
// PodIdentityProvider<IDENTITY_PROVIDER> specifies other available Identity providers
const (
	PodIdentityProviderNone          PodIdentityProvider = "none"
	PodIdentityProviderAzure         PodIdentityProvider = "azure"
	PodIdentityProviderAzureWorkload PodIdentityProvider = "azure-workload"
	PodIdentityProviderGCP           PodIdentityProvider = "gcp"
	PodIdentityProviderSpiffe        PodIdentityProvider = "spiffe"
	PodIdentityProviderAwsEKS        PodIdentityProvider = "aws-eks"
	PodIdentityProviderAwsKiam       PodIdentityProvider = "aws-kiam"
)

// PodIdentityAnnotationEKS specifies aws role arn for aws-eks Identity Provider
// PodIdentityAnnotationKiam specifies aws role arn for aws-iam Identity Provider
const (
	PodIdentityAnnotationEKS  = "eks.amazonaws.com/role-arn"
	PodIdentityAnnotationKiam = "iam.amazonaws.com/role"
)

// AuthPodIdentity allows users to select the platform native identity
// mechanism
type AuthPodIdentity struct {
	Provider PodIdentityProvider `json:"provider"`
	// +optional
	IdentityID string `json:"identityId"`
}

// AuthSecretTargetRef is used to authenticate using a reference to a secret
type AuthSecretTargetRef struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`
	Key       string `json:"key"`
}

// AuthEnvironment is used to authenticate using environment variables
// in the destination ScaleTarget spec
type AuthEnvironment struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`

	// +optional
	ContainerName string `json:"containerName,omitempty"`
}

// HashiCorpVault is used to authenticate using Hashicorp Vault
type HashiCorpVault struct {
	Address        string              `json:"address"`
	Authentication VaultAuthentication `json:"authentication"`
	Secrets        []VaultSecret       `json:"secrets"`

	// +optional
	Namespace string `json:"namespace,omitempty"`

	// +optional
	Credential *Credential `json:"credential,omitempty"`

	// +optional
	Role string `json:"role,omitempty"`

	// +optional
	Mount string `json:"mount,omitempty"`
}

// Credential defines the Hashicorp Vault credentials depending on the authentication method
type Credential struct {
	// +optional
	Token string `json:"token,omitempty"`

	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// VaultAuthentication contains the list of Hashicorp Vault authentication methods
type VaultAuthentication string

// Client authenticating to Vault
const (
	VaultAuthenticationToken      VaultAuthentication = "token"
	VaultAuthenticationKubernetes VaultAuthentication = "kubernetes"
	// VaultAuthenticationAWS                            = "aws"
)

// VaultSecret defines the mapping between the path of the secret in Vault to the parameter
type VaultSecret struct {
	Parameter string `json:"parameter"`
	Path      string `json:"path"`
	Key       string `json:"key"`
}

// AzureKeyVault is used to authenticate using Azure Key Vault
type AzureKeyVault struct {
	VaultURI string                `json:"vaultUri"`
	Secrets  []AzureKeyVaultSecret `json:"secrets"`
	// +optional
	Credentials *AzureKeyVaultCredentials `json:"credentials"`
	// +optional
	PodIdentity *AuthPodIdentity `json:"podIdentity"`
	// +optional
	Cloud *AzureKeyVaultCloudInfo `json:"cloud"`
}

type AzureKeyVaultCredentials struct {
	ClientID     string                     `json:"clientId"`
	TenantID     string                     `json:"tenantId"`
	ClientSecret *AzureKeyVaultClientSecret `json:"clientSecret"`
}

type AzureKeyVaultClientSecret struct {
	ValueFrom ValueFromSecret `json:"valueFrom"`
}

type ValueFromSecret struct {
	SecretKeyRef SecretKeyRef `json:"secretKeyRef"`
}

type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type AzureKeyVaultSecret struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`
	// +optional
	Version string `json:"version,omitempty"`
}

type AzureKeyVaultCloudInfo struct {
	Type string `json:"type"`
	// +optional
	KeyVaultResourceURL string `json:"keyVaultResourceURL"`
	// +optional
	ActiveDirectoryEndpoint string `json:"activeDirectoryEndpoint"`
}

func init() {
	SchemeBuilder.Register(&ClusterTriggerAuthentication{}, &ClusterTriggerAuthenticationList{})
	SchemeBuilder.Register(&TriggerAuthentication{}, &TriggerAuthenticationList{})
}
