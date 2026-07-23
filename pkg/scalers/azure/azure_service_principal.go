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

package azure

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const (
	// ServicePrincipalAuthKey marks authentication parameters resolved from
	// TriggerAuthentication.spec.azureServicePrincipal.
	ServicePrincipalAuthKey = "azureServicePrincipal"

	ServicePrincipalTenantIDKey                  = "azureServicePrincipalTenantId"
	ServicePrincipalClientIDKey                  = "azureServicePrincipalClientId"
	ServicePrincipalCloudKey                     = "azureServicePrincipalCloud"
	ServicePrincipalActiveDirectoryEndpointKey   = "azureServicePrincipalActiveDirectoryEndpoint"
	ServicePrincipalClientSecretKey              = "azureServicePrincipalClientSecret"
	ServicePrincipalClientCertificateKey         = "azureServicePrincipalClientCertificate"
	ServicePrincipalClientCertificatePasswordKey = "azureServicePrincipalClientCertificatePassword"
)

// IsServicePrincipalAuthConfigured reports whether authParams were resolved
// from an Azure service principal authentication provider.
func IsServicePrincipalAuthConfigured(authParams map[string]string) bool {
	return authParams[ServicePrincipalAuthKey] == "true"
}

// NewServicePrincipalCredential creates an Azure SDK token credential from
// authentication parameters resolved by the Azure service principal provider.
func NewServicePrincipalCredential(authParams map[string]string) (azcore.TokenCredential, error) {
	tenantID := authParams[ServicePrincipalTenantIDKey]
	clientID := authParams[ServicePrincipalClientIDKey]
	clientSecret := authParams[ServicePrincipalClientSecretKey]
	clientCertificate := authParams[ServicePrincipalClientCertificateKey]
	clientCertificatePassword := authParams[ServicePrincipalClientCertificatePasswordKey]

	if tenantID == "" {
		return nil, fmt.Errorf("azure service principal tenantId is required")
	}
	if clientID == "" {
		return nil, fmt.Errorf("azure service principal clientId is required")
	}
	if clientSecret != "" && clientCertificate != "" {
		return nil, fmt.Errorf("azure service principal clientSecret and clientCertificate are mutually exclusive")
	}
	if clientSecret == "" && clientCertificate == "" {
		return nil, fmt.Errorf("azure service principal requires either clientSecret or clientCertificate")
	}
	if clientCertificatePassword != "" && clientCertificate == "" {
		return nil, fmt.Errorf("azure service principal clientCertificatePassword requires clientCertificate")
	}

	clientOptions, disableInstanceDiscovery, err := getServicePrincipalCredentialOptions(authParams)
	if err != nil {
		return nil, err
	}

	if clientSecret != "" {
		return azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, &azidentity.ClientSecretCredentialOptions{
			ClientOptions:            clientOptions,
			DisableInstanceDiscovery: disableInstanceDiscovery,
		})
	}

	certificates, privateKey, err := azidentity.ParseCertificates([]byte(clientCertificate), []byte(clientCertificatePassword))
	if err != nil {
		return nil, fmt.Errorf("failed to parse azure service principal clientCertificate: %w", err)
	}
	return azidentity.NewClientCertificateCredential(tenantID, clientID, certificates, privateKey, &azidentity.ClientCertificateCredentialOptions{
		ClientOptions:            clientOptions,
		DisableInstanceDiscovery: disableInstanceDiscovery,
	})
}

func getServicePrincipalCredentialOptions(authParams map[string]string) (azcore.ClientOptions, bool, error) {
	cloudName := authParams[ServicePrincipalCloudKey]
	activeDirectoryEndpoint := authParams[ServicePrincipalActiveDirectoryEndpointKey]
	if cloudName == "" {
		return azcore.ClientOptions{}, false, nil
	}

	resolvedEndpoint, err := ParseActiveDirectoryEndpoint(map[string]string{
		"cloud":                   cloudName,
		"activeDirectoryEndpoint": activeDirectoryEndpoint,
	})
	if err != nil {
		return azcore.ClientOptions{}, false, fmt.Errorf("failed to resolve azure service principal cloud: %w", err)
	}

	return azcore.ClientOptions{
		Cloud: cloud.Configuration{
			ActiveDirectoryAuthorityHost: resolvedEndpoint,
		},
	}, strings.EqualFold(cloudName, PrivateCloud), nil
}
