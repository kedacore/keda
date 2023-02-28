//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azidentity

import (
	"context"
	"crypto"
	"crypto/x509"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
)

const credNameOBO = "OnBehalfOfCredential"

// OnBehalfOfCredential authenticates a service principal via the on-behalf-of flow. This is typically used by
// middle-tier services that authorize requests to other services with a delegated user identity. Because this
// is not an interactive authentication flow, an application using it must have admin consent for any delegated
// permissions before requesting tokens for them. See [Azure Active Directory documentation] for more details.
//
// [Azure Active Directory documentation]: https://docs.microsoft.com/azure/active-directory/develop/v2-oauth2-on-behalf-of-flow
type OnBehalfOfCredential struct {
	assertion string
	client    confidentialClient
}

// OnBehalfOfCredentialOptions contains optional parameters for OnBehalfOfCredential
type OnBehalfOfCredentialOptions struct {
	azcore.ClientOptions

	// SendCertificateChain applies only when the credential is configured to authenticate with a certificate.
	// This setting controls whether the credential sends the public certificate chain in the x5c header of each
	// token request's JWT. This is required for, and only used in, Subject Name/Issuer (SNI) authentication.
	SendCertificateChain bool
}

// NewOnBehalfOfCredentialFromCertificate constructs an OnBehalfOfCredential that authenticates with a certificate.
// See [ParseCertificates] for help loading a certificate.
func NewOnBehalfOfCredentialFromCertificate(tenantID, clientID, userAssertion string, certs []*x509.Certificate, key crypto.PrivateKey, options *OnBehalfOfCredentialOptions) (*OnBehalfOfCredential, error) {
	cred, err := confidential.NewCredFromCertChain(certs, key)
	if err != nil {
		return nil, err
	}
	return newOnBehalfOfCredential(tenantID, clientID, userAssertion, cred, options)
}

// NewOnBehalfOfCredentialFromSecret constructs an OnBehalfOfCredential that authenticates with a client secret.
func NewOnBehalfOfCredentialFromSecret(tenantID, clientID, userAssertion, clientSecret string, options *OnBehalfOfCredentialOptions) (*OnBehalfOfCredential, error) {
	cred, err := confidential.NewCredFromSecret(clientSecret)
	if err != nil {
		return nil, err
	}
	return newOnBehalfOfCredential(tenantID, clientID, userAssertion, cred, options)
}

func newOnBehalfOfCredential(tenantID, clientID, userAssertion string, cred confidential.Credential, options *OnBehalfOfCredentialOptions) (*OnBehalfOfCredential, error) {
	if options == nil {
		options = &OnBehalfOfCredentialOptions{}
	}
	opts := []confidential.Option{}
	if options.SendCertificateChain {
		opts = append(opts, confidential.WithX5C())
	}
	c, err := getConfidentialClient(clientID, tenantID, cred, &options.ClientOptions, opts...)
	if err != nil {
		return nil, err
	}
	return &OnBehalfOfCredential{assertion: userAssertion, client: c}, nil
}

// GetToken requests an access token from Azure Active Directory. This method is called automatically by Azure SDK clients.
func (o *OnBehalfOfCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if len(opts.Scopes) == 0 {
		return azcore.AccessToken{}, errors.New(credNameSecret + ": GetToken() requires at least one scope")
	}
	ar, err := o.client.AcquireTokenOnBehalfOf(ctx, o.assertion, opts.Scopes)
	if err != nil {
		return azcore.AccessToken{}, newAuthenticationFailedErrorFromMSALError(credNameOBO, err)
	}
	logGetTokenSuccess(o, opts)
	return azcore.AccessToken{Token: ar.AccessToken, ExpiresOn: ar.ExpiresOn.UTC()}, nil
}
