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

package gcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const testWorkloadIdentityAudience = "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/tenant-pool/providers/cluster"

type workloadIdentityRoundTripper func(*http.Request) (*http.Response, error)

func (r workloadIdentityRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return r(request)
}

func workloadIdentityResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestWorkloadIdentityProviderAudience(t *testing.T) {
	for _, audience := range []string{testWorkloadIdentityAudience, "https:" + testWorkloadIdentityAudience} {
		t.Run(audience, func(t *testing.T) {
			got, err := WorkloadIdentityProviderAudience(audience)
			require.NoError(t, err)
			assert.Equal(t, testWorkloadIdentityAudience, got)
		})
	}
	for _, audience := range []string{
		"", "https://kubernetes.default.svc", "http:" + testWorkloadIdentityAudience,
		" " + testWorkloadIdentityAudience, testWorkloadIdentityAudience + "/",
		testWorkloadIdentityAudience + "?redirect=elsewhere", testWorkloadIdentityAudience + "#fragment",
		strings.Replace(testWorkloadIdentityAudience, "iam.googleapis.com", "iam.googleapis.com.evil.example", 1),
		strings.Replace(testWorkloadIdentityAudience, "123456789", "project-name", 1),
		strings.Replace(testWorkloadIdentityAudience, "providers/cluster", "providers/../cluster", 1),
		strings.Replace(testWorkloadIdentityAudience, "providers/cluster", "providers/%63luster", 1),
	} {
		t.Run("invalid "+audience, func(t *testing.T) {
			_, err := WorkloadIdentityProviderAudience(audience)
			require.Error(t, err)
		})
	}
}

func TestGCPWorkloadIdentityRefresh(t *testing.T) {
	mints, exchanges := 0, 0
	provider := &scalersconfig.ServiceAccountTokenProvider{
		Audience: "https:" + testWorkloadIdentityAudience,
		GetToken: func(ctx context.Context) (string, error) {
			require.NoError(t, ctx.Err())
			deadline, ok := ctx.Deadline()
			require.True(t, ok, "minting must have a deadline")
			assert.WithinDuration(t, time.Now().Add(workloadIdentityRequestTimeout), deadline, time.Second)
			mints++
			return fmt.Sprintf("tenant-assertion-%d", mints), nil
		},
	}
	client := &http.Client{Transport: workloadIdentityRoundTripper(func(req *http.Request) (*http.Response, error) {
		exchanges++
		assert.Equal(t, "https://sts.googleapis.com/v1/token", req.URL.String())
		assert.Equal(t, http.MethodPost, req.Method)
		require.NoError(t, req.ParseForm())
		assert.Equal(t, testWorkloadIdentityAudience, req.Form.Get("audience"))
		assert.Equal(t, "urn:ietf:params:oauth:token-type:jwt", req.Form.Get("subject_token_type"))
		assert.Equal(t, "urn:ietf:params:oauth:grant-type:token-exchange", req.Form.Get("grant_type"))
		assert.Equal(t, GcpScopeMonitoringRead, req.Form.Get("scope"))
		assert.Equal(t, fmt.Sprintf("tenant-assertion-%d", exchanges), req.Form.Get("subject_token"))
		_, ok := req.Context().Deadline()
		assert.True(t, ok, "STS exchange must have a deadline")
		return workloadIdentityResponse(http.StatusOK, fmt.Sprintf(`{"access_token":"access-%d","token_type":"Bearer","expires_in":3600}`, exchanges)), nil
	})}
	ctx, cancel := context.WithCancel(context.WithValue(t.Context(), oauth2.HTTPClient, client))
	defer cancel()
	auth := &AuthorizationMetadata{ServiceAccountTokenProvider: provider}
	ts, err := auth.TokenSource(ctx, GcpScopeMonitoringRead)
	require.NoError(t, err)
	cancel() // Cached credentials must outlive the request that built the scaler.

	first, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "access-1", first.AccessToken)
	cached, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, first.AccessToken, cached.AccessToken)
	assert.Equal(t, 1, mints)
	assert.Equal(t, 1, exchanges)

	first.Expiry = time.Now().Add(-time.Second)
	second, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "access-2", second.AccessToken)
	assert.Equal(t, 2, mints)
	assert.Equal(t, 2, exchanges)
	assert.Equal(t, "https:"+testWorkloadIdentityAudience, provider.Audience, "STS normalization must not change the Kubernetes audience")
	assert.Zero(t, client.Timeout, "the caller's HTTP client must not be modified")
}

