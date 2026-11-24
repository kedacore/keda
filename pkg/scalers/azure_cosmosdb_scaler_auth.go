package scalers

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azcloud "github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/validation"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
)

type cosmosDBEnvironment struct {
	azure.AzEnvironment
	private bool
}

func resolveCosmosDBEnvironment(metadata map[string]string) (cosmosDBEnvironment, error) {
	for _, parameter := range []string{"cosmosDBResourceURL", "activeDirectoryEndpoint"} {
		if _, ok := metadata[parameter]; ok {
			return cosmosDBEnvironment{}, fmt.Errorf("%s metadata is not supported: configure private clouds with the operator's AZURE_ENVIRONMENT_FILEPATH instead", parameter)
		}
	}
	name := metadata["cloud"]
	if name == "" {
		name = "AzurePublicCloud"
	}
	private := strings.EqualFold(name, azure.PrivateCloud) || strings.EqualFold(name, "AzureStackCloud")
	if private {
		name = "AzureStackCloud"
	}
	env, err := azure.EnvironmentFromName(name)
	if err != nil {
		return cosmosDBEnvironment{}, fmt.Errorf("error resolving cosmos db cloud: %w", err)
	}
	suffix := strings.ToLower(env.CosmosDBDNSSuffix)
	if len(validation.IsDNS1123Subdomain(suffix)) != 0 || !strings.Contains(suffix, ".") || net.ParseIP(suffix) != nil {
		return cosmosDBEnvironment{}, fmt.Errorf("cosmos db cloud must configure a valid cosmosDBDNSSuffix")
	}
	for _, endpoint := range []string{env.ActiveDirectoryEndpoint, env.ResourceIdentifiers.CosmosDB} {
		if _, err := parseCosmosDBHTTPSRoot(endpoint); err != nil {
			return cosmosDBEnvironment{}, fmt.Errorf("cosmos db cloud must configure HTTPS activeDirectoryEndpoint and resourceIdentifiers.cosmosDB origins: %w", err)
		}
	}
	env.CosmosDBDNSSuffix = suffix
	return cosmosDBEnvironment{AzEnvironment: env, private: private}, nil
}

func parseCosmosDBHTTPSRoot(endpoint string) (*url.URL, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid cosmos db HTTPS origin")
	}
	if _, err := cosmosDBHTTPSOrigin(u); err != nil || (u.Path != "" && u.Path != "/") ||
		u.RawPath != "" || u.ForceQuery || u.RawQuery != "" || strings.Contains(endpoint, "#") {
		return nil, fmt.Errorf("cosmos db endpoints must be HTTPS origins without user information, query, fragment or non-root path")
	}
	return u, nil
}

