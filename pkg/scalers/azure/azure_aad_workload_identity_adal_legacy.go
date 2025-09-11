/*
Copyright 2022 The KEDA Authors

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

package azure

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	amqpAuth "github.com/Azure/azure-amqp-common-go/v4/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
)

// Azure AD Workload Identity Webhook will inject the following environment variables.
// * AZURE_CLIENT_ID - Client id set in the service account annotation
// * AZURE_TENANT_ID - Tenant id set in the service account annotation. If not defined, then tenant id provided via
// azure-wi-webhook-config will be used.
// * AZURE_FEDERATED_TOKEN_FILE - Service account token file path
// * AZURE_AUTHORITY_HOST -  Azure Active Directory (AAD) endpoint.
const (
	azureClientIDEnv           = "AZURE_CLIENT_ID"
	azureTenantIDEnv           = "AZURE_TENANT_ID"
	azureFederatedTokenFileEnv = "AZURE_FEDERATED_TOKEN_FILE"
	azureAuthorityHostEnv      = "AZURE_AUTHORITY_HOST"
)

var DefaultClientID string
var DefaultTenantID string
var TokenFilePath string
var DefaultAuthorityHost string

func init() {
	DefaultClientID = os.Getenv(azureClientIDEnv)
	DefaultTenantID = os.Getenv(azureTenantIDEnv)
	TokenFilePath = os.Getenv(azureFederatedTokenFileEnv)
	DefaultAuthorityHost = os.Getenv(azureAuthorityHostEnv)
}

// GetAzureADWorkloadIdentityToken returns the AADToken for resource
func GetAzureADWorkloadIdentityToken(ctx context.Context, identityID, identityTenantID, identityAuthorityHost, resource string) (AADToken, error) {
	clientID := DefaultClientID
	tenantID := DefaultTenantID
	authorityHost := DefaultAuthorityHost

	if identityID != "" {
		clientID = identityID
	}

	if identityTenantID != "" {
		tenantID = identityTenantID

		// override the authority host only if provided and tenant id is provided
		if identityAuthorityHost != "" {
			authorityHost = identityAuthorityHost
		}
	}

	signedAssertion, err := readJWTFromFileSystem(TokenFilePath)
	if err != nil {
		return AADToken{}, fmt.Errorf("error reading service account token - %w", err)
	}

	if signedAssertion == "" {
		return AADToken{}, fmt.Errorf("assertion can't be empty string")
	}

	cred := confidential.NewCredFromAssertionCallback(func(context.Context, confidential.AssertionRequestOptions) (string, error) {
		return signedAssertion, nil
	})

	confidentialClient, err := confidential.New(
		fmt.Sprintf("%s%s", authorityHost, tenantID),
		clientID,
		cred,
	)
	if err != nil {
		return AADToken{}, fmt.Errorf("error creating confidential client - %w", err)
	}

	result, err := confidentialClient.AcquireTokenByCredential(ctx, []string{getScopedResource(resource)})
	if err != nil {
		return AADToken{}, fmt.Errorf("error acquiring aad token - %w", err)
	}

	return AADToken{
		AccessToken:         result.AccessToken,
		ExpiresOn:           strconv.FormatInt(result.ExpiresOn.Unix(), 10),
		ExpiresOnTimeObject: result.ExpiresOn,
		GrantedScopes:       result.GrantedScopes,
		DeclinedScopes:      result.DeclinedScopes,
	}, nil
}

func readJWTFromFileSystem(tokenFilePath string) (string, error) {
	token, err := os.ReadFile(tokenFilePath)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

func getScopedResource(resource string) string {
	resource = strings.TrimSuffix(resource, "/")
	if !strings.HasSuffix(resource, ".default") {
		resource += "/.default"
	}

	return resource
}

type ADWorkloadIdentityConfig struct {
	ctx                   context.Context
	IdentityID            string
	IdentityTenantID      string
	IdentityAuthorityHost string
	Resource              string
}

func NewAzureADWorkloadIdentityConfig(ctx context.Context, identityID, identityTenantID, identityAuthorityHost, resource string) auth.AuthorizerConfig {
	return ADWorkloadIdentityConfig{ctx: ctx, IdentityID: identityID, IdentityTenantID: identityTenantID, IdentityAuthorityHost: identityAuthorityHost, Resource: resource}
}

// Authorizer implements the auth.AuthorizerConfig interface
func (aadWiConfig ADWorkloadIdentityConfig) Authorizer() (autorest.Authorizer, error) {
	return autorest.NewBearerAuthorizer(NewAzureADWorkloadIdentityTokenProvider(
		aadWiConfig.ctx, aadWiConfig.IdentityID, aadWiConfig.IdentityTenantID, aadWiConfig.IdentityAuthorityHost, aadWiConfig.Resource)), nil
}

func NewADWorkloadIdentityCredential(identityID, identityTenantID string) (*azidentity.WorkloadIdentityCredential, error) {
	options := &azidentity.WorkloadIdentityCredentialOptions{}
	if identityID != "" {
		options.ClientID = identityID
	}
	if identityTenantID != "" {
		options.TenantID = identityTenantID
	}
	return azidentity.NewWorkloadIdentityCredential(options)
}

// ADWorkloadIdentityTokenProvider is a type that implements the adal.OAuthTokenProvider and adal.Refresher interfaces.
// The OAuthTokenProvider interface is used by the BearerAuthorizer to get the token when preparing the HTTP Header.
// The Refresher interface is used by the BearerAuthorizer to refresh the token.
type ADWorkloadIdentityTokenProvider struct {
	ctx                   context.Context
	IdentityID            string
	IdentityTenantID      string
	IdentityAuthorityHost string
	Resource              string
	aadToken              AADToken
}

func NewAzureADWorkloadIdentityTokenProvider(ctx context.Context, identityID, identityTenantID, identityAuthorityHost, resource string) *ADWorkloadIdentityTokenProvider {
	return &ADWorkloadIdentityTokenProvider{ctx: ctx, IdentityID: identityID, IdentityTenantID: identityTenantID, IdentityAuthorityHost: identityAuthorityHost, Resource: resource}
}

// OAuthToken is for implementing the adal.OAuthTokenProvider interface. It returns the current access token.
func (wiTokenProvider *ADWorkloadIdentityTokenProvider) OAuthToken() string {
	return wiTokenProvider.aadToken.AccessToken
}

// Refresh is for implementing the adal.Refresher interface
func (wiTokenProvider *ADWorkloadIdentityTokenProvider) Refresh() error {
	if time.Now().Before(wiTokenProvider.aadToken.ExpiresOnTimeObject) {
		return nil
	}

	aadToken, err := GetAzureADWorkloadIdentityToken(wiTokenProvider.ctx, wiTokenProvider.IdentityID, wiTokenProvider.IdentityTenantID, wiTokenProvider.IdentityAuthorityHost, wiTokenProvider.Resource)
	if err != nil {
		return err
	}

	wiTokenProvider.aadToken = aadToken
	return nil
}

// RefreshExchange is for implementing the adal.Refresher interface
func (wiTokenProvider *ADWorkloadIdentityTokenProvider) RefreshExchange(resource string) error {
	wiTokenProvider.Resource = resource
	return wiTokenProvider.Refresh()
}

// EnsureFresh is for implementing the adal.Refresher interface
func (wiTokenProvider *ADWorkloadIdentityTokenProvider) EnsureFresh() error {
	return wiTokenProvider.Refresh()
}

// GetToken is for implementing the auth.TokenProvider interface
func (wiTokenProvider *ADWorkloadIdentityTokenProvider) GetToken(_ string) (*amqpAuth.Token, error) {
	err := wiTokenProvider.Refresh()
	if err != nil {
		return nil, err
	}

	return amqpAuth.NewToken(amqpAuth.CBSTokenTypeJWT, wiTokenProvider.aadToken.AccessToken,
		wiTokenProvider.aadToken.ExpiresOn), nil
}
