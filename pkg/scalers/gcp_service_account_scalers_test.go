package scalers

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type gcpServiceAccountRoundTripper func(*http.Request) (*http.Response, error)

func (f gcpServiceAccountRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
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
	t.Run("Stackdriver", func(t *testing.T) {
		scaler, err := NewStackdriverScaler(t.Context(), &scalersconfig.ScalerConfig{
			TriggerMetadata: map[string]string{
				"projectId": "tenant-project", "filter": `metric.type="custom.googleapis.com/queue"`, "targetValue": "5",
			},
			PodIdentity: kedav1alpha1.AuthPodIdentity{
				Provider: kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount, ServiceAccountName: new("reader"),
			},
			ServiceAccountTokenProvider: provider,
		})
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, scaler.Close(t.Context())) })
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