func TestGCPWorkloadIdentityImpersonation(t *testing.T) {
	const identity = "metrics-reader@tenant-project.iam.gserviceaccount.com"
	auth := &AuthorizationMetadata{
		IdentityID: identity,
		ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
			Audience: testWorkloadIdentityAudience,
			GetToken: func(context.Context) (string, error) { return "tenant-assertion", nil },
		},
	}
	requests := 0
	client := &http.Client{Transport: workloadIdentityRoundTripper(func(req *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			assert.Equal(t, "https://sts.googleapis.com/v1/token", req.URL.String())
			require.NoError(t, req.ParseForm())
			assert.Equal(t, "https://www.googleapis.com/auth/cloud-platform", req.Form.Get("scope"))
			return workloadIdentityResponse(http.StatusOK, `{"access_token":"federated","token_type":"Bearer","expires_in":3600}`), nil
		}
		assert.Equal(t, "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/"+identity+":generateAccessToken", req.URL.String())
		assert.Equal(t, "Bearer federated", req.Header.Get("Authorization"))
		var body struct {
			Scope []string `json:"scope"`
		}
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		assert.Equal(t, []string{GcpScopeMonitoringRead}, body.Scope)
		return workloadIdentityResponse(http.StatusOK, fmt.Sprintf(`{"accessToken":"impersonated","expireTime":%q}`, time.Now().Add(time.Hour).Format(time.RFC3339))), nil
	})}
	ctx := context.WithValue(t.Context(), oauth2.HTTPClient, client)
	ts, err := auth.TokenSource(ctx, GcpScopeMonitoringRead)
	require.NoError(t, err)
	token, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "impersonated", token.AccessToken)
	assert.Equal(t, 2, requests)
}

func TestGCPWorkloadIdentityFailsWithoutFallback(t *testing.T) {
	mintErr := errors.New("token request forbidden")
	for _, tt := range []struct {
		name  string
		token string
		err   error
	}{
		{name: "mint rejected", err: mintErr},
		{name: "empty token"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			client := &http.Client{Transport: workloadIdentityRoundTripper(func(*http.Request) (*http.Response, error) {
				requests++
				return nil, errors.New("unexpected request after failed token mint")
			})}
			auth := &AuthorizationMetadata{
				PodIdentityProviderEnabled: true,
				ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
					Audience: testWorkloadIdentityAudience,
					GetToken: func(context.Context) (string, error) { return tt.token, tt.err },
				},
			}
			ts, err := auth.TokenSource(context.WithValue(t.Context(), oauth2.HTTPClient, client), GcpScopeMonitoringRead)
			require.NoError(t, err)
			_, err = ts.Token()
			require.Error(t, err)
			if tt.err != nil {
				assert.ErrorIs(t, err, tt.err)
			} else {
				assert.ErrorContains(t, err, "empty token")
			}
			assert.Zero(t, requests)
		})
	}
	for _, status := range []int{http.StatusForbidden, http.StatusTemporaryRedirect} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			requests := 0
			client := &http.Client{Transport: workloadIdentityRoundTripper(func(req *http.Request) (*http.Response, error) {
				requests++
				assert.Equal(t, "sts.googleapis.com", req.URL.Host)
				response := workloadIdentityResponse(status, `{"error":"access_denied"}`)
				response.Header.Set("Location", "https://untrusted.example/token")
				return response, nil
			})}
			auth := &AuthorizationMetadata{
				PodIdentityProviderEnabled: true,
				ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
					Audience: testWorkloadIdentityAudience,
					GetToken: func(context.Context) (string, error) { return "assertion", nil },
				},
			}
			ts, err := auth.TokenSource(context.WithValue(t.Context(), oauth2.HTTPClient, client), GcpScopeMonitoringRead)
			require.NoError(t, err)
			_, err = ts.Token()
			require.Error(t, err)
			assert.Equal(t, 1, requests)
		})
	}
}

func TestGetGCPAuthorizationServiceAccount(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		PodIdentity: kedav1alpha1.AuthPodIdentity{
			Provider:           kedav1alpha1.PodIdentityProviderGCP,
			ServiceAccountName: new("reader"),
			IdentityID:         new("reader@tenant-project.iam.gserviceaccount.com"),
		},
		ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
			Audience: testWorkloadIdentityAudience,
			GetToken: func(context.Context) (string, error) { return "assertion", nil },
		},
	}
	auth, err := GetGCPAuthorization(config)
	require.NoError(t, err)
	assert.Same(t, config.ServiceAccountTokenProvider, auth.ServiceAccountTokenProvider)
	assert.Equal(t, config.PodIdentity.GetIdentityID(), auth.IdentityID)

	for _, identity := range []string{"reader@example.com", "reader@tenant.iam.gserviceaccount.com/../../", "https://elsewhere/token", "reader@tenant.iam.gserviceaccount.com?query"} {
		config.PodIdentity.IdentityID = &identity
		_, err = GetGCPAuthorization(config)
		require.ErrorContains(t, err, "Google service account email")
	}
	config.PodIdentity.IdentityID = nil
	config.ServiceAccountTokenProvider.Audience = "https://kubernetes.default.svc"
	_, err = GetGCPAuthorization(config)
	require.ErrorContains(t, err, "workload identity pool provider")

	config.ServiceAccountTokenProvider = nil
	_, err = GetGCPAuthorization(config)
	require.ErrorContains(t, err, "requires a resolved service account token provider")
	config.ServiceAccountTokenProvider = &scalersconfig.ServiceAccountTokenProvider{Audience: testWorkloadIdentityAudience}
	_, err = GetGCPAuthorization(config)
	require.ErrorContains(t, err, "requires a resolved service account token provider")

	config.PodIdentity.ServiceAccountName = nil
	auth, err = GetGCPAuthorization(config)
	require.NoError(t, err)
	assert.True(t, auth.PodIdentityProviderEnabled)
	assert.Nil(t, auth.ServiceAccountTokenProvider)
	assert.Empty(t, auth.IdentityID)
}
