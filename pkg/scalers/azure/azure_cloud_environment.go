package azure

import (
	"fmt"
	"strings"

	az "github.com/Azure/go-autorest/autorest/azure"
)

// EnvironmentSuffixProvider for different types of Azure scalers
type EnvironmentSuffixProvider func(env az.Environment) (string, error)

// ParseEndpointSuffix parses cloud and endpointSuffix metadata and returns the resolved endpoint suffix
func ParseEndpointSuffix(metadata map[string]string, suffixProvider EnvironmentSuffixProvider) (string, error) {
	if val, ok := metadata["cloud"]; ok && val != "" {
		if strings.EqualFold(val, PrivateCloud) {
			if val, ok := metadata["endpointSuffix"]; ok && val != "" {
				return val, nil
			}
			return "", fmt.Errorf("endpointSuffix must be provided for %s cloud type", PrivateCloud)
		}

		env, err := az.EnvironmentFromName(val)
		if err != nil {
			return "", fmt.Errorf("invalid cloud environment %s", val)
		}

		return suffixProvider(env)
	}

	// Use public cloud suffix if `cloud` isn't specified
	return suffixProvider(az.PublicCloud)
}
