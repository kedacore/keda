/*
Copyright 2026 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azure

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	serviceAccountWorkloadIdentityAudience = "api://AzureADTokenExchange"
	serviceAccountWorkloadIdentityTimeout  = 30 * time.Second
)

// NewServiceAccountWorkloadIdentityCredential exchanges freshly minted Kubernetes
// assertions for Azure access tokens without consulting the operator's credentials.
func NewServiceAccountWorkloadIdentityCredential(podIdentity kedav1alpha1.AuthPodIdentity, provider *scalersconfig.ServiceAccountTokenProvider) (azcore.TokenCredential, error) {
	return newServiceAccountWorkloadIdentityCredential(podIdentity, provider, http.DefaultTransport)
}

func newServiceAccountWorkloadIdentityCredential(podIdentity kedav1alpha1.AuthPodIdentity, provider *scalersconfig.ServiceAccountTokenProvider, transport http.RoundTripper) (azcore.TokenCredential, error) {
	if podIdentity.Provider != kedav1alpha1.PodIdentityProviderAzureWorkload || podIdentity.ServiceAccountName == nil {
		return nil, fmt.Errorf("azure service account workload identity requires provider azure-workload and serviceAccountName")
	}
	if podIdentity.GetIdentityID() == "" || podIdentity.GetIdentityTenantID() == "" {
		return nil, fmt.Errorf("azure service account workload identity requires explicit identityId and identityTenantId")
	}
	if provider == nil || provider.GetToken == nil {
		return nil, fmt.Errorf("azure service account workload identity requires a service account token provider")
	}
	if provider.Audience != serviceAccountWorkloadIdentityAudience {
		return nil, fmt.Errorf("azure service account token audience must be %s", serviceAccountWorkloadIdentityAudience)
	}
	cloudConfig, err := serviceAccountWorkloadIdentityCloud(podIdentity.GetIdentityAuthorityHost())
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   serviceAccountWorkloadIdentityTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	credential, err := azidentity.NewClientAssertionCredential(podIdentity.GetIdentityTenantID(), podIdentity.GetIdentityID(), func(ctx context.Context) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, serviceAccountWorkloadIdentityTimeout)
		defer cancel()
		assertion, err := provider.GetToken(ctx)
		if err != nil {
			return "", err
		}
		if assertion == "" {
			return "", fmt.Errorf("azure service account token must not be empty")
		}
		return assertion, nil
	}, &azidentity.ClientAssertionCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: cloudConfig,
			Transport: serviceAccountWorkloadIdentityTransport{
				client: client,
				host:   strings.TrimSuffix(strings.TrimPrefix(cloudConfig.ActiveDirectoryAuthorityHost, "https://"), "/"),
			},
		},
		// The authority is restricted to the known Azure public and sovereign
		// clouds above. Do not discover an alternate authority for assertions.
		DisableInstanceDiscovery: true,
	})
	if err != nil {
		return nil, err
	}
	return &serviceAccountWorkloadIdentityCredential{credential: credential}, nil
}

func serviceAccountWorkloadIdentityCloud(authority string) (cloud.Configuration, error) {
	switch strings.TrimSuffix(authority, "/") {
	case "", "https://login.microsoftonline.com":
		return cloud.AzurePublic, nil
	case "https://login.microsoftonline.us":
		return cloud.AzureGovernment, nil
	case "https://login.chinacloudapi.cn":
		return cloud.AzureChina, nil
	default:
		return cloud.Configuration{}, fmt.Errorf("azure service account workload identity requires a public, government, or China Azure authority host")
	}
}

type serviceAccountWorkloadIdentityTransport struct {
	client *http.Client
	host   string
}

func (t serviceAccountWorkloadIdentityTransport) Do(request *http.Request) (*http.Response, error) {
	if request.URL.Scheme != "https" || request.URL.Host != t.host || request.URL.User != nil {
		return nil, fmt.Errorf("azure service account token exchange requested an untrusted authority")
	}
	return t.client.Do(request)
}

type serviceAccountWorkloadIdentityCredential struct {
	credential azcore.TokenCredential
}

func (c *serviceAccountWorkloadIdentityCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if err := ctx.Err(); err != nil {
		return azcore.AccessToken{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, serviceAccountWorkloadIdentityTimeout)
	defer cancel()
	return c.credential.GetToken(ctx, options)
}
