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

// GetAzureADWorkloadIdentityToken returns the AADToken for resource
func GetAzureADWorkloadIdentityToken(ctx context.Context, resource string) (AADToken, error) {
	clientID := os.Getenv(azureClientIDEnv)
	tenantID := os.Getenv(azureTenantIDEnv)
	tokenFilePath := os.Getenv(azureFederatedTokenFileEnv)
	authorityHost := os.Getenv(azureAuthrityHostEnv)

	signedAssertion, err := readJWTFromFileSystem(tokenFilePath)
	if err != nil {
		return AADToken{}, fmt.Errorf("error reading service account token - %w", err)
	}

	cred, err := confidential.NewCredFromAssertion(signedAssertion)
	if err != nil {
		return AADToken{}, fmt.Errorf("error getting credentials from service account token - %w", err)
	}

	authorityOption := confidential.WithAuthority(fmt.Sprintf("%s%s/oauth2/token", authorityHost, tenantID))
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
		AccessToken:    result.AccessToken,
		ExpiresOn:      strconv.FormatInt(result.ExpiresOn.Unix(), 10),
		GrantedScopes:  result.GrantedScopes,
		DeclinedScopes: result.DeclinedScopes,
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
