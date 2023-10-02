package azure

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	az "github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/util"
)

var azureManagedPrometheusResourceURLInCloud = map[string]string{
	"AZUREPUBLICCLOUD":       "https://prometheus.monitor.azure.com/.default",
	"AZUREUSGOVERNMENTCLOUD": "https://prometheus.monitor.azure.us/.default",
	"AZURECHINACLOUD":        "https://prometheus.monitor.azure.cn/.default",
}

type azureManagedPrometheusHTTPRoundTripper struct {
	chainedCredential *azidentity.ChainedTokenCredential
	next              http.RoundTripper
	resourceURL       string
}

// TryAndGetAzureManagedPrometheusHTTPRoundTripper tries to get a round tripper.
// If the pod identity represents azure auth, it creates a round tripper and returns that. Returns error if fails to create one.
// If its not azure auth, then this becomes a no-op. Neither returns round tripper nor error.
func TryAndGetAzureManagedPrometheusHTTPRoundTripper(logger logr.Logger, podIdentity kedav1alpha1.AuthPodIdentity, triggerMetadata map[string]string) (http.RoundTripper, error) {
	switch podIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzureWorkload, kedav1alpha1.PodIdentityProviderAzure:

		if triggerMetadata == nil {
			return nil, fmt.Errorf("trigger metadata cannot be nil")
		}

		chainedCred, err := NewChainedCredential(logger, podIdentity.GetIdentityID(), podIdentity.Provider)
		if err != nil {
			return nil, err
		}

		azureManagedPrometheusResourceURLProvider := func(env az.Environment) (string, error) {
			if resource, ok := azureManagedPrometheusResourceURLInCloud[strings.ToUpper(env.Name)]; ok {
				return resource, nil
			}

			return "", fmt.Errorf("azure managed prometheus is not available in cloud %s", env.Name)
		}

		resourceURLBasedOnCloud, err := ParseEnvironmentProperty(triggerMetadata, "azureManagedPrometheusResourceURL", azureManagedPrometheusResourceURLProvider)
		if err != nil {
			return nil, err
		}

		transport := util.CreateHTTPTransport(false)
		rt := &azureManagedPrometheusHTTPRoundTripper{
			next:              transport,
			chainedCredential: chainedCred,
			resourceURL:       resourceURLBasedOnCloud,
		}
		return rt, nil
	}

	// Not azure managed prometheus. Don't create a round tripper and don't return error.
	return nil, nil
}

// RoundTrip sets authorization header for requests
func (rt *azureManagedPrometheusHTTPRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := rt.chainedCredential.GetToken(req.Context(), policy.TokenRequestOptions{Scopes: []string{rt.resourceURL}})

	if err != nil {
		return nil, err
	}

	bearerAccessToken := "Bearer " + token.Token
	req.Header.Set("Authorization", bearerAccessToken)

	return rt.next.RoundTrip(req)
}
