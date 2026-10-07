/*
Copyright 2026 The KEDA Authors

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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestAuthPodIdentityValidateServiceAccountName(t *testing.T) {
	type testCase struct {
		name        string
		podIdentity *AuthPodIdentity
		wantError   string
	}
	tests := []testCase{
		{name: "no pod identity"},
		{
			name:        "operator GCP identity",
			podIdentity: &AuthPodIdentity{Provider: PodIdentityProviderGCP},
		},
		{
			name:        "other provider without service account selection",
			podIdentity: &AuthPodIdentity{Provider: PodIdentityProviderAzureWorkload},
		},
		{
			name: "selected GCP service account",
			podIdentity: &AuthPodIdentity{
				Provider:           PodIdentityProviderGCP,
				ServiceAccountName: ptr.To("scaling.service-account"),
			},
		},
		{
			name: "selected Azure service account",
			podIdentity: &AuthPodIdentity{
				Provider: PodIdentityProviderAzureWorkload, ServiceAccountName: ptr.To("reader"),
				IdentityID: ptr.To("client-id"), IdentityTenantID: ptr.To("tenant-id"),
			},
		},
		{
			name: "selected AWS service account",
			podIdentity: &AuthPodIdentity{
				Provider: PodIdentityProviderAws, ServiceAccountName: ptr.To("reader"),
				RoleArn: ptr.To("arn:aws:iam::123456789012:role/reader"),
			},
		},
		{
			name: "Azure cannot inherit operator client",
			podIdentity: &AuthPodIdentity{
				Provider: PodIdentityProviderAzureWorkload, ServiceAccountName: ptr.To("reader"),
				IdentityTenantID: ptr.To("tenant-id"),
			},
			wantError: "requires explicit identityId and identityTenantId",
		},
		{
			name: "Azure cannot inherit operator tenant",
			podIdentity: &AuthPodIdentity{
				Provider: PodIdentityProviderAzureWorkload, ServiceAccountName: ptr.To("reader"),
				IdentityID: ptr.To("client-id"),
			},
			wantError: "requires explicit identityId and identityTenantId",
		},
		{
			name: "Azure empty client",
			podIdentity: &AuthPodIdentity{
				Provider: PodIdentityProviderAzureWorkload, ServiceAccountName: ptr.To("reader"),
				IdentityID: ptr.To(" "), IdentityTenantID: ptr.To("tenant-id"),
			},
			wantError: "requires explicit identityId and identityTenantId",
		},
		{
			name: "AWS cannot inherit operator role",
			podIdentity: &AuthPodIdentity{
				Provider: PodIdentityProviderAws, ServiceAccountName: ptr.To("reader"),
			},
			wantError: "requires an explicit roleArn",
		},
		{
			name: "AWS cannot use identity owner discovery",
			podIdentity: &AuthPodIdentity{
				Provider: PodIdentityProviderAws, ServiceAccountName: ptr.To("reader"),
				RoleArn: ptr.To("arn:aws:iam::123456789012:role/reader"), IdentityOwner: ptr.To("keda"),
			},
			wantError: "cannot be combined with identityOwner or externalID",
		},
		{
			name: "AWS external ID is not a web identity parameter",
			podIdentity: &AuthPodIdentity{
				Provider: PodIdentityProviderAws, ServiceAccountName: ptr.To("reader"),
				RoleArn: ptr.To("arn:aws:iam::123456789012:role/reader"), ExternalID: ptr.To("external-id"),
			},
			wantError: "cannot be combined with identityOwner or externalID",
		},
	}
	for _, name := range []string{"", "Invalid", "space name", "other-namespace/account", "-account", "account.", strings.Repeat("a", 254)} {
		tests = append(tests, testCase{
			name: "invalid service account name " + name,
			podIdentity: &AuthPodIdentity{
				Provider:           PodIdentityProviderGCP,
				ServiceAccountName: ptr.To(name),
			},
			wantError: "podIdentity.serviceAccountName must be a valid Kubernetes service account name",
		})
	}
	for _, provider := range []PodIdentityProvider{PodIdentityProviderNone, PodIdentityProviderAwsEKS, ""} {
		tests = append(tests, testCase{
			name: "unsupported provider " + string(provider),
			podIdentity: &AuthPodIdentity{
				Provider:           provider,
				ServiceAccountName: ptr.To("scaling-account"),
			},
			wantError: "podIdentity.serviceAccountName is only supported with providers gcp, azure-workload and aws",
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.podIdentity.ValidateServiceAccountName()
			if test.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.wantError)
			}
		})
	}
}

func TestTriggerAuthenticationServiceAccountSelection(t *testing.T) {
	selectedIdentity := &AuthPodIdentity{
		Provider:           PodIdentityProviderGCP,
		ServiceAccountName: ptr.To("scaling-account"),
	}
	tests := []struct {
		name      string
		spec      TriggerAuthenticationSpec
		wantError string
	}{
		{name: "no pod identity"},
		{
			name: "selected GCP service account",
			spec: TriggerAuthenticationSpec{PodIdentity: selectedIdentity},
		},
		{
			name: "selected Azure service account",
			spec: TriggerAuthenticationSpec{PodIdentity: &AuthPodIdentity{
				Provider: PodIdentityProviderAzureWorkload, ServiceAccountName: ptr.To("reader"),
				IdentityID: ptr.To("client-id"), IdentityTenantID: ptr.To("tenant-id"),
			}},
		},
		{
			name: "selected AWS service account",
			spec: TriggerAuthenticationSpec{PodIdentity: &AuthPodIdentity{
				Provider: PodIdentityProviderAws, ServiceAccountName: ptr.To("reader"),
				RoleArn: ptr.To("arn:aws:iam::123456789012:role/reader"),
			}},
		},
		{
			name: "service principal cannot override selected account",
			spec: TriggerAuthenticationSpec{
				PodIdentity: selectedIdentity, AzureServicePrincipal: &AzureServicePrincipal{},
			},
			wantError: "cannot be combined with azureServicePrincipal",
		},
		{
			name: "empty service account name",
			spec: TriggerAuthenticationSpec{PodIdentity: &AuthPodIdentity{
				Provider:           PodIdentityProviderGCP,
				ServiceAccountName: ptr.To(""),
			}},
			wantError: "podIdentity.serviceAccountName must be a valid Kubernetes service account name",
		},
		{
			name: "other provider",
			spec: TriggerAuthenticationSpec{PodIdentity: &AuthPodIdentity{
				Provider:           PodIdentityProviderAwsEKS,
				ServiceAccountName: ptr.To("scaling-account"),
			}},
			wantError: "podIdentity.serviceAccountName is only supported with providers gcp, azure-workload and aws",
		},
		{
			name:      "nested Azure Key Vault selection",
			spec:      TriggerAuthenticationSpec{AzureKeyVault: &AzureKeyVault{PodIdentity: selectedIdentity}},
			wantError: "azureKeyVault.podIdentity.serviceAccountName is not supported",
		},
		{
			name:      "nested AWS Secret Manager selection",
			spec:      TriggerAuthenticationSpec{AwsSecretManager: &AwsSecretManager{PodIdentity: selectedIdentity}},
			wantError: "awsSecretManager.podIdentity.serviceAccountName is not supported",
		},
		{
			name: "nested GCP Secret Manager selection with top-level GCP provider",
			spec: TriggerAuthenticationSpec{
				PodIdentity:      selectedIdentity,
				GCPSecretManager: &GCPSecretManager{PodIdentity: selectedIdentity},
			},
			wantError: "gcpSecretManager.podIdentity.serviceAccountName is not supported",
		},
		{
			name: "nested secret providers without service account selection",
			spec: TriggerAuthenticationSpec{
				AzureKeyVault:    &AzureKeyVault{PodIdentity: &AuthPodIdentity{Provider: PodIdentityProviderAzureWorkload}},
				AwsSecretManager: &AwsSecretManager{PodIdentity: &AuthPodIdentity{Provider: PodIdentityProviderAws}},
				GCPSecretManager: &GCPSecretManager{PodIdentity: &AuthPodIdentity{Provider: PodIdentityProviderGCP}},
			},
		},
		{
			name: "nested secret providers without pod identity",
			spec: TriggerAuthenticationSpec{
				AzureKeyVault:    &AzureKeyVault{},
				AwsSecretManager: &AwsSecretManager{},
				GCPSecretManager: &GCPSecretManager{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ta := &TriggerAuthentication{Spec: test.spec}
			cta := &ClusterTriggerAuthentication{Spec: test.spec}
			validators := map[string]func() error{
				"spec": test.spec.ValidateServiceAccountSelection,
				"TriggerAuthentication create": func() error {
					_, err := ta.ValidateCreate(nil)
					return err
				},
				"TriggerAuthentication update": func() error {
					_, err := ta.ValidateUpdate(&TriggerAuthentication{}, nil)
					return err
				},
				"ClusterTriggerAuthentication create": func() error {
					_, err := cta.ValidateCreate(nil)
					return err
				},
				"ClusterTriggerAuthentication update": func() error {
					_, err := cta.ValidateUpdate(&ClusterTriggerAuthentication{}, nil)
					return err
				},
			}
			for name, validate := range validators {
				t.Run(name, func(t *testing.T) {
					err := validate()
					if test.wantError == "" {
						assert.NoError(t, err)
					} else {
						assert.ErrorContains(t, err, test.wantError)
					}
				})
			}
		})
	}
}
