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

package scaling

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	authenticationv1 "k8s.io/api/authentication/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	mock_serviceaccounts "github.com/kedacore/keda/v2/pkg/mock/mock_serviceaccounts"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

func TestScalerFactoryUsesContextFromCurrentInvocation(t *testing.T) {
	const (
		namespace                 = "default"
		triggerAuthenticationName = "auth"
		serviceAccountName        = "scaler"
		audience                  = "metrics"
	)

	resolver.SetConfig(&resolver.Config{
		ServiceAccountTokenMode: "enforce-audience",
		ServiceAccountTokenAudiences: []resolver.ServiceAccountTokenAudience{
			{Namespace: namespace, ServiceAccountName: serviceAccountName, Audience: audience},
		},
	})
	t.Cleanup(func() { resolver.SetConfig(&resolver.Config{}) })
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "system:serviceaccount:" + namespace + ":" + serviceAccountName,
		"aud": []string{audience},
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(newTestJWTSigningKey(t))
	require.NoError(t, err)

	testScheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(testScheme))
	require.NoError(t, kedav1alpha1.AddToScheme(testScheme))

	triggerAuthentication := &kedav1alpha1.TriggerAuthentication{
		ObjectMeta: metav1.ObjectMeta{Name: triggerAuthenticationName, Namespace: namespace},
		Spec: kedav1alpha1.TriggerAuthenticationSpec{
			BoundServiceAccountToken: []kedav1alpha1.BoundServiceAccountToken{{
				Parameter:          "token",
				ServiceAccountName: serviceAccountName,
			}},
		},
	}
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: namespace},
	}

	ctrl := gomock.NewController(t)
	coreClient := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	coreClient.GetServiceAccountInterface().EXPECT().CreateToken(
		gomock.Any(), serviceAccountName, gomock.Any(), gomock.Any(),
	).DoAndReturn(func(ctx context.Context, _ string, request *authenticationv1.TokenRequest, _ metav1.CreateOptions) (*authenticationv1.TokenRequest, error) {
		require.Equal(t, []string{audience}, request.Spec.Audiences)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: token}}, nil
	}).Times(2)

	handler := &scaleHandler{
		client:   fake.NewClientBuilder().WithScheme(testScheme).WithObjects(triggerAuthentication, serviceAccount).Build(),
		recorder: events.NewFakeRecorder(2),
		authClientSet: &authentication.AuthClientSet{
			CoreV1Interface: coreClient,
		},
	}
	withTriggers := &kedav1alpha1.WithTriggers{
		ObjectMeta:   metav1.ObjectMeta{Name: "scaled-object", Namespace: namespace},
		InternalKind: "ScaledObject",
		Spec: kedav1alpha1.WithTriggersSpec{Triggers: []kedav1alpha1.ScaleTriggers{{
			Type:       "cpu",
			Metadata:   map[string]string{"value": "50"},
			MetricType: autoscalingv2.UtilizationMetricType,
			AuthenticationRef: &kedav1alpha1.AuthenticationRef{
				Name: triggerAuthenticationName,
			},
		}}},
	}

	initialCtx, cancelInitial := context.WithCancel(context.Background())
	builders, err := handler.buildScalers(initialCtx, withTriggers, nil, "", false)
	require.NoError(t, err)
	require.Len(t, builders, 1)
	cancelInitial()

	refreshCtx := context.Background()
	refreshedScaler, config, err := builders[0].Factory(refreshCtx)
	require.NoError(t, err)
	require.Equal(t, token, config.AuthParams["token"])
	require.NoError(t, refreshedScaler.Close(refreshCtx))
	require.NoError(t, builders[0].Scaler.Close(refreshCtx))
}

func TestCloudServiceAccountUsesScaledObjectNamespace(t *testing.T) {
	for _, tt := range []struct {
		identity kedav1alpha1.AuthPodIdentity
		audience string
	}{
		{
			identity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderGCP, ServiceAccountName: new("reader")},
			audience: "https://iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster",
		},
		{
			identity: kedav1alpha1.AuthPodIdentity{
				Provider: kedav1alpha1.PodIdentityProviderAzureWorkload, ServiceAccountName: new("reader"),
				IdentityID: new("client-id"), IdentityTenantID: new("tenant-id"),
			},
			audience: "api://AzureADTokenExchange",
		},
		{
			identity: kedav1alpha1.AuthPodIdentity{
				Provider: kedav1alpha1.PodIdentityProviderAws, ServiceAccountName: new("reader"),
				RoleArn: new("arn:aws:iam::123456789012:role/reader"),
			},
			audience: "sts.amazonaws.com",
		},
	} {
		t.Run(string(tt.identity.Provider), func(t *testing.T) {
			testCloudServiceAccountNamespace(t, tt.identity, tt.audience)
		})
	}
}

