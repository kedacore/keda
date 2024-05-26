package azure

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func NewChainedCredential(logger logr.Logger, podIdentity v1alpha1.AuthPodIdentity) (*azidentity.ChainedTokenCredential, error) {
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

	switch podIdentity.Provider {
	case v1alpha1.PodIdentityProviderAzureWorkload:
		wiCred, err := NewADWorkloadIdentityCredential(podIdentity.GetIdentityID(), podIdentity.GetIdentityTenantID())
		if err != nil {
			logger.Error(err, "error starting azure workload-identity token provider")
		} else {
			logger.V(1).Info("azure workload-identity token provider registered")
			creds = append(creds, wiCred)
		}
	default:
		return nil, fmt.Errorf("pod identity %s not supported for azure credentials chain", podIdentity.Provider)
	}

	// Create the chained credential based on the previous 3
	return azidentity.NewChainedTokenCredential(creds, nil)
}
