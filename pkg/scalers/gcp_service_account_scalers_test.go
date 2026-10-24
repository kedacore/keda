package scalers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type gcpServiceAccountRoundTripper func(*http.Request) (*http.Response, error)

func (f gcpServiceAccountRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestGCPServiceAccountTokenScopes(t *testing.T) {
	for _, tc := range []struct {
		name        string
		tokenSource func(context.Context, *gcp.AuthorizationMetadata) (oauth2.TokenSource, error)
		wantScope   string
	}{
		{name: "Monitoring", tokenSource: newGCPMonitoringTokenSource, wantScope: gcp.GcpScopeMonitoringRead},
		{name: "Spanner", tokenSource: newGCPSpannerTokenSource, wantScope: spanner.Scope},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotScope string
			httpClient := &http.Client{Transport: gcpServiceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
				require.Equal(t, "https://sts.googleapis.com/v1/token", request.URL.String())
				require.NoError(t, request.ParseForm())
				gotScope = request.Form.Get("scope")
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"access-token","token_type":"Bearer","expires_in":3600}`)),
				}, nil
			})}
			auth := &gcp.AuthorizationMetadata{ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
				Audience: "https://iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster",
				GetToken: func(context.Context) (string, error) { return "assertion", nil },
			}}
			tokenSource, err := tc.tokenSource(context.WithValue(t.Context(), oauth2.HTTPClient, httpClient), auth)
			require.NoError(t, err)
			_, err = tokenSource.Token()
			require.NoError(t, err)
			assert.Equal(t, tc.wantScope, gotScope)
		})
	}
}

func TestGCPSpannerServiceAccountDisablesNativeMetrics(t *testing.T) {
	assert.True(t, gcpSpannerServiceAccountClientConfig().DisableNativeMetrics,
		"native metrics require a Cloud Monitoring write scope and must stay disabled for the data-only credential")
}

func TestGCPServiceAccountClientsDoNotUseOperatorCredentials(t *testing.T) {
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing-operator-credentials.json"))
	t.Setenv("CLOUDSDK_CORE_PROJECT", "")
	t.Setenv("SPANNER_EMULATOR_HOST", "")
	t.Setenv("STORAGE_EMULATOR_HOST", "")
	provider := &scalersconfig.ServiceAccountTokenProvider{
		Audience: "https://iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster",
		GetToken: func(context.Context) (string, error) {
			return "", errors.New("token requests are unavailable in this test")
		},
	}
	auth := &gcp.AuthorizationMetadata{PodIdentityProviderEnabled: true, ServiceAccountTokenProvider: provider}

	t.Run("PubSub", func(t *testing.T) {
		scaler := &pubsubScaler{metadata: &pubsubMetadata{gcpAuthorization: auth}}
		require.NoError(t, scaler.setStackdriverClient(t.Context()))
		t.Cleanup(func() { require.NoError(t, scaler.Close(t.Context())) })
	})
	t.Run("CloudTasks", func(t *testing.T) {
		scaler := &gcpCloudTasksScaler{metadata: &gcpCloudTaskMetadata{gcpAuthorization: auth}}
		require.NoError(t, scaler.setStackdriverClient(t.Context()))
		t.Cleanup(func() { require.NoError(t, scaler.Close(t.Context())) })
	})
	t.Run("Stackdriver", func(t *testing.T) {
		client, err := initializeStackdriverClient(t.Context(), auth, logr.Discard())
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, client.Close()) })
	})
	t.Run("Storage", func(t *testing.T) {
		scaler, err := NewGcsScaler(&scalersconfig.ScalerConfig{
			TriggerMetadata: map[string]string{"bucketName": "tenant-bucket"},
			PodIdentity: kedav1alpha1.AuthPodIdentity{
				Provider: kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount, ServiceAccountName: new("reader"),
			},
			ServiceAccountTokenProvider: provider,
		})
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, scaler.Close(t.Context())) })
	})
	t.Run("Spanner", func(t *testing.T) {
		client, err := newSpannerClient(t.Context(), &spannerMetadata{
			ProjectID: "tenant-project", InstanceID: "instance", DatabaseID: "database", gcpAuthorization: auth,
		})
		require.NoError(t, err)
		t.Cleanup(client.Close)
	})
}
