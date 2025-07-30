package azure

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// MSALWorkloadIdentityCredential provides Azure AD Workload Identity authentication using pure MSAL
type MSALWorkloadIdentityCredential struct {
	ctx                context.Context
	clientID           string
	tenantID           string
	authorityHost      string
	tokenFilePath      string
	confidentialClient confidential.Client
}

// NewMSALWorkloadIdentityCredential creates a new MSAL-based workload identity credential
func NewMSALWorkloadIdentityCredential(ctx context.Context, podIdentity kedav1alpha1.AuthPodIdentity) (*MSALWorkloadIdentityCredential, error) {
	clientID := DefaultClientID
	tenantID := DefaultTenantID
	authorityHost := DefaultAuthorityHost
	tokenFilePath := TokenFilePath

	if identityID := podIdentity.GetIdentityID(); identityID != "" {
		clientID = identityID
	}
	if identityTenantID := podIdentity.GetIdentityTenantID(); identityTenantID != "" {
		tenantID = identityTenantID
		if identityAuthorityHost := podIdentity.GetIdentityAuthorityHost(); identityAuthorityHost != "" {
			authorityHost = identityAuthorityHost
		}
	}

	cred := confidential.NewCredFromAssertionCallback(func(context.Context, confidential.AssertionRequestOptions) (string, error) {
		token, err := os.ReadFile(tokenFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to read token file: %w", err)
		}
		return string(token), nil
	})

	confidentialClient, err := confidential.New(
		fmt.Sprintf("%s%s", authorityHost, tenantID),
		clientID,
		cred,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create confidential client: %w", err)
	}

	return &MSALWorkloadIdentityCredential{
		ctx:                ctx,
		clientID:           clientID,
		tenantID:           tenantID,
		authorityHost:      authorityHost,
		tokenFilePath:      tokenFilePath,
		confidentialClient: confidentialClient,
	}, nil
}

// GetToken implements azcore.TokenCredential interface
func (c *MSALWorkloadIdentityCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	scopes := options.Scopes
	if len(scopes) == 0 {
		return azcore.AccessToken{}, fmt.Errorf("no scopes provided")
	}

	// Ensure scopes are properly formatted for MSAL
	for i, scope := range scopes {
		scopes[i] = getScopedResource(scope)
	}

	result, err := c.confidentialClient.AcquireTokenByCredential(ctx, scopes)
	if err != nil {
		return azcore.AccessToken{}, fmt.Errorf("failed to acquire token: %w", err)
	}

	return azcore.AccessToken{
		Token:     result.AccessToken,
		ExpiresOn: result.ExpiresOn,
	}, nil
}

// CreateCredentialForPodIdentity creates appropriate credential based on pod identity provider
func CreateCredentialForPodIdentity(ctx context.Context, podIdentity kedav1alpha1.AuthPodIdentity, clientID, clientSecret, tenantID, authorityHost string) (azcore.TokenCredential, error) {
	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// Use client secret credential
		options := &azidentity.ClientSecretCredentialOptions{}
		if authorityHost != "" {
			options.AuthorityHost = authorityHost
		}
		return azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, options)

	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		// Use Azure Workload Identity with azidentity (preferred for most cases)
		options := &azidentity.WorkloadIdentityCredentialOptions{}
		if identityID := podIdentity.GetIdentityID(); identityID != "" {
			options.ClientID = identityID
		}
		if identityTenantID := podIdentity.GetIdentityTenantID(); identityTenantID != "" {
			options.TenantID = identityTenantID
		}
		return azidentity.NewWorkloadIdentityCredential(options)

	default:
		return nil, fmt.Errorf("unsupported pod identity provider: %s", podIdentity.Provider)
	}
}
