package azure

import (
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

func NewChainedCredential(identityID string) (*azidentity.ChainedTokenCredential, error) {
	var creds []azcore.TokenCredential

	// Used for local debug based on az-cli user
	// As production images don't have shell, we can't register this provider always
	if _, err := os.Stat("/bin/sh"); err == nil {
		cliCred, err := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{})
		if err == nil {
			creds = append(creds, cliCred)
		}
	}

	// Used for aad-pod-identity
	msiCred, err := ManagedIdentityWrapperCredential(identityID)
	if err == nil {
		creds = append(creds, msiCred)
	}

	wiCred, err := NewADWorkloadIdentityCredential(identityID)
	if err == nil {
		creds = append(creds, wiCred)
	}

	// Create the chained credential based on the previous 3
	return azidentity.NewChainedTokenCredential(creds, nil)
}
