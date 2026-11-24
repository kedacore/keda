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
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	authenticationv1 "k8s.io/api/authentication/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	mock_serviceaccounts "github.com/kedacore/keda/v2/pkg/mock/mock_serviceaccounts"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
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
	}).SignedString([]byte("test-only"))
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

func TestCloudServiceAccountMissingNameFailsBeforeScalerConstruction(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kedav1alpha1.AddToScheme(scheme))
	triggerAuthentication := &kedav1alpha1.TriggerAuthentication{
		ObjectMeta: metav1.ObjectMeta{Name: "auth", Namespace: "tenant"},
		Spec: kedav1alpha1.TriggerAuthenticationSpec{PodIdentity: &kedav1alpha1.AuthPodIdentity{
			Provider: kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount,
		}},
	}
	handler := &scaleHandler{
		client:   fake.NewClientBuilder().WithScheme(scheme).WithObjects(triggerAuthentication).Build(),
		recorder: events.NewFakeRecorder(1),
	}
	object := &kedav1alpha1.WithTriggers{
		ObjectMeta: metav1.ObjectMeta{Name: "workload", Namespace: "tenant"}, InternalKind: "ScaledObject",
		Spec: kedav1alpha1.WithTriggersSpec{Triggers: []kedav1alpha1.ScaleTriggers{{
			Type: "prometheus",
			Metadata: map[string]string{
				"serverAddress": "https://metrics.example.com", "query": "up", "threshold": "1",
			},
			AuthenticationRef: &kedav1alpha1.AuthenticationRef{Name: "auth"},
		}}},
	}

	_, err := handler.buildScalers(t.Context(), object, nil, "", false)
	require.ErrorContains(t, err, "requires podIdentity.serviceAccountName")
}

func TestMissingAuthRefFailsBeforeScalerConstruction(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kedav1alpha1.AddToScheme(scheme))
	for _, tc := range []struct {
		name      string
		authRef   *kedav1alpha1.AuthenticationRef
		wantError string
	}{
		{name: "missing triggerauth", authRef: &kedav1alpha1.AuthenticationRef{Name: "missing-auth"}, wantError: "not found"},
		{name: "empty authRef name", authRef: &kedav1alpha1.AuthenticationRef{}, wantError: "authenticationRef.name must not be empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := &scaleHandler{
				client:   fake.NewClientBuilder().WithScheme(scheme).Build(),
				recorder: events.NewFakeRecorder(1),
			}
			object := &kedav1alpha1.WithTriggers{
				ObjectMeta: metav1.ObjectMeta{Name: "workload", Namespace: "tenant"}, InternalKind: "ScaledObject",
				Spec: kedav1alpha1.WithTriggersSpec{Triggers: []kedav1alpha1.ScaleTriggers{{
					Type: "prometheus",
					Metadata: map[string]string{
						"serverAddress": "https://monitoring.googleapis.com/v1/projects/tenant-project/location/global/prometheus",
						"query":         "up",
						"threshold":     "1",
					},
					AuthenticationRef: tc.authRef,
				}}},
			}

			_, err := handler.buildScalers(t.Context(), object, nil, "", false)
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}

func TestCloudServiceAccountRejectsUnsupportedScalers(t *testing.T) {
	resolver.SetConfig(&resolver.Config{CloudWorkloadIdentityAudiences: []resolver.CloudWorkloadIdentityAudience{
		{Namespace: "tenant", ServiceAccountName: "gcp-sa", Audience: "https://iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster"},
	}})
	t.Cleanup(func() { resolver.SetConfig(&resolver.Config{}) })
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kedav1alpha1.AddToScheme(scheme))

	identity := kedav1alpha1.AuthPodIdentity{
		Provider:           kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount,
		ServiceAccountName: new("gcp-sa"),
	}
	for _, triggerType := range []string{"cpu", "aws-sqs-queue", "postgresql", "gcp-cloudtasks"} {
		t.Run(triggerType, func(t *testing.T) {
			ta := &kedav1alpha1.TriggerAuthentication{
				ObjectMeta: metav1.ObjectMeta{Name: "auth", Namespace: "tenant"},
				Spec:       kedav1alpha1.TriggerAuthenticationSpec{PodIdentity: &identity},
			}
			handler := &scaleHandler{
				client:   fake.NewClientBuilder().WithScheme(scheme).WithObjects(ta).Build(),
				recorder: events.NewFakeRecorder(1),
			}
			object := &kedav1alpha1.WithTriggers{
				ObjectMeta: metav1.ObjectMeta{Name: "workload", Namespace: "tenant"}, InternalKind: "ScaledObject",
				Spec: kedav1alpha1.WithTriggersSpec{Triggers: []kedav1alpha1.ScaleTriggers{{
					Type:              triggerType,
					Metadata:          map[string]string{"value": "50"},
					AuthenticationRef: &kedav1alpha1.AuthenticationRef{Name: "auth"},
				}}},
			}
			_, err := handler.buildScalers(t.Context(), object, nil, "", false)
			require.ErrorContains(t, err, "is not supported by scaler")
		})
	}
}

