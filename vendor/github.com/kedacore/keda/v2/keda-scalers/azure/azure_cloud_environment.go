package azure

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	az "github.com/Azure/go-autorest/autorest/azure"
)

var AzureClouds = map[string]cloud.Configuration{
	"AZUREPUBLICCLOUD":       cloud.AzurePublic,
	"AZUREUSGOVERNMENTCLOUD": cloud.AzureGovernment,
	"AZURECHINACLOUD":        cloud.AzureChina,
}

const (
	DefaultCloud = "azurePublicCloud"

	// PrivateCloud cloud type
	PrivateCloud string = "Private"

	// DefaultEndpointSuffixKey is the default endpoint key in trigger metadata
	DefaultEndpointSuffixKey string = "endpointSuffix"

	// DefaultStorageSuffixKey is the default storage endpoint key in trigger metadata
	DefaultStorageSuffixKey string = "storageEndpointSuffix"

	// DefaultActiveDirectoryEndpointKey is the default active directory endpoint key in trigger metadata
	DefaultActiveDirectoryEndpointKey string = "activeDirectoryEndpoint"
)

// EnvironmentPropertyProvider for different types of Azure scalers
type EnvironmentPropertyProvider func(env az.Environment) (string, error)

var activeDirectoryEndpointProvider = func(env az.Environment) (string, error) {
	return env.ActiveDirectoryEndpoint, nil
}

// ParseEnvironmentProperty parses cloud metadata and returns the resolved property
func ParseEnvironmentProperty(metadata map[string]string, propertyKey string, envPropertyProvider EnvironmentPropertyProvider) (string, error) {
	if val, ok := metadata["cloud"]; ok && val != "" {
		if strings.EqualFold(val, PrivateCloud) {
			if val, ok := metadata[propertyKey]; ok && val != "" {
				return val, nil
			}
			return "", fmt.Errorf("%s must be provided for %s cloud type", propertyKey, PrivateCloud)
		}

		env, err := az.EnvironmentFromName(val)
		if err != nil {
			return "", fmt.Errorf("invalid cloud environment %s", val)
		}

		return envPropertyProvider(env)
	}

	// Use public cloud suffix if `cloud` isn't specified
	return envPropertyProvider(az.PublicCloud)
}

func ParseActiveDirectoryEndpoint(metadata map[string]string) (string, error) {
	return ParseEnvironmentProperty(metadata, DefaultActiveDirectoryEndpointKey, activeDirectoryEndpointProvider)
}
