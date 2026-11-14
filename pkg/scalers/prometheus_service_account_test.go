package scalers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestPrometheusServiceAccountRedirects(t *testing.T) {
	for _, provider := range []kedav1alpha1.PodIdentityProvider{
		kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount,
		kedav1alpha1.PodIdentityProviderAzureWorkloadServiceAccount,
		kedav1alpha1.PodIdentityProviderAwsWorkloadServiceAccount,
	} {
		for _, tc := range []struct {
			name     string
			location string
			blocked  bool
		}{
			{name: "same origin", location: "/redirected"},
			{name: "different host", location: "https://other.example/redirected", blocked: true},
			{name: "downgrade", location: "http://metrics.example/redirected", blocked: true},
		} {
			t.Run(string(provider)+"/"+tc.name, func(t *testing.T) {
				config := azureServiceAccountConfig()
				config.PodIdentity.Provider = provider
				config.TriggerMetadata = map[string]string{
					"serverAddress": "https://metrics.example", "query": "up", "threshold": "1", "awsRegion": "us-east-1",
				}
				switch provider {
				case kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount:
					config.PodIdentity.IdentityID = nil
					config.ServiceAccountTokenProvider.Audience = "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster"
				case kedav1alpha1.PodIdentityProviderAwsWorkloadServiceAccount:
					config.PodIdentity.RoleArn = new("arn:aws:iam::123456789012:role/reader")
					config.ServiceAccountTokenProvider.Audience = "sts.amazonaws.com"
				}
				scaler, err := NewPrometheusScaler(config)
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, scaler.Close(t.Context())) })
				prometheus := scaler.(*prometheusScaler)
				requests := 0
				// Exercise the constructed client's redirect policy with a
				// transport that authenticates every request, including redirects.
				prometheus.httpClient.Transport = &oauth2.Transport{
					Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-cloud-token"}),
					Base: gcpServiceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
						requests++
						require.Equal(t, "Bearer test-cloud-token", request.Header.Get("Authorization"))
						response := &http.Response{Header: make(http.Header), Request: request}
						if requests == 1 {
							response.StatusCode = http.StatusFound
							response.Header.Set("Location", tc.location)
							response.Body = io.NopCloser(strings.NewReader(""))
						} else {
							response.StatusCode = http.StatusOK
							response.Body = io.NopCloser(strings.NewReader(`{"status":"success","data":{"result":[{"value":[1,"2"]}]}}`))
						}
						return response, nil
					}),
				}
				value, err := prometheus.ExecutePromQuery(t.Context())
				if tc.blocked {
					require.ErrorContains(t, err, "different origin")
					require.Equal(t, 1, requests, "the redirected endpoint must never receive credentials")
				} else {
					require.NoError(t, err)
					require.Equal(t, float64(2), value)
					require.Equal(t, 2, requests)
				}
			})
		}
	}
}

func TestPrometheusGCPServiceAccountRedirectDoesNotExposeCloudToken(t *testing.T) {
	previousClient := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previousClient })
	exchanges := 0
	http.DefaultClient = &http.Client{Transport: gcpServiceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
		exchanges++
		require.Equal(t, "https://sts.googleapis.com/v1/token", request.URL.String())
		return &http.Response{
			StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Request: request,
			Body: io.NopCloser(strings.NewReader(`{"access_token":"test-cloud-token","token_type":"Bearer","expires_in":3600}`)),
		}, nil
	})}
	scaler, err := NewPrometheusScaler(&scalersconfig.ScalerConfig{
		PodIdentity: kedav1alpha1.AuthPodIdentity{
			Provider: kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount, ServiceAccountName: new("reader"),
		},
		ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
			Namespace: "tenant", ServiceAccountName: "reader",
			Audience: "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster",
			GetToken: func(context.Context) (string, error) { return "test-kubernetes-assertion", nil },
		},
		TriggerMetadata: map[string]string{
			"serverAddress": "https://metrics.example", "query": "up", "threshold": "1",
		},
		GlobalHTTPTimeout: time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, scaler.Close(t.Context())) })
	prometheus := scaler.(*prometheusScaler)
	transport := prometheus.httpClient.Transport.(*oauth2.Transport)
	requests := 0
	transport.Base = gcpServiceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
		requests++
		require.Equal(t, "metrics.example", request.URL.Host)
		require.Equal(t, "Bearer test-cloud-token", request.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"http://other.example/collect"}},
			Body: io.NopCloser(strings.NewReader("")), Request: request,
		}, nil
	})
	_, err = prometheus.ExecutePromQuery(t.Context())
	require.ErrorContains(t, err, "different origin")
	require.Equal(t, 1, exchanges)
	require.Equal(t, 1, requests)
}
