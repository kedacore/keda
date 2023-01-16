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

	amqpAuth "github.com/Azure/azure-amqp-common-go/v3/auth"
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
	azureAuthrityHostEnv       = "AZURE_AUTHORITY_HOST"
)

var DefaultClientID string
var TenantID string
var TokenFilePath string
var AuthorityHost string

func init() {
	DefaultClientID = os.Getenv(azureClientIDEnv)
	TenantID = os.Getenv(azureTenantIDEnv)
	TokenFilePath = os.Getenv(azureFederatedTokenFileEnv)
	AuthorityHost = os.Getenv(azureAuthrityHostEnv)
}

// GetAzureADWorkloadIdentityToken returns the AADToken for resource
func GetAzureADWorkloadIdentityToken(ctx context.Context, identityID, resource string) (AADToken, error) {
	clientID := DefaultClientID
	if identityID != "" {
		clientID = identityID
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

	authorityOption := confidential.WithAuthority(fmt.Sprintf("%s%s/oauth2/token", AuthorityHost, TenantID))
	confidentialClient, err := confidential.New(
		clientID,
		cred,
		authorityOption,
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
	ctx        context.Context
	IdentityID string
	Resource   string
}

func NewAzureADWorkloadIdentityConfig(ctx context.Context, identityID, resource string) auth.AuthorizerConfig {
	return ADWorkloadIdentityConfig{ctx: ctx, IdentityID: identityID, Resource: resource}
}

// Authorizer implements the auth.AuthorizerConfig interface
func (aadWiConfig ADWorkloadIdentityConfig) Authorizer() (autorest.Authorizer, error) {
	return autorest.NewBearerAuthorizer(NewAzureADWorkloadIdentityTokenProvider(
		aadWiConfig.ctx, aadWiConfig.IdentityID, aadWiConfig.Resource)), nil
}

func NewADWorkloadIdentityCredential(identityID string) (*azidentity.WorkloadIdentityCredential, error) {
	clientID := DefaultClientID
	if identityID != "" {
		clientID = identityID
	}
	return azidentity.NewWorkloadIdentityCredential(TenantID, clientID, TokenFilePath, &azidentity.WorkloadIdentityCredentialOptions{})
}

// ADWorkloadIdentityTokenProvider is a type that implements the adal.OAuthTokenProvider and adal.Refresher interfaces.
// The OAuthTokenProvider interface is used by the BearerAuthorizer to get the token when preparing the HTTP Header.
// The Refresher interface is used by the BearerAuthorizer to refresh the token.
type ADWorkloadIdentityTokenProvider struct {
	ctx        context.Context
	IdentityID string
	Resource   string
	aadToken   AADToken
}

func NewAzureADWorkloadIdentityTokenProvider(ctx context.Context, identityID, resource string) *ADWorkloadIdentityTokenProvider {
	return &ADWorkloadIdentityTokenProvider{ctx: ctx, IdentityID: identityID, Resource: resource}
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

	aadToken, err := GetAzureADWorkloadIdentityToken(wiTokenProvider.ctx, wiTokenProvider.IdentityID, wiTokenProvider.Resource)
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
func (wiTokenProvider *ADWorkloadIdentityTokenProvider) GetToken(uri string) (*amqpAuth.Token, error) {
	err := wiTokenProvider.Refresh()
	if err != nil {
		return nil, err
	}

	return amqpAuth.NewToken(amqpAuth.CBSTokenTypeJWT, wiTokenProvider.aadToken.AccessToken,
		wiTokenProvider.aadToken.ExpiresOn), nil
}
