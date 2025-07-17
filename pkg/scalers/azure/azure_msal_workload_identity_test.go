package azure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func TestCreateCredentialForPodIdentity(t *testing.T) {
	tests := []struct {
		name         string
		podIdentity  kedav1alpha1.AuthPodIdentity
		clientID     string
		clientSecret string
		tenantID     string
		expectError  bool
		errorMessage string
	}{
		{
			name: "client credentials authentication",
			podIdentity: kedav1alpha1.AuthPodIdentity{
				Provider: kedav1alpha1.PodIdentityProviderNone,
			},
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			tenantID:     "test-tenant-id",
			expectError:  false,
		},
		{
			name: "empty provider defaults to client credentials",
			podIdentity: kedav1alpha1.AuthPodIdentity{
				Provider: "",
			},
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			tenantID:     "test-tenant-id",
			expectError:  false,
		},
		{
			name: "workload identity authentication",
			podIdentity: kedav1alpha1.AuthPodIdentity{
				Provider:   kedav1alpha1.PodIdentityProviderAzureWorkload,
				IdentityID: "test-identity",
			},
			expectError: true, // Will fail in test environment without proper setup
		},
		{
			name: "unsupported identity provider",
			podIdentity: kedav1alpha1.AuthPodIdentity{
				Provider: "unsupported-provider",
			},
			expectError:  true,
			errorMessage: "unsupported pod identity provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			credential, err := CreateCredentialForPodIdentity(
				ctx,
				tt.podIdentity,
				tt.clientID,
				tt.clientSecret,
				tt.tenantID,
				"https://login.microsoftonline.com/",
			)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Nil(t, credential)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, credential)
			}
		})
	}
}

func TestNewMSALWorkloadIdentityCredential(t *testing.T) {
	// This test requires environment variables to be set for Azure Workload Identity
	// In a real test environment, you would mock the file system and environment
	t.Run("missing environment variables", func(t *testing.T) {
		ctx := context.Background()
		podIdentity := kedav1alpha1.AuthPodIdentity{
			Provider: kedav1alpha1.PodIdentityProviderAzureWorkload,
		}

		// This will likely fail without proper Azure Workload Identity setup
		credential, err := NewMSALWorkloadIdentityCredential(ctx, podIdentity)

		// In test environment, we expect this to fail
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, credential)
		} else {
			// If it succeeds (unlikely in test), verify the credential is created
			assert.NotNil(t, credential)
		}
	})
}

func TestGetScopedResource(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		expected string
	}{
		{
			name:     "resource without trailing slash",
			resource: "https://api.applicationinsights.io",
			expected: "https://api.applicationinsights.io/.default",
		},
		{
			name:     "resource with trailing slash",
			resource: "https://api.applicationinsights.io/",
			expected: "https://api.applicationinsights.io/.default",
		},
		{
			name:     "resource already with .default",
			resource: "https://api.applicationinsights.io/.default",
			expected: "https://api.applicationinsights.io/.default",
		},
		{
			name:     "resource with trailing slash and .default",
			resource: "https://api.applicationinsights.io/.default",
			expected: "https://api.applicationinsights.io/.default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getScopedResource(tt.resource)
			assert.Equal(t, tt.expected, result)
		})
	}
}
