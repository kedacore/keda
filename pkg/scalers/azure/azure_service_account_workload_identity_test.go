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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type serviceAccountRoundTripper func(*http.Request) (*http.Response, error)

func (f serviceAccountRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type countingTokenCredential struct {
	calls int
}

func (c *countingTokenCredential) GetToken(_ context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.calls++
	return azcore.AccessToken{Token: strings.Join(options.Scopes, ","), ExpiresOn: time.Now().Add(time.Hour)}, nil
}

func azureServiceAccountIdentity() kedav1alpha1.AuthPodIdentity {
	return kedav1alpha1.AuthPodIdentity{
		Provider:           kedav1alpha1.PodIdentityProviderAzureWorkload,
		ServiceAccountName: ptr.To("reader"),
		IdentityID:         ptr.To("11111111-1111-1111-1111-111111111111"),
		IdentityTenantID:   ptr.To("22222222-2222-2222-2222-222222222222"),
	}
}

func azureIdentityResponse(request *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}
}

func azureIdentityDiscovery(request *http.Request, tokenEndpoint string) *http.Response {
	return azureIdentityResponse(request, http.StatusOK, fmt.Sprintf(`{
		"token_endpoint": %q,
		"issuer": "https://%s/22222222-2222-2222-2222-222222222222/v2.0",
		"authorization_endpoint": "https://%s/22222222-2222-2222-2222-222222222222/oauth2/v2.0/authorize"
	}`, tokenEndpoint, request.URL.Host, request.URL.Host))
}

func TestAzureServiceAccountCredentialExchangeAndRefresh(t *testing.T) {
	identity := azureServiceAccountIdentity()
	mints, exchanges := 0, 0
	provider := &scalersconfig.ServiceAccountTokenProvider{
		Audience: serviceAccountWorkloadIdentityAudience,
		GetToken: func(ctx context.Context) (string, error) {
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			assert.LessOrEqual(t, time.Until(deadline), serviceAccountWorkloadIdentityTimeout)
			mints++
			return fmt.Sprintf("assertion-%d", mints), nil
		},
	}
	transport := serviceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
		assert.Equal(t, "login.microsoftonline.com", request.URL.Host)
		if strings.HasSuffix(request.URL.Path, "openid-configuration") {
			return azureIdentityDiscovery(request, "https://login.microsoftonline.com/"+identity.GetIdentityTenantID()+"/oauth2/v2.0/token"), nil
		}
		require.Equal(t, "/"+identity.GetIdentityTenantID()+"/oauth2/v2.0/token", request.URL.Path)
		require.NoError(t, request.ParseForm())
		exchanges++
		assert.Equal(t, identity.GetIdentityID(), request.Form.Get("client_id"))
		assert.Equal(t, "client_credentials", request.Form.Get("grant_type"))
		assert.Equal(t, "urn:ietf:params:oauth:client-assertion-type:jwt-bearer", request.Form.Get("client_assertion_type"))
		assert.Equal(t, "https://management.azure.com/.default openid offline_access profile", request.Form.Get("scope"))
		assert.Equal(t, fmt.Sprintf("assertion-%d", exchanges), request.Form.Get("client_assertion"))
		expires := 3600
		if exchanges == 1 {
			expires = 1
		}
		return azureIdentityResponse(request, http.StatusOK, fmt.Sprintf(`{"access_token":"access-%d","expires_in":%d,"token_type":"Bearer"}`, exchanges, expires)), nil
	})
	credential, err := newServiceAccountWorkloadIdentityCredential(identity, provider, transport)
	require.NoError(t, err)
	options := policy.TokenRequestOptions{Scopes: []string{"https://management.azure.com/.default"}}
	for _, expected := range []string{"access-1", "access-2", "access-2"} {
		token, tokenErr := credential.GetToken(context.Background(), options)
		require.NoError(t, tokenErr)
		assert.Equal(t, expected, token.Token)
	}
	assert.Equal(t, 2, mints)
	assert.Equal(t, 2, exchanges)
}

func TestAzureServiceAccountCredentialRejectsInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*kedav1alpha1.AuthPodIdentity, **scalersconfig.ServiceAccountTokenProvider)
	}{
		{"provider", func(identity *kedav1alpha1.AuthPodIdentity, _ **scalersconfig.ServiceAccountTokenProvider) {
			identity.Provider = kedav1alpha1.PodIdentityProviderGCP
		}},
		{"service account", func(identity *kedav1alpha1.AuthPodIdentity, _ **scalersconfig.ServiceAccountTokenProvider) {
			identity.ServiceAccountName = nil
		}},
		{"client", func(identity *kedav1alpha1.AuthPodIdentity, _ **scalersconfig.ServiceAccountTokenProvider) {
			identity.IdentityID = nil
		}},
		{"tenant", func(identity *kedav1alpha1.AuthPodIdentity, _ **scalersconfig.ServiceAccountTokenProvider) {
			identity.IdentityTenantID = nil
		}},
		{"missing token provider", func(_ *kedav1alpha1.AuthPodIdentity, provider **scalersconfig.ServiceAccountTokenProvider) {
			*provider = nil
		}},
		{"missing callback", func(_ *kedav1alpha1.AuthPodIdentity, provider **scalersconfig.ServiceAccountTokenProvider) {
			(*provider).GetToken = nil
		}},
		{"Kubernetes audience", func(_ *kedav1alpha1.AuthPodIdentity, provider **scalersconfig.ServiceAccountTokenProvider) {
			(*provider).Audience = "https://kubernetes.default.svc"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			identity := azureServiceAccountIdentity()
			provider := &scalersconfig.ServiceAccountTokenProvider{Audience: serviceAccountWorkloadIdentityAudience, GetToken: func(context.Context) (string, error) { return "token", nil }}
			test.change(&identity, &provider)
			_, err := NewServiceAccountWorkloadIdentityCredential(identity, provider)
			require.Error(t, err)
		})
	}
	_, err := NewChainedCredential(logr.Discard(), azureServiceAccountIdentity())
	require.ErrorContains(t, err, "requires a service account token provider")
}

func TestAzureServiceAccountCredentialAuthority(t *testing.T) {
	t.Setenv(azureAuthorityHostEnv, "https://operator-authority.example")
	for _, authority := range []string{"", "https://login.microsoftonline.com", "https://login.microsoftonline.us/", "https://login.chinacloudapi.cn/"} {
		t.Run(authority, func(t *testing.T) {
			identity := azureServiceAccountIdentity()
			identity.IdentityAuthorityHost = &authority
			provider := &scalersconfig.ServiceAccountTokenProvider{Audience: serviceAccountWorkloadIdentityAudience, GetToken: func(context.Context) (string, error) { return "assertion", nil }}
			transport := serviceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
				expectedAuthority := strings.TrimSuffix(authority, "/")
				if expectedAuthority == "" {
					expectedAuthority = "https://login.microsoftonline.com"
				}
				assert.Equal(t, expectedAuthority, "https://"+request.URL.Host)
				if strings.HasSuffix(request.URL.Path, "openid-configuration") {
					return azureIdentityDiscovery(request, expectedAuthority+"/"+identity.GetIdentityTenantID()+"/oauth2/v2.0/token"), nil
				}
				return azureIdentityResponse(request, http.StatusOK, `{"access_token":"access","expires_in":3600,"token_type":"Bearer"}`), nil
			})
			credential, err := newServiceAccountWorkloadIdentityCredential(identity, provider, transport)
			require.NoError(t, err)
			_, err = credential.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{"scope/.default"}})
			require.NoError(t, err)
		})
	}
	for _, authority := range []string{"http://login.microsoftonline.com", "https://login.microsoftonline.com/tenant", "https://login.microsoftonline.com:443", "https://login.microsoftonline.com.evil.example", "https://evil.example", "https://user@login.microsoftonline.com", "https://login.microsoftonline.com?query=1", "https://login.microsoftonline.com#fragment"} {
		t.Run(authority, func(t *testing.T) {
			_, err := serviceAccountWorkloadIdentityCloud(authority)
			require.Error(t, err)
		})
	}
}

func TestAzureServiceAccountCredentialFailsClosed(t *testing.T) {
	for _, failure := range []string{"callback", "empty assertion", "rejected assertion", "redirect", "discovered endpoint"} {
		t.Run(failure, func(t *testing.T) {
			identity := azureServiceAccountIdentity()
			mints, exchanges := 0, 0
			provider := &scalersconfig.ServiceAccountTokenProvider{
				Audience: serviceAccountWorkloadIdentityAudience,
				GetToken: func(context.Context) (string, error) {
					mints++
					if failure == "callback" {
						return "", errors.New("minting denied")
					}
					if failure == "empty assertion" {
						return "", nil
					}
					return "assertion", nil
				},
			}
			transport := serviceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
				require.Equal(t, "login.microsoftonline.com", request.URL.Host, "assertions must never reach another authority")
				if strings.HasSuffix(request.URL.Path, "openid-configuration") {
					endpoint := "https://login.microsoftonline.com/" + identity.GetIdentityTenantID() + "/oauth2/v2.0/token"
					if failure == "discovered endpoint" {
						endpoint = "https://evil.example/token"
					}
					return azureIdentityDiscovery(request, endpoint), nil
				}
				exchanges++
				if failure == "redirect" {
					response := azureIdentityResponse(request, http.StatusTemporaryRedirect, `{}`)
					response.Header.Set("Location", "https://evil.example/token")
					return response, nil
				}
				return azureIdentityResponse(request, http.StatusForbidden, `{"error":"invalid_client","error_description":"denied"}`), nil
			})
			credential, err := newServiceAccountWorkloadIdentityCredential(identity, provider, transport)
			require.NoError(t, err)
			_, err = credential.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{"scope/.default"}})
			require.Error(t, err)
			assert.Equal(t, 1, mints)
			if failure == "callback" || failure == "empty assertion" || failure == "discovered endpoint" {
				assert.Zero(t, exchanges)
			} else {
				assert.Equal(t, 1, exchanges)
			}
		})
	}
}

