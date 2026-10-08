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

package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
)

const testGCPAudience = "https://iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster"

func TestPodIdentityServiceAccountApproval(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Namespace: "tenant", Name: "reader"},
	}).Build()
	for _, tt := range []struct {
		name      string
		config    Config
		namespace string
		wantError string
	}{
		{name: "unconfigured", namespace: "tenant", wantError: "no configured token audience"},
		{name: "global audience is not delegation", namespace: "tenant", config: Config{
			ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{Audience: testGCPAudience}},
		}, wantError: "no configured token audience"},
		{name: "another namespace", namespace: "other", config: Config{
			ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{Namespace: "tenant", ServiceAccountName: "reader", Audience: testGCPAudience}},
		}, wantError: "no configured token audience"},
		{name: "operator account approval does not authorize tenant account", namespace: "tenant", config: Config{
			ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{Namespace: "keda", ServiceAccountName: "reader", Audience: testGCPAudience}},
		}, wantError: "no configured token audience"},
		{name: "legacy cannot bypass approval", namespace: "tenant", config: Config{ServiceAccountTokenMode: "legacy"}, wantError: "requires enforce-audience"},
		{name: "API audience", namespace: "tenant", config: Config{
			ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{Namespace: "tenant", ServiceAccountName: "reader", Audience: "https://kubernetes.default.svc"}},
		}, wantError: "audience"},
		{name: "approved account", namespace: "tenant", config: Config{
			ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{Namespace: "tenant", ServiceAccountName: "reader", Audience: testGCPAudience}},
		}},
		{name: "missing account", namespace: "missing", config: Config{
			ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{Namespace: "missing", ServiceAccountName: "reader", Audience: testGCPAudience}},
		}, wantError: "failed to get pod identity service account"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			SetConfig(&tt.config)
			tokenClient := kubernetesfake.NewClientset()
			provider, err := ResolveServiceAccountTokenProvider(t.Context(), kubeClient, kedav1alpha1.AuthPodIdentity{
				Provider: kedav1alpha1.PodIdentityProviderGCP, ServiceAccountName: new("reader"),
			}, tt.namespace, &authentication.AuthClientSet{CoreV1Interface: tokenClient.CoreV1()})
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				require.NotNil(t, provider)
				assert.Equal(t, testGCPAudience, provider.Audience)
				assert.Equal(t, "tenant", provider.Namespace)
				assert.Equal(t, "reader", provider.ServiceAccountName)
			}
			assert.Empty(t, tokenClient.Actions(), "resolution must not mint or expose an assertion")
		})
	}
}

func TestPodIdentityCloudAudienceSelection(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Namespace: "tenant", Name: "reader"},
	}).Build()
	for _, provider := range []kedav1alpha1.PodIdentityProvider{
		kedav1alpha1.PodIdentityProviderGCP, kedav1alpha1.PodIdentityProviderAzureWorkload, kedav1alpha1.PodIdentityProviderAws,
	} {
		for _, audience := range []string{testGCPAudience, "api://AzureADTokenExchange", "sts.amazonaws.com", "https://kubernetes.default.svc"} {
			t.Run(string(provider)+"/"+audience, func(t *testing.T) {
				SetConfig(&Config{ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{
					{Namespace: "tenant", ServiceAccountName: "reader", Audience: audience},
				}})
				identity := kedav1alpha1.AuthPodIdentity{Provider: provider, ServiceAccountName: new("reader")}
				if provider == kedav1alpha1.PodIdentityProviderAzureWorkload {
					identity.IdentityID, identity.IdentityTenantID = new("client-id"), new("tenant-id")
				}
				if provider == kedav1alpha1.PodIdentityProviderAws {
					identity.RoleArn = new("arn:aws:iam::123456789012:role/reader")
				}
				tokenClient := kubernetesfake.NewClientset()
				resolved, err := ResolveServiceAccountTokenProvider(t.Context(), kubeClient, identity, "tenant",
					&authentication.AuthClientSet{CoreV1Interface: tokenClient.CoreV1()})
				validAudience := map[kedav1alpha1.PodIdentityProvider]string{
					kedav1alpha1.PodIdentityProviderGCP:           testGCPAudience,
					kedav1alpha1.PodIdentityProviderAzureWorkload: "api://AzureADTokenExchange",
					kedav1alpha1.PodIdentityProviderAws:           "sts.amazonaws.com",
				}[provider]
				if audience == validAudience {
					require.NoError(t, err)
					require.NotNil(t, resolved)
					assert.Equal(t, audience, resolved.Audience)
				} else {
					require.ErrorContains(t, err, "audience")
					assert.Nil(t, resolved)
				}
				assert.Empty(t, tokenClient.Actions(), "invalid audiences must be rejected before token minting")
			})
		}
	}
}