func testCloudServiceAccountNamespace(t *testing.T, identity kedav1alpha1.AuthPodIdentity, audience string) {
	t.Helper()
	t.Setenv("KEDA_CLUSTER_OBJECT_NAMESPACE", "keda")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/nonexistent/operator-credentials.json")
	resolver.SetConfig(&resolver.Config{ServiceAccountTokenAudiences: []resolver.ServiceAccountTokenAudience{
		{Namespace: "tenant", ServiceAccountName: "reader", Audience: audience},
	}})
	t.Cleanup(func() { resolver.SetConfig(&resolver.Config{}) })
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kedav1alpha1.AddToScheme(scheme))
	spec := kedav1alpha1.TriggerAuthenticationSpec{PodIdentity: &identity}
	for _, kind := range []string{"TriggerAuthentication", "ClusterTriggerAuthentication"} {
		t.Run(kind, func(t *testing.T) {
			kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kedav1alpha1.TriggerAuthentication{ObjectMeta: metav1.ObjectMeta{Name: "auth", Namespace: "tenant"}, Spec: spec},
				&kedav1alpha1.ClusterTriggerAuthentication{ObjectMeta: metav1.ObjectMeta{Name: "auth"}, Spec: spec},
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "reader", Namespace: "tenant"}},
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "reader", Namespace: "keda"}},
			).Build()
			coreClient := mock_serviceaccounts.NewMockCoreV1Interface(gomock.NewController(t))
			coreClient.GetServiceAccountInterface().EXPECT().CreateToken(
				gomock.Any(), "reader", gomock.Any(), gomock.Any(),
			).DoAndReturn(func(ctx context.Context, _ string, request *authenticationv1.TokenRequest, _ metav1.CreateOptions) (*authenticationv1.TokenRequest, error) {
				require.NoError(t, ctx.Err())
				require.Equal(t, []string{audience}, request.Spec.Audiences)
				token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub": "system:serviceaccount:tenant:reader", "aud": []string{audience},
					"exp": time.Now().Add(time.Hour).Unix(),
				}).SignedString(newTestJWTSigningKey(t))
				require.NoError(t, err)
				return &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: token}}, nil
			}).Times(1)
			handler := &scaleHandler{
				client: kubeClient, recorder: events.NewFakeRecorder(4),
				authClientSet: &authentication.AuthClientSet{CoreV1Interface: coreClient},
			}
			object := &kedav1alpha1.WithTriggers{
				ObjectMeta: metav1.ObjectMeta{Name: "workload", Namespace: "tenant"}, InternalKind: "ScaledObject",
				Spec: kedav1alpha1.WithTriggersSpec{Triggers: []kedav1alpha1.ScaleTriggers{{
					Type: "prometheus",
					Metadata: map[string]string{
						"serverAddress": "https://metrics.example.com", "query": "up", "threshold": "1", "awsRegion": "us-east-1",
					},
					AuthenticationRef: &kedav1alpha1.AuthenticationRef{Kind: kind, Name: "auth"},
				}}},
			}
			ctx, cancel := context.WithCancel(t.Context())
			builders, err := handler.buildScalers(ctx, object, nil, "", false)
			require.NoError(t, err)
			cancel()
			require.Len(t, builders, 1)
			t.Cleanup(func() { require.NoError(t, builders[0].Scaler.Close(t.Context())) })
			config := builders[0].ScalerConfig
			require.Empty(t, config.AuthParams, "Kubernetes assertions must not become generic scaler auth parameters")
			require.NotNil(t, config.ServiceAccountTokenProvider)
			require.Equal(t, audience, config.ServiceAccountTokenProvider.Audience)
			require.Equal(t, "tenant", config.ServiceAccountTokenProvider.Namespace)
			require.Equal(t, "reader", config.ServiceAccountTokenProvider.ServiceAccountName)
			_, err = config.ServiceAccountTokenProvider.GetToken(t.Context())
			require.NoError(t, err)
			// This namespace has neither a TA nor an audience mapping, but can
			// reference the same CTA. It must not inherit tenant's delegation.
			if kind == "ClusterTriggerAuthentication" {
				object.Namespace = "other-tenant"
				_, err = handler.buildScalers(t.Context(), object, nil, "", false)
				require.ErrorContains(t, err, "no configured token audience")
			}
		})
	}
}

func newTestJWTSigningKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}
