package azure

import (
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	az "github.com/Azure/go-autorest/autorest/azure"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/util"
)

type azureManagedPrometheusHttpRoundTripper struct {
	chainedCredential *azidentity.ChainedTokenCredential
	next              http.RoundTripper
	resourceURL       string
}

// Tries to get a round tripper.
// If the pod identity represents azure auth, it creates a round tripper and returns that. Returns error if fails to create one.
// If its not azure auth, then this becomes a no-op. Neither returns round tripper nor error.
func TryAndGetAzureManagedPrometheusHTTPRoundTripper(podIdentity kedav1alpha1.AuthPodIdentity, triggerMetadata map[string]string) (http.RoundTripper, error) {

	switch podIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzureWorkload, kedav1alpha1.PodIdentityProviderAzure:

		if triggerMetadata == nil {
			return nil, fmt.Errorf("trigger metadata cannot be nil")
		}

		chainedCred, err := NewChainedCredential(podIdentity.IdentityID, podIdentity.Provider)
		if err != nil {
			return nil, err
		}

		azureManagedPrometheusResourceURLProvider := func(env az.Environment) (string, error) {
			if env.ResourceIdentifiers.AzureManagedPrometheus == az.NotAvailable {
				return "", fmt.Errorf("Azure Managed Prometheus is not avaiable in cloud %s", env.Name)
			}
			return env.ResourceIdentifiers.AzureManagedPrometheus, nil
		}

		resourceURLBasedOnCloud, err := ParseEnvironmentProperty(triggerMetadata, "azureManagedPrometheusResourceURL", azureManagedPrometheusResourceURLProvider)
		if err != nil {
			return nil, err
		}

		transport := util.CreateHTTPTransport(false)
		rt := &azureManagedPrometheusHttpRoundTripper{
			next:              transport,
			chainedCredential: chainedCred,
			resourceURL:       resourceURLBasedOnCloud,
		}
		return rt, nil
	}

	// Not azure managed prometheus. Don't create a round tripper and don't return error.
	return nil, nil
}

// Sets Auhtorization header for requests
func (rt *azureManagedPrometheusHttpRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := rt.chainedCredential.GetToken(req.Context(), policy.TokenRequestOptions{Scopes: []string{rt.resourceURL}})
	if err != nil {
		return nil, err
	}

	bearerAccessToken := "Bearer " + token.Token
	req.Header.Set("Authorization", bearerAccessToken)

	return rt.next.RoundTrip(req)
}