func TestPodIdentityServiceAccountRefresh(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	cfg := Config{ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{
		{Namespace: "tenant", ServiceAccountName: "reader", Audience: testGCPAudience},
	}}
	SetConfig(&cfg)
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Namespace: "tenant", Name: "reader"},
	}).Build()
	tokenClient := kubernetesfake.NewClientset()
	want := testServiceAccountJWT(t, testGCPAudience)
	var tokenError error
	tokenClient.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
		assert.Equal(t, "tenant", action.GetNamespace())
		assert.Equal(t, "token", action.GetSubresource())
		assert.Equal(t, "reader", action.(k8stesting.CreateActionImpl).Name)
		request := action.(k8stesting.CreateAction).GetObject().(*authenticationv1.TokenRequest)
		assert.Equal(t, []string{testGCPAudience}, request.Spec.Audiences)
		return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: want}}, tokenError
	})
	ctx, cancel := context.WithCancel(t.Context())
	provider, err := ResolveServiceAccountTokenProvider(ctx, kubeClient, kedav1alpha1.AuthPodIdentity{
		Provider: kedav1alpha1.PodIdentityProviderGCP, ServiceAccountName: new("reader"),
	}, "tenant", &authentication.AuthClientSet{CoreV1Interface: tokenClient.CoreV1()})
	require.NoError(t, err)
	cancel()
	for range 2 {
		token, err := provider.GetToken(t.Context())
		require.NoError(t, err, "refresh must use the current context, not the canceled factory context")
		assert.Equal(t, want, token)
	}
	assert.Len(t, tokenClient.Actions(), 2, "each callback must request a fresh assertion")

	tokenError = errors.New("token creation forbidden")
	token, err := provider.GetToken(t.Context())
	require.ErrorIs(t, err, tokenError)
	assert.Empty(t, token)
	tokenError = nil
	want = testServiceAccountJWT(t, testGCPAudience, "https://kubernetes.default.svc")
	token, err = provider.GetToken(t.Context())
	require.Error(t, err, "returned API-valid tokens must not reach Google STS")
	assert.Empty(t, token)

	actionsBeforeRevocation := len(tokenClient.Actions())
	for _, replacement := range []Config{
		{},
		{ServiceAccountTokenMode: "legacy"},
		{ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{
			{Namespace: "tenant", ServiceAccountName: "reader", Audience: testGCPAudience + "-changed"},
		}},
	} {
		SetConfig(&replacement)
		token, err = provider.GetToken(t.Context())
		require.Error(t, err)
		assert.Empty(t, token)
	}
	assert.Len(t, tokenClient.Actions(), actionsBeforeRevocation, "revoked or changed approval must prevent token creation")
}

func TestPodIdentityWithoutServiceAccountPreservesDefault(t *testing.T) {
	provider, err := ResolveServiceAccountTokenProvider(t.Context(), nil, kedav1alpha1.AuthPodIdentity{
		Provider: kedav1alpha1.PodIdentityProviderGCP,
	}, "tenant", nil)
	require.NoError(t, err)
	assert.Nil(t, provider)
}
