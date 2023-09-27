package azure

import (
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func NewChainedCredential(logger logr.Logger, identityID string, podIdentity v1alpha1.PodIdentityProvider) (*azidentity.ChainedTokenCredential, error) {
	var creds []azcore.TokenCredential

	// Used for local debug based on az-cli user
	// As production images don't have shell, we can't register this provider always
	if _, err := os.Stat("/bin/sh"); err == nil {
		cliCred, err := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{})
		if err != nil {
			logger.Error(err, "error starting az-cli token provider")
		} else {
			logger.V(1).Info("az-cli token provider registered")
			creds = append(creds, cliCred)
		}
	}

	// https://github.com/kedacore/keda/issues/4123
	// We shouldn't register both in the same chain because if both are registered, KEDA will use the first one
	// which returns a valid token. This could produce an unintended behaviour if end-users use 2 different identities
	// with 2 different permissions. They could set workload-identity with the identity A, but KEDA would use
	// aad-pod-identity with the identity B. If both identities are differents or have different permissions, this blocks
	// workload identity
	switch podIdentity {
	case v1alpha1.PodIdentityProviderAzure:
		// Used for aad-pod-identity
		msiCred, err := ManagedIdentityWrapperCredential(identityID)
		if err != nil {
			logger.Error(err, "error starting aad-pod-identity token provider")
		} else {
			logger.V(1).Info("aad-pod-identity token provider registered")
			creds = append(creds, msiCred)
		}
	case v1alpha1.PodIdentityProviderAzureWorkload:
		wiCred, err := NewADWorkloadIdentityCredential(identityID)
		if err != nil {
			logger.Error(err, "error starting azure workload-identity token provider")
		} else {
			logger.V(1).Info("azure workload-identity token provider registered")
			creds = append(creds, wiCred)
		}
	}

	// Create the chained credential based on the previous 3
	return azidentity.NewChainedTokenCredential(creds, nil)
}
