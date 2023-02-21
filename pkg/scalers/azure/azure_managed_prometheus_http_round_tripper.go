package azure

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultAzureManagedPrometheusResourceURL = "https://prometheus.monitor.azure.com/.default"
)

var azureManagedPrometheusResourceURLInCloud = map[string]string{
	"AZUREPUBLICCLOUD":       "https://prometheus.monitor.azure.com/.default",
	"AZUREUSGOVERNMENTCLOUD": "https://prometheus.monitor.usgovcloudapi.net/.default",
	"AZURECHINACLOUD":        "https://prometheus.monitor.chinacloudapp.cn/.default",
}

type azureManagedPrometheusHttpRoundTripper struct {
	chainedCredential *azidentity.ChainedTokenCredential
	next              http.RoundTripper
	resourceURL       string
}

func TryAndGetAzureManagedPrometheusHttpRoundTripper(podIdentity kedav1alpha1.AuthPodIdentity, triggerMetadata map[string]string) (http.RoundTripper, error) {

	switch podIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzureWorkload, kedav1alpha1.PodIdentityProviderAzure:

		if triggerMetadata == nil {
			return nil, fmt.Errorf("trigger metadata cannot be nil")
		}

		tlsConfig := &tls.Config{
			MinVersion:         kedautil.GetMinTLSVersion(),
			InsecureSkipVerify: false,
		}

		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     tlsConfig,
		}

		chainedCred, err := NewChainedCredential(podIdentity.IdentityID, podIdentity.Provider)
		if err != nil {
			return nil, err
		}

		resourceUrlBasedOnCloud, err := getResourceUrlBasedOnCloud(triggerMetadata)
		if err != nil {
			return nil, err
		}

		rt := &azureManagedPrometheusHttpRoundTripper{
			next:              transport,
			chainedCredential: chainedCred,
			resourceURL:       resourceUrlBasedOnCloud,
		}
		return rt, nil
	}

	// Not azure managed prometheus. Don't do anything.
	return nil, nil
}

func getResourceUrlBasedOnCloud(triggerMetadata map[string]string) (string, error) {
	if cloud, ok := triggerMetadata["cloud"]; ok {
		if strings.EqualFold(cloud, PrivateCloud) {
			if resource, ok := triggerMetadata["azureManagedPrometheusResourceURL"]; ok && resource != "" {
				return resource, nil
			}

			return "", fmt.Errorf("azureManagedPrometheusResourceURL must be provided for %s cloud type", PrivateCloud)
		}

		if resource, ok := azureManagedPrometheusResourceURLInCloud[strings.ToUpper(cloud)]; ok {
			return resource, nil
		}
		return "", fmt.Errorf("there is no cloud environment matching the name %s", cloud)
	}

	// return default resourceURL for public cloud
	return defaultAzureManagedPrometheusResourceURL, nil
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
