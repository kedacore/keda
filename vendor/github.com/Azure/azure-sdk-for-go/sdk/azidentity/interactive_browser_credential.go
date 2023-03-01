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
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

const credNameBrowser = "InteractiveBrowserCredential"

// InteractiveBrowserCredentialOptions contains optional parameters for InteractiveBrowserCredential.
type InteractiveBrowserCredentialOptions struct {
	azcore.ClientOptions

	// AdditionallyAllowedTenants specifies additional tenants for which the credential may acquire
	// tokens. Add the wildcard value "*" to allow the credential to acquire tokens for any tenant.
	AdditionallyAllowedTenants []string
	// ClientID is the ID of the application users will authenticate to.
	// Defaults to the ID of an Azure development application.
	ClientID string

	// DisableInstanceDiscovery allows disconnected cloud solutions to skip instance discovery for unknown authority hosts.
	DisableInstanceDiscovery bool

	// LoginHint pre-populates the account prompt with a username. Users may choose to authenticate a different account.
	LoginHint string
	// RedirectURL is the URL Azure Active Directory will redirect to with the access token. This is required
	// only when setting ClientID, and must match a redirect URI in the application's registration.
	// Applications which have registered "http://localhost" as a redirect URI need not set this option.
	RedirectURL string

	// TenantID is the Azure Active Directory tenant the credential authenticates in. Defaults to the
	// "organizations" tenant, which can authenticate work and school accounts.
	TenantID string
}

func (o *InteractiveBrowserCredentialOptions) init() {
	if o.TenantID == "" {
		o.TenantID = organizationsTenantID
	}
	if o.ClientID == "" {
		o.ClientID = developerSignOnClientID
	}
}

// InteractiveBrowserCredential opens a browser to interactively authenticate a user.
type InteractiveBrowserCredential struct {
	account                    public.Account
	additionallyAllowedTenants []string
	client                     publicClient
	options                    InteractiveBrowserCredentialOptions
}

// NewInteractiveBrowserCredential constructs a new InteractiveBrowserCredential. Pass nil to accept default options.
func NewInteractiveBrowserCredential(options *InteractiveBrowserCredentialOptions) (*InteractiveBrowserCredential, error) {
	cp := InteractiveBrowserCredentialOptions{}
	if options != nil {
		cp = *options
	}
	cp.init()
	c, err := getPublicClient(cp.ClientID, cp.TenantID, &cp.ClientOptions, public.WithInstanceDiscovery(!cp.DisableInstanceDiscovery))
	if err != nil {
		return nil, err
	}
	return &InteractiveBrowserCredential{
		additionallyAllowedTenants: resolveAdditionallyAllowedTenants(cp.AdditionallyAllowedTenants),
		client:                     c,
		options:                    cp,
	}, nil
}

// GetToken requests an access token from Azure Active Directory. This method is called automatically by Azure SDK clients.
func (c *InteractiveBrowserCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if len(opts.Scopes) == 0 {
		return azcore.AccessToken{}, errors.New(credNameBrowser + ": GetToken() requires at least one scope")
	}
	tenant, err := resolveTenant(c.options.TenantID, opts.TenantID, c.additionallyAllowedTenants)
	if err != nil {
		return azcore.AccessToken{}, err
	}
	ar, err := c.client.AcquireTokenSilent(ctx, opts.Scopes,
		public.WithClaims(opts.Claims),
		public.WithSilentAccount(c.account),
		public.WithTenantID(tenant),
	)
	if err == nil {
		logGetTokenSuccess(c, opts)
		return azcore.AccessToken{Token: ar.AccessToken, ExpiresOn: ar.ExpiresOn.UTC()}, err
	}

	ar, err = c.client.AcquireTokenInteractive(ctx, opts.Scopes,
		public.WithClaims(opts.Claims),
		public.WithLoginHint(c.options.LoginHint),
		public.WithRedirectURI(c.options.RedirectURL),
		public.WithTenantID(tenant),
	)
	if err != nil {
		return azcore.AccessToken{}, newAuthenticationFailedErrorFromMSALError(credNameBrowser, err)
	}
	c.account = ar.Account
	logGetTokenSuccess(c, opts)
	return azcore.AccessToken{Token: ar.AccessToken, ExpiresOn: ar.ExpiresOn.UTC()}, err
}

var _ azcore.TokenCredential = (*InteractiveBrowserCredential)(nil)
