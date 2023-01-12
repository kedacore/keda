//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azidentity

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/internal/log"
)

// DefaultAzureCredentialOptions contains optional parameters for DefaultAzureCredential.
// These options may not apply to all credentials in the chain.
type DefaultAzureCredentialOptions struct {
	azcore.ClientOptions

	// TenantID identifies the tenant the Azure CLI should authenticate in.
	// Defaults to the CLI's default tenant, which is typically the home tenant of the user logged in to the CLI.
	TenantID string
}

// DefaultAzureCredential is a default credential chain for applications that will deploy to Azure.
// It combines credentials suitable for deployment with credentials suitable for local development.
// It attempts to authenticate with each of these credential types, in the following order, stopping
// when one provides a token:
//
//   - [EnvironmentCredential]
//   - [WorkloadIdentityCredential], if environment variable configuration is set by the Azure workload
//     identity webhook. Use [WorkloadIdentityCredential] directly when not using the webhook or needing
//     more control over its configuration.
//   - [ManagedIdentityCredential]
//   - [AzureCLICredential]
//
// Consult the documentation for these credential types for more information on how they authenticate.
// Once a credential has successfully authenticated, DefaultAzureCredential will use that credential for
// every subsequent authentication.
type DefaultAzureCredential struct {
	chain *ChainedTokenCredential
}

// NewDefaultAzureCredential creates a DefaultAzureCredential. Pass nil for options to accept defaults.
func NewDefaultAzureCredential(options *DefaultAzureCredentialOptions) (*DefaultAzureCredential, error) {
	var creds []azcore.TokenCredential
	var errorMessages []string

	if options == nil {
		options = &DefaultAzureCredentialOptions{}
	}

	envCred, err := NewEnvironmentCredential(&EnvironmentCredentialOptions{ClientOptions: options.ClientOptions})
	if err == nil {
		creds = append(creds, envCred)
	} else {
		errorMessages = append(errorMessages, "EnvironmentCredential: "+err.Error())
		creds = append(creds, &defaultCredentialErrorReporter{credType: "EnvironmentCredential", err: err})
	}

	// workload identity requires values for AZURE_AUTHORITY_HOST, AZURE_CLIENT_ID, AZURE_FEDERATED_TOKEN_FILE, AZURE_TENANT_ID
	haveWorkloadConfig := false
	clientID, haveClientID := os.LookupEnv(azureClientID)
	if haveClientID {
		if file, ok := os.LookupEnv(azureFederatedTokenFile); ok {
			if _, ok := os.LookupEnv(azureAuthorityHost); ok {
				if tenantID, ok := os.LookupEnv(azureTenantID); ok {
					haveWorkloadConfig = true
					workloadCred, err := NewWorkloadIdentityCredential(tenantID, clientID, file, &WorkloadIdentityCredentialOptions{
						ClientOptions: options.ClientOptions},
					)
					if err == nil {
						creds = append(creds, workloadCred)
					} else {
						errorMessages = append(errorMessages, credNameWorkloadIdentity+": "+err.Error())
						creds = append(creds, &defaultCredentialErrorReporter{credType: credNameWorkloadIdentity, err: err})
					}
				}
			}
		}
	}
	if !haveWorkloadConfig {
		err := errors.New("missing environment variables for workload identity. Check webhook and pod configuration")
		creds = append(creds, &defaultCredentialErrorReporter{credType: credNameWorkloadIdentity, err: err})
	}

	o := &ManagedIdentityCredentialOptions{ClientOptions: options.ClientOptions}
	if haveClientID {
		o.ID = ClientID(clientID)
	}
	msiCred, err := NewManagedIdentityCredential(o)
	if err == nil {
		creds = append(creds, msiCred)
		msiCred.mic.imdsTimeout = time.Second
	} else {
		errorMessages = append(errorMessages, credNameManagedIdentity+": "+err.Error())
		creds = append(creds, &defaultCredentialErrorReporter{credType: credNameManagedIdentity, err: err})
	}

	cliCred, err := NewAzureCLICredential(&AzureCLICredentialOptions{TenantID: options.TenantID})
	if err == nil {
		creds = append(creds, cliCred)
	} else {
		errorMessages = append(errorMessages, credNameAzureCLI+": "+err.Error())
		creds = append(creds, &defaultCredentialErrorReporter{credType: credNameAzureCLI, err: err})
	}

	err = defaultAzureCredentialConstructorErrorHandler(len(creds), errorMessages)
	if err != nil {
		return nil, err
	}

	chain, err := NewChainedTokenCredential(creds, nil)
	if err != nil {
		return nil, err
	}
	chain.name = "DefaultAzureCredential"
	return &DefaultAzureCredential{chain: chain}, nil
}

// GetToken requests an access token from Azure Active Directory. This method is called automatically by Azure SDK clients.
func (c *DefaultAzureCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return c.chain.GetToken(ctx, opts)
}

var _ azcore.TokenCredential = (*DefaultAzureCredential)(nil)

func defaultAzureCredentialConstructorErrorHandler(numberOfSuccessfulCredentials int, errorMessages []string) (err error) {
	errorMessage := strings.Join(errorMessages, "\n\t")

	if numberOfSuccessfulCredentials == 0 {
		return errors.New(errorMessage)
	}

	if len(errorMessages) != 0 {
		log.Writef(EventAuthentication, "NewDefaultAzureCredential failed to initialize some credentials:\n\t%s", errorMessage)
	}

	return nil
}

// defaultCredentialErrorReporter is a substitute for credentials that couldn't be constructed.
// Its GetToken method always returns a credentialUnavailableError having the same message as
// the error that prevented constructing the credential. This ensures the message is present
// in the error returned by ChainedTokenCredential.GetToken()
type defaultCredentialErrorReporter struct {
	credType string
	err      error
}

func (d *defaultCredentialErrorReporter) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if _, ok := d.err.(*credentialUnavailableError); ok {
		return azcore.AccessToken{}, d.err
	}
	return azcore.AccessToken{}, newCredentialUnavailableError(d.credType, d.err.Error())
}

var _ azcore.TokenCredential = (*defaultCredentialErrorReporter)(nil)
