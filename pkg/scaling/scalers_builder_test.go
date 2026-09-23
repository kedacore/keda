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
