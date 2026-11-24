package scalers

import (
	"context"
	"errors"
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

const testGMPServerAddress = "https://monitoring.googleapis.com/v1/projects/tenant-project/location/global/prometheus"

func newPrometheusWorkloadServiceAccountConfig(metadata, authParams map[string]string) *scalersconfig.ScalerConfig {
	return &scalersconfig.ScalerConfig{
		PodIdentity: kedav1alpha1.AuthPodIdentity{
			Provider: kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount, ServiceAccountName: new("reader"),
		},
		ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
			Audience: "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster",
			GetToken: func(context.Context) (string, error) {
				return "", errors.New("token requests are unavailable in this test")
			},
		},
		TriggerMetadata:   metadata,
		AuthParams:        authParams,
		GlobalHTTPTimeout: time.Second,
	}
}

func TestPrometheusWorkloadServiceAccountEndpointPolicy(t *testing.T) {
	for _, tc := range []struct {
		name          string
		serverAddress string
		unsafeSsl     bool
		wantError     string
	}{
		{name: "managed prometheus", serverAddress: testGMPServerAddress},
		{name: "managed prometheus with explicit port", serverAddress: "https://monitoring.googleapis.com:443/v1/projects/tenant-project/location/global/prometheus"},
		{name: "managed prometheus with mixed-case host", serverAddress: "https://Monitoring.GoogleAPIs.com/v1/projects/tenant-project/location/global/prometheus"},
		{name: "self-hosted server", serverAddress: "https://metrics.example.com", wantError: "requires serverAddress to use https://monitoring.googleapis.com"},
		{name: "plaintext", serverAddress: "http://monitoring.googleapis.com/v1/projects/tenant-project/location/global/prometheus", wantError: "requires serverAddress"},
		{name: "lookalike host", serverAddress: "https://monitoring.googleapis.com.evil.example/v1", wantError: "requires serverAddress"},
		{name: "subdomain", serverAddress: "https://evil.monitoring.googleapis.com/v1", wantError: "requires serverAddress"},
		{name: "user information", serverAddress: "https://user@monitoring.googleapis.com/v1", wantError: "requires serverAddress"},
		{name: "non-default port", serverAddress: "https://monitoring.googleapis.com:8443/v1", wantError: "requires serverAddress"},
		{name: "relative address", serverAddress: "monitoring.googleapis.com/v1", wantError: "requires serverAddress"},
		{name: "query string", serverAddress: testGMPServerAddress + "?foo=bar", wantError: "cannot contain a query or fragment"},
		{name: "trailing query", serverAddress: testGMPServerAddress + "?", wantError: "cannot contain a query or fragment"},
		{name: "fragment", serverAddress: testGMPServerAddress + "#frag", wantError: "cannot contain a query or fragment"},
		{name: "trailing fragment", serverAddress: testGMPServerAddress + "#", wantError: "cannot contain a query or fragment"},
		{name: "unsafe TLS", serverAddress: testGMPServerAddress, unsafeSsl: true, wantError: "unsafeSsl cannot be enabled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metadata := map[string]string{"serverAddress": tc.serverAddress, "query": "up", "threshold": "1"}
			if tc.unsafeSsl {
				metadata["unsafeSsl"] = "true"
			}
			scaler, err := NewPrometheusScaler(newPrometheusWorkloadServiceAccountConfig(metadata, nil))
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				require.Nil(t, scaler)
				return
			}
			require.NoError(t, err)
			require.NoError(t, scaler.Close(t.Context()))
		})
	}
}

func TestPrometheusWorkloadServiceAccountRejectsOtherAuthModes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		authModes  string
		authParams map[string]string
	}{
		{name: "bearer", authModes: "bearer", authParams: map[string]string{"bearerToken": "token"}},
		{name: "basic", authModes: "basic", authParams: map[string]string{"username": "user", "password": "password"}},
		{name: "tls", authModes: "tls", authParams: map[string]string{"ca": "ca", "cert": "cert", "key": "key"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scaler, err := NewPrometheusScaler(newPrometheusWorkloadServiceAccountConfig(map[string]string{
				"serverAddress": testGMPServerAddress, "query": "up", "threshold": "1", "authModes": tc.authModes,
			}, tc.authParams))
			require.ErrorContains(t, err, "pod identity cannot be enabled with other auth types")
			require.Nil(t, scaler)
		})
	}
}

func TestPrometheusServiceAccountRedirects(t *testing.T) {
	for _, tc := range []struct {
		name     string
		location string
		blocked  bool
	}{
		{name: "same origin", location: "/redirected"},
		{name: "different host", location: "https://other.example/redirected", blocked: true},
		{name: "downgrade", location: "http://monitoring.googleapis.com/redirected", blocked: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scaler, err := NewPrometheusScaler(newPrometheusWorkloadServiceAccountConfig(map[string]string{
				"serverAddress": testGMPServerAddress, "query": "up", "threshold": "1",
			}, nil))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, scaler.Close(t.Context())) })
			prometheus := scaler.(*prometheusScaler)
			// Keep the GCP transport built by the scaler, which authenticates every
			// request including redirects, and stub only its token and network.
			transport := prometheus.httpClient.Transport.(*oauth2.Transport)
			transport.Source = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-cloud-token"})
			requests := 0
			transport.Base = gcpServiceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
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
			})
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

func TestPrometheusWorkloadServiceAccountIgnoresAWSOperatorCredentials(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-operator-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-operator-secret-key")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	metadata := map[string]string{
		"serverAddress": testGMPServerAddress,
		"query":         "up",
		"threshold":     "1",
		"awsRegion":     "us-east-1",
		"identityOwner": "operator",
	}

	// Prove that the same metadata selects AWS SigV4 when workload service
	// account pod identity is not active.
	awsBaseline, err := NewPrometheusScaler(&scalersconfig.ScalerConfig{
		PodIdentity:       kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		TriggerMetadata:   metadata,
		TriggerUniqueKey:  "prometheus-aws-baseline",
		GlobalHTTPTimeout: time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, awsBaseline.Close(t.Context())) })
	_, baselineIsOAuth := awsBaseline.(*prometheusScaler).httpClient.Transport.(*oauth2.Transport)
	require.False(t, baselineIsOAuth, "baseline config with awsRegion and identityOwner=operator should select AWS SigV4")

	// With gcp-workload-service-account, operator AWS credentials must not
	// replace the GCP OAuth2 transport.
	config := newPrometheusWorkloadServiceAccountConfig(metadata, nil)
	config.TriggerUniqueKey = "prometheus-gcp-sa-no-aws-fallback"
	scaler, err := NewPrometheusScaler(config)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, scaler.Close(t.Context())) })
	prometheus := scaler.(*prometheusScaler)
	transport, ok := prometheus.httpClient.Transport.(*oauth2.Transport)
	require.True(t, ok, "GCP workload service account must retain oauth2.Transport")
	transport.Source = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-cloud-token"})
	requests := 0
	transport.Base = gcpServiceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
		requests++
		require.Equal(t, "Bearer test-cloud-token", request.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Request:    request,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"result":[{"value":[1,"7"]}]}}`)),
		}, nil
	})
	value, err := prometheus.ExecutePromQuery(t.Context())
	require.NoError(t, err)
	require.Equal(t, float64(7), value)
	require.Equal(t, 1, requests)
}
