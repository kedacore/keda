package scalers

import (
	"fmt"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type gcpAuthorizationMetadata struct {
	GoogleApplicationCredentials string
	GoogleApplicationCredentialsFile string
	podIdentityOwner             bool
	podIdentityProviderEnabled   bool
}

func getGcpAuthorization(config *ScalerConfig, resolvedEnv map[string]string) (*gcpAuthorizationMetadata, error) {
	metadata := config.TriggerMetadata
	authParams := config.AuthParams
	meta := gcpAuthorizationMetadata{}
	if metadata["identityOwner"] == "operator" {
		meta.podIdentityOwner = false
	} else if metadata["identityOwner"] == "" || metadata["identityOwner"] == "pod" {
		meta.podIdentityOwner = true
		switch {
		case config.PodIdentity == kedav1alpha1.PodIdentityProviderGCP:
			// do nothing, rely on underneath metadata google
			meta.podIdentityProviderEnabled = true
		case authParams["GoogleApplicationCredentials"] != "":
			meta.GoogleApplicationCredentials = authParams["GoogleApplicationCredentials"]
		default:
			if metadata["credentialsFromEnv"] != "" {
				meta.GoogleApplicationCredentials = resolvedEnv[metadata["credentialsFromEnv"]]
			} else if metadata["credentialsFromEnvFile"] != "" {
				meta.GoogleApplicationCredentialsFile = resolvedEnv[metadata["credentialsFromEnvFile"]]
			} else {
				return nil, fmt.Errorf("GoogleApplicationCredentials not found")
			}
		}
	}
	return &meta, nil
}
