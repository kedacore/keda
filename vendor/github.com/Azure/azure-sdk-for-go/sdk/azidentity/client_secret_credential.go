//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azidentity

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
)

const credNameSecret = "ClientSecretCredential"

// ClientSecretCredentialOptions contains optional parameters for ClientSecretCredential.
type ClientSecretCredentialOptions struct {
	azcore.ClientOptions

	// AdditionallyAllowedTenants specifies additional tenants for which the credential may acquire tokens.
	// Add the wildcard value "*" to allow the credential to acquire tokens for any tenant in which the
	// application is registered.
	AdditionallyAllowedTenants []string
	// DisableInstanceDiscovery allows disconnected cloud solutions to skip instance discovery for unknown authority hosts.
	DisableInstanceDiscovery bool
}

// ClientSecretCredential authenticates an application with a client secret.
type ClientSecretCredential struct {
	additionallyAllowedTenants []string
	client                     confidentialClient
	tenant                     string
}

// NewClientSecretCredential constructs a ClientSecretCredential. Pass nil for options to accept defaults.
func NewClientSecretCredential(tenantID string, clientID string, clientSecret string, options *ClientSecretCredentialOptions) (*ClientSecretCredential, error) {
	if options == nil {
		options = &ClientSecretCredentialOptions{}
	}
	cred, err := confidential.NewCredFromSecret(clientSecret)
	if err != nil {
		return nil, err
	}
	c, err := getConfidentialClient(clientID, tenantID, cred, &options.ClientOptions, confidential.WithInstanceDiscovery(!options.DisableInstanceDiscovery))
	if err != nil {
		return nil, err
	}
	return &ClientSecretCredential{
		additionallyAllowedTenants: resolveAdditionallyAllowedTenants(options.AdditionallyAllowedTenants),
		client:                     c,
		tenant:                     tenantID,
	}, nil
}

// GetToken requests an access token from Azure Active Directory. This method is called automatically by Azure SDK clients.
func (c *ClientSecretCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if len(opts.Scopes) == 0 {
		return azcore.AccessToken{}, errors.New(credNameSecret + ": GetToken() requires at least one scope")
	}
	tenant, err := resolveTenant(c.tenant, opts.TenantID, c.additionallyAllowedTenants)
	if err != nil {
		return azcore.AccessToken{}, err
	}
	ar, err := c.client.AcquireTokenSilent(ctx, opts.Scopes, confidential.WithClaims(opts.Claims), confidential.WithTenantID(tenant))
	if err == nil {
		logGetTokenSuccess(c, opts)
		return azcore.AccessToken{Token: ar.AccessToken, ExpiresOn: ar.ExpiresOn.UTC()}, err
	}

	ar, err = c.client.AcquireTokenByCredential(ctx, opts.Scopes, confidential.WithClaims(opts.Claims), confidential.WithTenantID(tenant))
	if err != nil {
		return azcore.AccessToken{}, newAuthenticationFailedErrorFromMSALError(credNameSecret, err)
	}
	logGetTokenSuccess(c, opts)
	return azcore.AccessToken{Token: ar.AccessToken, ExpiresOn: ar.ExpiresOn.UTC()}, err
}

var _ azcore.TokenCredential = (*ClientSecretCredential)(nil)