func cosmosDBHTTPSOrigin(u *url.URL) (string, error) {
	if u == nil || u.Scheme != "https" || u.Opaque != "" || u.User != nil || u.Fragment != "" {
		return "", fmt.Errorf("invalid cosmos db HTTPS origin")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || (net.ParseIP(host) == nil && len(validation.IsDNS1123Subdomain(host)) != 0) {
		return "", fmt.Errorf("invalid cosmos db HTTPS hostname")
	}
	port := u.Port()
	if port == "" {
		if strings.HasSuffix(u.Host, ":") {
			return "", fmt.Errorf("invalid cosmos db HTTPS port")
		}
		port = "443"
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return "", fmt.Errorf("invalid cosmos db HTTPS port")
	}
	return net.JoinHostPort(host, strconv.Itoa(number)), nil
}

func validateCosmosDBEndpoint(endpoint, suffix string) error {
	u, err := parseCosmosDBHTTPSRoot(endpoint)
	if err != nil {
		return err
	}
	if net.ParseIP(u.Hostname()) != nil || !strings.HasSuffix(strings.ToLower(u.Hostname()), "."+suffix) {
		return fmt.Errorf("cosmos db account endpoint must be a subdomain of the configured cloud's cosmosDBDNSSuffix")
	}
	return nil
}

func cosmosDBRequestMatchesEndpoint(req *http.Request, endpoint string) bool {
	u, err := parseCosmosDBHTTPSRoot(endpoint)
	if err != nil {
		return false
	}
	expected, err := cosmosDBHTTPSOrigin(u)
	if err != nil {
		return false
	}
	actual, err := cosmosDBHTTPSOrigin(req.URL)
	if err != nil || actual != expected {
		return false
	}
	if req.Host != "" {
		hostOrigin, err := cosmosDBHTTPSOrigin(&url.URL{Scheme: "https", Host: req.Host})
		return err == nil && hostOrigin == expected
	}
	return true
}

func rejectCosmosDBRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

// Discovery may use Microsoft's discovery host, but credentials must only reach the configured authority.
type cosmosDBCredentialTransport struct {
	client    *http.Client
	authority string
	region    string
}

func (t cosmosDBCredentialTransport) Do(req *http.Request) (*http.Response, error) {
	if _, err := cosmosDBHTTPSOrigin(req.URL); err != nil {
		return nil, err
	}
	if !cosmosDBRequestMatchesEndpoint(req, t.authority) && !isCosmosDBInstanceDiscoveryRequest(req) &&
		!isCosmosDBRegionalAuthorityRequest(req, t.authority, t.region) {
		return nil, fmt.Errorf("refusing cosmos db credential request outside the configured authority origin")
	}
	return t.client.Do(req)
}

func isCosmosDBRegionalAuthorityRequest(req *http.Request, authority, region string) bool {
	if region == "" {
		return false
	}
	u, err := parseCosmosDBHTTPSRoot(authority)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	discoveryHost := host
	switch host {
	case "login.microsoftonline.com", "login.windows.net", "login.microsoft.com", "sts.windows.net":
		discoveryHost = "login.microsoft.com"
	case "login.chinacloudapi.cn", "login.partner.microsoftonline.cn":
		discoveryHost = "login.partner.microsoftonline.cn"
	case "login.microsoftonline.us", "login.microsoftonline.de":
	default:
		return false
	}
	for _, base := range []string{host, discoveryHost} {
		candidate := region
		if region == "TryAutoDetect" {
			// MSAL detects the region via administrator environment or IMDS; trust only
			// one regional DNS label beneath the selected Microsoft authority host.
			candidate = strings.TrimSuffix(strings.ToLower(req.URL.Hostname()), "."+base)
		}
		if len(candidate) == 0 || candidate[0] < 'a' || candidate[0] > 'z' ||
			len(validation.IsDNS1123Label(candidate)) != 0 {
			continue
		}
		if cosmosDBRequestMatchesEndpoint(req, "https://"+candidate+"."+base) {
			return true
		}
	}
	return false
}

func isCosmosDBInstanceDiscoveryRequest(req *http.Request) bool {
	if req.Method != http.MethodGet || req.URL.Path != "/common/discovery/instance" ||
		req.Body != nil || req.Header.Get("Authorization") != "" || req.Header.Get("Cookie") != "" {
		return false
	}
	return cosmosDBRequestMatchesEndpoint(req, "https://login.microsoft.com") ||
		cosmosDBRequestMatchesEndpoint(req, "https://login.microsoftonline.com")
}

func newCosmosDBTokenCredential(meta *azureCosmosDBMetadata, env cosmosDBEnvironment, podIdentity kedav1alpha1.AuthPodIdentity, logger logr.Logger, client *http.Client) (azcore.TokenCredential, string, error) {
	authority := env.ActiveDirectoryEndpoint
	if podIdentity.Provider == kedav1alpha1.PodIdentityProviderAzureWorkload {
		if host := os.Getenv("AZURE_AUTHORITY_HOST"); host != "" {
			authority = host
		}
	}
	if _, err := parseCosmosDBHTTPSRoot(authority); err != nil {
		return nil, "", fmt.Errorf("invalid cosmos db credential authority: %w", err)
	}
	credentialClient := *client
	credentialClient.CheckRedirect = rejectCosmosDBRedirect
	options := azcore.ClientOptions{
		Cloud:     azcloud.Configuration{ActiveDirectoryAuthorityHost: authority},
		Transport: cosmosDBCredentialTransport{client: &credentialClient, authority: authority, region: os.Getenv("AZURE_REGIONAL_AUTHORITY_NAME")},
	}
	var credential azcore.TokenCredential
	var err error
	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		credential, err = azidentity.NewClientSecretCredential(meta.TenantID, meta.ClientID, meta.ClientSecret, &azidentity.ClientSecretCredentialOptions{
			ClientOptions: options, DisableInstanceDiscovery: env.private,
		})
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		credential, err = newCosmosDBWorkloadCredential(options, env.private, podIdentity, logger)
	default:
		return nil, "", fmt.Errorf("pod identity %s not supported for azure cosmos db", podIdentity.Provider)
	}
	if err != nil {
		return nil, "", fmt.Errorf("error creating cosmos db credential: %w", err)
	}
	return credential, env.ResourceIdentifiers.CosmosDB, nil
}

func newCosmosDBWorkloadCredential(options azcore.ClientOptions, private bool, podIdentity kedav1alpha1.AuthPodIdentity, logger logr.Logger) (azcore.TokenCredential, error) {
	var credentials []azcore.TokenCredential
	// Preserve the Azure chain's CLI fallback for local development; production images have no shell.
	if _, err := os.Stat("/bin/sh"); err == nil {
		credential, err := azidentity.NewAzureCLICredential(nil)
		if err != nil {
			logger.Error(err, "error starting az-cli token provider")
		} else {
			credentials = append(credentials, credential)
		}
	}
	credential, err := azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
		ClientOptions: options, DisableInstanceDiscovery: private,
		ClientID: podIdentity.GetIdentityID(), TenantID: podIdentity.GetIdentityTenantID(),
	})
	if err != nil {
		logger.Error(err, "error starting azure workload-identity token provider")
	} else {
		credentials = append(credentials, credential)
	}
	return azidentity.NewChainedTokenCredential(credentials, nil)
}