func TestAzureServiceAccountLegacyAdapterOutlivesCreationContext(t *testing.T) {
	identity := azureServiceAccountIdentity()
	mints := 0
	provider := &scalersconfig.ServiceAccountTokenProvider{
		Audience: serviceAccountWorkloadIdentityAudience,
		GetToken: func(ctx context.Context) (string, error) {
			require.NoError(t, ctx.Err())
			mints++
			return "assertion", nil
		},
	}
	transport := serviceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
		if strings.HasSuffix(request.URL.Path, "openid-configuration") {
			return azureIdentityDiscovery(request, "https://login.microsoftonline.com/"+identity.GetIdentityTenantID()+"/oauth2/v2.0/token"), nil
		}
		require.NoError(t, request.ParseForm())
		assert.Equal(t, "https://resource.example/.default openid offline_access profile", request.Form.Get("scope"))
		return azureIdentityResponse(request, http.StatusOK, `{"access_token":"access","expires_in":3600,"token_type":"Bearer"}`), nil
	})
	credential, err := newServiceAccountWorkloadIdentityCredential(identity, provider, transport)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	adapter := NewAzureADWorkloadIdentityTokenProviderWithCredential(ctx, credential, "https://resource.example/")
	cancel()
	require.NoError(t, adapter.Refresh())
	assert.Equal(t, "access", adapter.OAuthToken())
	require.NoError(t, adapter.Refresh())
	assert.Equal(t, 1, mints)
	emptyAdapter := NewAzureADWorkloadIdentityTokenProviderWithCredential(context.Background(), nil, "https://resource.example/")
	require.ErrorContains(t, emptyAdapter.Refresh(), "explicit Azure workload identity credential is required")
}

func TestAzureServiceAccountLegacyAdapterCachesCredentialByResource(t *testing.T) {
	credential := &countingTokenCredential{}
	adapter := NewAzureADWorkloadIdentityTokenProviderWithCredential(t.Context(), credential, "https://first.example/")

	require.NoError(t, adapter.Refresh())
	require.NoError(t, adapter.Refresh())
	assert.Equal(t, 1, credential.calls)
	assert.Equal(t, "https://first.example/.default", adapter.OAuthToken())

	require.NoError(t, adapter.RefreshExchange("https://second.example/"))
	assert.Equal(t, 2, credential.calls)
	assert.Equal(t, "https://second.example/.default", adapter.OAuthToken())
}

func TestAzureServiceAccountCredentialsDoNotShareTokens(t *testing.T) {
	identity := azureServiceAccountIdentity()
	for _, serviceAccount := range []string{"first", "second"} {
		identity.ServiceAccountName = &serviceAccount
		mints := 0
		provider := &scalersconfig.ServiceAccountTokenProvider{
			Audience: serviceAccountWorkloadIdentityAudience,
			GetToken: func(context.Context) (string, error) {
				mints++
				return serviceAccount, nil
			},
		}
		transport := serviceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
			if strings.HasSuffix(request.URL.Path, "openid-configuration") {
				return azureIdentityDiscovery(request, "https://login.microsoftonline.com/"+identity.GetIdentityTenantID()+"/oauth2/v2.0/token"), nil
			}
			require.NoError(t, request.ParseForm())
			assert.Equal(t, serviceAccount, request.Form.Get("client_assertion"))
			return azureIdentityResponse(request, http.StatusOK, fmt.Sprintf(`{"access_token":%q,"expires_in":3600,"token_type":"Bearer"}`, serviceAccount)), nil
		})
		credential, err := newServiceAccountWorkloadIdentityCredential(identity, provider, transport)
		require.NoError(t, err)
		token, err := credential.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{"scope/.default"}})
		require.NoError(t, err)
		assert.Equal(t, serviceAccount, token.Token)
		assert.Equal(t, 1, mints)
	}
}

func TestAzureServiceAccountCredentialHonorsCancellation(t *testing.T) {
	provider := &scalersconfig.ServiceAccountTokenProvider{
		Audience: serviceAccountWorkloadIdentityAudience,
		GetToken: func(context.Context) (string, error) {
			t.Fatal("a canceled request must not mint an assertion")
			return "", nil
		},
	}
	transport := serviceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
		return nil, request.Context().Err()
	})
	credential, err := newServiceAccountWorkloadIdentityCredential(azureServiceAccountIdentity(), provider, transport)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{"scope/.default"}})
	require.ErrorIs(t, err, context.Canceled)
}
