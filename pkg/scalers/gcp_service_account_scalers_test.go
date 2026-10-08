package scalers

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

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
				Provider: kedav1alpha1.PodIdentityProviderGCP, ServiceAccountName: new("reader"),
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
