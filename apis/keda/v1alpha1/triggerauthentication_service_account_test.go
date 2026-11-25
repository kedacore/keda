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
	"context"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				Provider:           PodIdentityProviderGCPWorkloadServiceAccount,
				ServiceAccountName: ptr.To("scaling.service-account"),
			},
		},
		{
			name: "selected GCP service account with valid impersonation email",
			podIdentity: &AuthPodIdentity{
				Provider:           PodIdentityProviderGCPWorkloadServiceAccount,
				ServiceAccountName: ptr.To("scaling.service-account"),
				IdentityID:         ptr.To("metrics-reader@tenant-project.iam.gserviceaccount.com"),
			},
		},
		{
			name: "selected GCP service account with Compute Engine default impersonation email",
			podIdentity: &AuthPodIdentity{
				Provider:           PodIdentityProviderGCPWorkloadServiceAccount,
				ServiceAccountName: ptr.To("scaling.service-account"),
				IdentityID:         ptr.To("123456789012-compute@developer.gserviceaccount.com"),
			},
		},
		{
			name: "selected GCP service account with App Engine default impersonation email",
			podIdentity: &AuthPodIdentity{
				Provider:           PodIdentityProviderGCPWorkloadServiceAccount,
				ServiceAccountName: ptr.To("scaling.service-account"),
				IdentityID:         ptr.To("tenant-project@appspot.gserviceaccount.com"),
			},
		},
		{
			name: "GCP rejects invalid impersonation email",
			podIdentity: &AuthPodIdentity{
				Provider:           PodIdentityProviderGCPWorkloadServiceAccount,
				ServiceAccountName: ptr.To("reader"),
				IdentityID:         ptr.To("reader@example.com"),
			},
			wantError: "must be a valid Google service account email",
		},
		{
			name: "GCP rejects whitespace around impersonation email",
			podIdentity: &AuthPodIdentity{
				Provider:           PodIdentityProviderGCPWorkloadServiceAccount,
				ServiceAccountName: ptr.To("reader"),
				IdentityID:         ptr.To(" reader@tenant-project.iam.gserviceaccount.com "),
			},
			wantError: "must be a valid Google service account email",
		},
		{
			name:        "selected GCP provider requires service account name",
			podIdentity: &AuthPodIdentity{Provider: PodIdentityProviderGCPWorkloadServiceAccount},
			wantError:   "requires podIdentity.serviceAccountName",
		},
	}
	for _, name := range []string{"", "Invalid", "space name", "other-namespace/account", "-account", "account.", strings.Repeat("a", 254)} {
		tests = append(tests, testCase{
			name: "invalid service account name " + name,
			podIdentity: &AuthPodIdentity{
				Provider:           PodIdentityProviderGCPWorkloadServiceAccount,
				ServiceAccountName: ptr.To(name),
			},
			wantError: "podIdentity.serviceAccountName must be a valid Kubernetes service account name",
		})
	}
	for _, provider := range []PodIdentityProvider{
		PodIdentityProviderNone, PodIdentityProviderAzureWorkload, PodIdentityProviderGCP,
		PodIdentityProviderAws, PodIdentityProviderAwsEKS, "",
	} {
		tests = append(tests, testCase{
			name: "unsupported provider " + string(provider),
			podIdentity: &AuthPodIdentity{
				Provider:           provider,
				ServiceAccountName: ptr.To("scaling-account"),
			},
			wantError: "podIdentity.serviceAccountName requires provider gcp-workload-service-account",
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

	var nilPodIdentity *AuthPodIdentity
	assert.False(t, nilPodIdentity.IsWorkloadServiceAccountProvider())
	assert.True(t, (&AuthPodIdentity{Provider: PodIdentityProviderGCPWorkloadServiceAccount}).IsWorkloadServiceAccountProvider())
	assert.False(t, (&AuthPodIdentity{Provider: PodIdentityProviderGCP}).IsWorkloadServiceAccountProvider())
}

func TestTriggerAuthenticationServiceAccountSelection(t *testing.T) {
	selectedIdentity := &AuthPodIdentity{
		Provider:           PodIdentityProviderGCPWorkloadServiceAccount,
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
			name: "service principal cannot override selected account",
			spec: TriggerAuthenticationSpec{
				PodIdentity: selectedIdentity, AzureServicePrincipal: &AzureServicePrincipal{},
			},
			wantError: "cannot be combined with azureServicePrincipal",
		},
		{
			name: "empty service account name",
			spec: TriggerAuthenticationSpec{PodIdentity: &AuthPodIdentity{
				Provider:           PodIdentityProviderGCPWorkloadServiceAccount,
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
			wantError: "podIdentity.serviceAccountName requires provider",
		},
		{
			name:      "nested Azure Key Vault selection",
			spec:      TriggerAuthenticationSpec{AzureKeyVault: &AzureKeyVault{PodIdentity: selectedIdentity}},
			wantError: "azureKeyVault.podIdentity.serviceAccountName is not supported",
		},
		{
			name: "nested selected provider without service account name",
			spec: TriggerAuthenticationSpec{AzureKeyVault: &AzureKeyVault{PodIdentity: &AuthPodIdentity{
				Provider: PodIdentityProviderGCPWorkloadServiceAccount,
			}}},
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
		{
			name: "GCP workload service account still validates OAuth2 in webhook",
			spec: TriggerAuthenticationSpec{
				PodIdentity: selectedIdentity,
				OAuth2: &OAuth2{
					Type:     OAuth2GrantTypeClientCredentials,
					ClientID: "client-id",
					TokenURL: "ftp://invalid.example/token",
				},
			},
			wantError: "oauth2.tokenUrl must be a valid http or https URL",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error
			if test.spec.OAuth2 != nil {
				_, err = validateSpec(&test.spec)
			} else {
				err = test.spec.ValidateServiceAccountSelection()
			}
			if test.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.wantError)
			}
		})
	}

	// Verify TriggerAuthentication and ClusterTriggerAuthentication create/update
	// admission hooks wire through to spec validation.
	invalidSpec := TriggerAuthenticationSpec{PodIdentity: &AuthPodIdentity{Provider: PodIdentityProviderGCPWorkloadServiceAccount}}
	t.Run("webhook entry points", func(t *testing.T) {
		ta := &TriggerAuthentication{Spec: invalidSpec}
		cta := &ClusterTriggerAuthentication{Spec: invalidSpec}
		_, err := ta.ValidateCreate(nil)
		assert.ErrorContains(t, err, "requires podIdentity.serviceAccountName")
		_, err = ta.ValidateUpdate(&TriggerAuthentication{}, nil)
		assert.ErrorContains(t, err, "requires podIdentity.serviceAccountName")
		_, err = cta.ValidateCreate(nil)
		assert.ErrorContains(t, err, "requires podIdentity.serviceAccountName")
		_, err = cta.ValidateUpdate(&ClusterTriggerAuthentication{}, nil)
		assert.ErrorContains(t, err, "requires podIdentity.serviceAccountName")
	})
}

var _ = Describe("Workload service account CRD CEL validation", func() {
	It("accepts valid TriggerAuthentication and ClusterTriggerAuthentication and enforces generated CRD CEL rules", func() {
		namespaceName := "workload-sa-cel"
		Expect(k8sClient.Create(context.Background(), createNamespace(namespaceName))).To(Succeed())

		selected := &AuthPodIdentity{
			Provider:           PodIdentityProviderGCPWorkloadServiceAccount,
			ServiceAccountName: ptr.To("scaling-reader"),
			IdentityID:         ptr.To("tenant-project@appspot.gserviceaccount.com"),
		}

		validTA := &TriggerAuthentication{
			ObjectMeta: metav1.ObjectMeta{Name: "valid-gcp-sa", Namespace: namespaceName},
			Spec:       TriggerAuthenticationSpec{PodIdentity: selected},
		}
		Expect(k8sClient.Create(context.Background(), validTA)).To(Succeed())

		validCTA := &ClusterTriggerAuthentication{
			ObjectMeta: metav1.ObjectMeta{Name: "valid-gcp-sa-cta"},
			Spec:       TriggerAuthenticationSpec{PodIdentity: selected},
		}
		Expect(k8sClient.Create(context.Background(), validCTA)).To(Succeed())

		celRejections := []struct {
			name        string
			spec        TriggerAuthenticationSpec
			wantMessage string
		}{
			{
				name:        "missing-sa-name",
				spec:        TriggerAuthenticationSpec{PodIdentity: &AuthPodIdentity{Provider: PodIdentityProviderGCPWorkloadServiceAccount}},
				wantMessage: "workload service account providers require serviceAccountName",
			},
			{
				name: "sa-name-wrong-provider",
				spec: TriggerAuthenticationSpec{PodIdentity: &AuthPodIdentity{
					Provider:           PodIdentityProviderGCP,
					ServiceAccountName: ptr.To("scaling-reader"),
				}},
				wantMessage: "serviceAccountName requires a workload service account provider",
			},
			{
				name: "nested-azure-kv",
				spec: TriggerAuthenticationSpec{AzureKeyVault: &AzureKeyVault{
					VaultURI:    "https://example.vault.azure.net",
					Secrets:     []AzureKeyVaultSecret{{Parameter: "p", Name: "s"}},
					PodIdentity: selected,
				}},
				wantMessage: "azureKeyVault.podIdentity does not support workload service account selection",
			},
			{
				name: "nested-aws-sm",
				spec: TriggerAuthenticationSpec{AwsSecretManager: &AwsSecretManager{
					Secrets:     []AwsSecretManagerSecret{{Parameter: "p", Name: "s"}},
					PodIdentity: selected,
				}},
				wantMessage: "awsSecretManager.podIdentity does not support workload service account selection",
			},
			{
				name: "nested-gcp-sm",
				spec: TriggerAuthenticationSpec{GCPSecretManager: &GCPSecretManager{
					Secrets:     []GCPSecretManagerSecret{{Parameter: "p", ID: "s"}},
					PodIdentity: selected,
				}},
				wantMessage: "gcpSecretManager.podIdentity does not support workload service account selection",
			},
			{
				name: "azure-sp-conflict",
				spec: TriggerAuthenticationSpec{
					PodIdentity: selected,
					AzureServicePrincipal: &AzureServicePrincipal{
						TenantID: "tenant",
						ClientID: "client",
						ClientSecret: &AzureServicePrincipalCredential{
							ValueFrom: ValueFromSecret{SecretKeyRef: SecretKeyRef{Name: "secret", Key: "key"}},
						},
					},
				},
				wantMessage: "podIdentity.serviceAccountName cannot be combined with azureServicePrincipal",
			},
		}

		for _, tc := range celRejections {
			ta := &TriggerAuthentication{
				ObjectMeta: metav1.ObjectMeta{Name: tc.name, Namespace: namespaceName},
				Spec:       tc.spec,
			}
			err := k8sClient.Create(context.Background(), ta)
			Expect(err).To(HaveOccurred(), tc.name)
			Expect(err.Error()).To(ContainSubstring(tc.wantMessage), tc.name)
		}

		invalidCTA := &ClusterTriggerAuthentication{
			ObjectMeta: metav1.ObjectMeta{Name: "invalid-gcp-sa-cta"},
			Spec:       TriggerAuthenticationSpec{PodIdentity: &AuthPodIdentity{Provider: PodIdentityProviderGCPWorkloadServiceAccount}},
		}
		err := k8sClient.Create(context.Background(), invalidCTA)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("workload service account providers require serviceAccountName"))
	})
})