func TestCloudServiceAccountUsesScaledObjectNamespace(t *testing.T) {
	const audience = "https://iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster"
	identity := kedav1alpha1.AuthPodIdentity{
		Provider:           kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount,
		ServiceAccountName: new("reader"),
	}
	t.Setenv("KEDA_CLUSTER_OBJECT_NAMESPACE", "keda")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/nonexistent/operator-credentials.json")
	resolver.SetConfig(&resolver.Config{CloudWorkloadIdentityAudiences: []resolver.CloudWorkloadIdentityAudience{
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
			clientset := kubernetesfake.NewClientset()
			var tokenRequestNamespaces []string
			clientset.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
				createAction := action.(k8stesting.CreateAction)
				require.Equal(t, "token", createAction.GetSubresource())
				tokenRequestNamespaces = append(tokenRequestNamespaces, createAction.GetNamespace())
				request := createAction.GetObject().(*authenticationv1.TokenRequest)
				require.Equal(t, []string{audience}, request.Spec.Audiences)
				signingKey := make([]byte, 32)
				_, err := rand.Read(signingKey)
				require.NoError(t, err)
				token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub": "system:serviceaccount:" + createAction.GetNamespace() + ":reader", "aud": []string{audience},
					"exp": time.Now().Add(time.Hour).Unix(),
				}).SignedString(signingKey)
				require.NoError(t, err)
				return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: token}}, nil
			})
			handler := &scaleHandler{
				client: kubeClient, recorder: events.NewFakeRecorder(4),
				authClientSet: &authentication.AuthClientSet{CoreV1Interface: clientset.CoreV1()},
			}
			object := &kedav1alpha1.WithTriggers{
				ObjectMeta: metav1.ObjectMeta{Name: "workload", Namespace: "tenant"}, InternalKind: "ScaledObject",
				Spec: kedav1alpha1.WithTriggersSpec{Triggers: []kedav1alpha1.ScaleTriggers{{
					Type: "prometheus",
					Metadata: map[string]string{
						"serverAddress": "https://monitoring.googleapis.com/v1/projects/tenant-project/location/global/prometheus", "query": "up", "threshold": "1",
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
			_, err = config.ServiceAccountTokenProvider.GetToken(t.Context())
			require.NoError(t, err)
			require.Equal(t, []string{"tenant"}, tokenRequestNamespaces, "TokenRequest must target the ScaledObject namespace, not the operator namespace")
			// This namespace has neither a TA nor an audience mapping, but can
			// reference the same CTA. It must not inherit tenant's delegation.
			if kind == "ClusterTriggerAuthentication" {
				object.Namespace = "other-tenant"
				_, err = handler.buildScalers(t.Context(), object, nil, "", false)
				require.ErrorContains(t, err, "no configured cloud workload identity token audience")
			}
		})
	}
}

func TestCloudServiceAccountRotationRebuildsCachedScaler(t *testing.T) {
	const audience = "https://iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster"
	t.Setenv("KEDA_CLUSTER_OBJECT_NAMESPACE", "keda")
	resolver.SetConfig(&resolver.Config{CloudWorkloadIdentityAudiences: []resolver.CloudWorkloadIdentityAudience{
		{Namespace: "tenant", ServiceAccountName: "reader-a", Audience: audience},
		{Namespace: "tenant", ServiceAccountName: "reader-b", Audience: audience},
	}})
	t.Cleanup(func() { resolver.SetConfig(&resolver.Config{}) })
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kedav1alpha1.AddToScheme(scheme))

	for _, kind := range []string{"TriggerAuthentication", "ClusterTriggerAuthentication"} {
		t.Run(kind, func(t *testing.T) {
			ta := &kedav1alpha1.TriggerAuthentication{
				ObjectMeta: metav1.ObjectMeta{Name: "auth", Namespace: "tenant", UID: "ta-uid-1", Generation: 1},
				Spec: kedav1alpha1.TriggerAuthenticationSpec{PodIdentity: &kedav1alpha1.AuthPodIdentity{
					Provider:           kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount,
					ServiceAccountName: new("reader-a"),
				}},
			}
			cta := &kedav1alpha1.ClusterTriggerAuthentication{
				ObjectMeta: metav1.ObjectMeta{Name: "auth", UID: "cta-uid-1", Generation: 1},
				Spec: kedav1alpha1.TriggerAuthenticationSpec{PodIdentity: &kedav1alpha1.AuthPodIdentity{
					Provider:           kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount,
					ServiceAccountName: new("reader-a"),
				}},
			}
			kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				ta, cta,
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "reader-a", Namespace: "tenant"}},
				&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "reader-b", Namespace: "tenant"}},
			).Build()

			clientset := kubernetesfake.NewClientset()
			var mintedAccounts []string
			clientset.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
				createAction := action.(k8stesting.CreateActionImpl)
				require.Equal(t, "tenant", createAction.GetNamespace())
				mintedAccounts = append(mintedAccounts, createAction.Name)
				signingKey := make([]byte, 32)
				_, err := rand.Read(signingKey)
				require.NoError(t, err)
				token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub": "system:serviceaccount:tenant:" + createAction.Name, "aud": []string{audience},
					"exp": time.Now().Add(time.Hour).Unix(),
				}).SignedString(signingKey)
				require.NoError(t, err)
				return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: token}}, nil
			})

			handler := &scaleHandler{
				client:           kubeClient,
				recorder:         events.NewFakeRecorder(8),
				authClientSet:    &authentication.AuthClientSet{CoreV1Interface: clientset.CoreV1()},
				scalerCaches:     make(map[string]*cache.ScalersCache),
				scalerCachesLock: &sync.RWMutex{},
			}
			scaledJob := &kedav1alpha1.ScaledJob{
				ObjectMeta: metav1.ObjectMeta{Name: "job", Namespace: "tenant", Generation: 1},
				Spec: kedav1alpha1.ScaledJobSpec{
					JobTargetRef: &batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{Name: "worker"}},
							},
						},
					},
					Triggers: []kedav1alpha1.ScaleTriggers{{
						Type: "prometheus",
						Metadata: map[string]string{
							"serverAddress": "https://monitoring.googleapis.com/v1/projects/tenant-project/location/global/prometheus",
							"query":         "up",
							"threshold":     "1",
						},
						AuthenticationRef: &kedav1alpha1.AuthenticationRef{Kind: kind, Name: "auth"},
					}}},
			}

			firstCache, err := handler.GetScalersCache(t.Context(), scaledJob)
			require.NoError(t, err)
			_, firstConfigs := firstCache.GetScalers()
			require.Len(t, firstConfigs, 1)
			_, err = firstConfigs[0].ServiceAccountTokenProvider.GetToken(t.Context())
			require.NoError(t, err)
			require.Equal(t, []string{"reader-a"}, mintedAccounts)

			// Unchanged auth resource reuses the cached scaler without rebuilding.
			sameCache, err := handler.GetScalersCache(t.Context(), scaledJob)
			require.NoError(t, err)
			require.Same(t, firstCache, sameCache)

			// Rotate the TA/CTA without editing the ScaledJob (Generation stays 1).
			if kind == "TriggerAuthentication" {
				ta.Spec.PodIdentity.ServiceAccountName = new("reader-b")
				require.NoError(t, kubeClient.Update(t.Context(), ta))
			} else {
				cta.Spec.PodIdentity.ServiceAccountName = new("reader-b")
				require.NoError(t, kubeClient.Update(t.Context(), cta))
			}

			// Old cached scaler refuses to renew for reader-a.
			_, err = firstConfigs[0].ServiceAccountTokenProvider.GetToken(t.Context())
			require.ErrorContains(t, err, "changed")
			require.Equal(t, []string{"reader-a"}, mintedAccounts)

			// Next GetScalersCache rebuilds for reader-b without editing ScaledJob.
			rotatedCache, err := handler.GetScalersCache(t.Context(), scaledJob)
			require.NoError(t, err)
			require.NotSame(t, firstCache, rotatedCache)
			_, rotatedConfigs := rotatedCache.GetScalers()
			require.Len(t, rotatedConfigs, 1)
			require.Equal(t, "reader-b", *rotatedConfigs[0].PodIdentity.ServiceAccountName)
			_, err = rotatedConfigs[0].ServiceAccountTokenProvider.GetToken(t.Context())
			require.NoError(t, err)
			require.Equal(t, []string{"reader-a", "reader-b"}, mintedAccounts)

			// Deleting the TA/CTA evicts the cache and fails closed.
			if kind == "TriggerAuthentication" {
				require.NoError(t, kubeClient.Delete(t.Context(), ta))
			} else {
				require.NoError(t, kubeClient.Delete(t.Context(), cta))
			}
			_, err = handler.GetScalersCache(t.Context(), scaledJob)
			require.ErrorContains(t, err, "not found")
		})
	}
}
