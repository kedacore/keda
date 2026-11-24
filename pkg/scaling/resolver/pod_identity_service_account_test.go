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
	"os"
	"path/filepath"
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

const (
	testGCPAudience       = "https://iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/kubernetes/providers/cluster"
	testSecondGCPAudience = "//iam.googleapis.com/projects/987654321/locations/global/workloadIdentityPools/other/providers/cluster"
	testGKEAudience       = "identitynamespace:tenant-project.svc.id.goog:https://container.googleapis.com/v1/projects/tenant-project/locations/us-central1/clusters/prod"
	testGKETokenAudience  = "tenant-project.svc.id.goog"
)

func TestCloudWorkloadIdentityAudienceConfiguration(t *testing.T) {
	for _, value := range []string{
		`[{"namespace":"tenant","serviceAccountName":"gcp-reader","audience":"` + testGCPAudience + `"},{"namespace":"tenant","serviceAccountName":"gcp-writer","audience":"` + testSecondGCPAudience + `"}]`,
		"- namespace: tenant\n  serviceAccountName: gcp-reader\n  audience: " + testGCPAudience + "\n- namespace: tenant\n  serviceAccountName: gcp-writer\n  audience: " + testSecondGCPAudience + "\n",
		`[{"namespace":"tenant","serviceAccountName":"gke-reader","audience":"` + testGKEAudience + `"}]`,
		"- namespace: tenant\n  serviceAccountName: gke-reader\n  audience: " + testGKEAudience + "\n",
	} {
		cfg := Config{}
		require.NoError(t, cfg.LoadCloudWorkloadIdentityAudiences(value))
		require.NoError(t, cfg.Validate())
		assert.Empty(t, cfg.serviceAccountTokenAllowedAudiences(), "cloud workload identity audiences must not widen Vault file-token audiences")
	}
	for _, value := range []string{
		"null", "{}", "[", "[]\n---\n[]",
		`[{"audience":"` + testGCPAudience + `"}]`,
		`[{"namespace":"tenant","audience":"` + testGCPAudience + `"}]`,
		`[{"serviceAccountName":"reader","audience":"` + testGCPAudience + `"}]`,
		`[{"namespace":"tenant","serviceAccountName":"reader","audience":""}]`,
		`[{"namespace":"tenant","serviceAccountName":"reader","audience":" ` + testGCPAudience + `"}]`,
		`[{"namespace":"tenant","serviceAccountName":"reader","audience":"https://kubernetes.default.svc"}]`,
		`[{"namespace":"tenant","serviceAccountName":"reader","audience":"vault"}]`,
		`[{"namespace":"*","serviceAccountName":"reader","audience":"` + testGCPAudience + `"}]`,
		`[{"namespace":"tenant","serviceAccountName":"*","audience":"` + testGCPAudience + `"}]`,
		`[{"namespace":"tenant","serviceAccountName":"reader","audience":"` + testGCPAudience + `","extra":true}]`,
		`[{"namespace":"tenant","serviceAccountName":"reader","audience":"` + testGCPAudience + `"},{"namespace":"tenant","serviceAccountName":"reader","audience":"` + testSecondGCPAudience + `"}]`,
	} {
		t.Run(value, func(t *testing.T) {
			cfg := Config{}
			assert.Error(t, cfg.LoadCloudWorkloadIdentityAudiences(value))
		})
	}
	for _, value := range []string{"", "[]"} {
		for _, mode := range []string{"enforce-audience", "legacy"} {
			cfg := Config{ServiceAccountTokenMode: mode}
			require.NoError(t, cfg.LoadCloudWorkloadIdentityAudiences(value))
			require.NoError(t, cfg.Validate())
		}
	}
	assert.Error(t, (&Config{
		ServiceAccountTokenMode: "legacy",
		CloudWorkloadIdentityAudiences: []CloudWorkloadIdentityAudience{
			{Namespace: "tenant", ServiceAccountName: "reader", Audience: testGCPAudience},
		},
	}).Validate())
}

func TestCloudWorkloadIdentityAudiencesIsolatedFromVaultAndBSAT(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	path := filepath.Join(t.TempDir(), "token")
	SetConfig(&Config{
		VaultKubernetesAuthTokenFile: path,
		CloudWorkloadIdentityAudiences: []CloudWorkloadIdentityAudience{
			{Namespace: "tenant", ServiceAccountName: "gcp-reader", Audience: testGCPAudience},
		},
	})

	// Vault file tokens must not accept cloud workload identity audiences.
	token := testServiceAccountJWT(t, testGCPAudience)
	require.NoError(t, os.WriteFile(path, []byte(token), 0o600))
	vh := NewHashicorpVaultHandler(&kedav1alpha1.HashiCorpVault{
		Credential: &kedav1alpha1.Credential{ServiceAccount: path},
	}, nil, "tenant")
	got, err := vh.kubernetesToken(t.Context())
	require.Error(t, err, "Vault file token must reject cloud audience")
	assert.Empty(t, got)

	// Vault serviceAccountName and generic BSAT must not mint using cloud workload identity approvals.
	tokenClient := kubernetesfake.NewClientset()
	acs := &authentication.AuthClientSet{CoreV1Interface: tokenClient.CoreV1()}
	vh = NewHashicorpVaultHandler(&kedav1alpha1.HashiCorpVault{
		Credential: &kedav1alpha1.Credential{ServiceAccountName: "gcp-reader"},
	}, acs, "tenant")
	got, err = vh.kubernetesToken(t.Context())
	require.ErrorContains(t, err, "no configured token audience")
	assert.Empty(t, got)

	bsatToken, err := GenerateBoundServiceAccountToken(t.Context(), "gcp-reader", "tenant", acs)
	require.ErrorContains(t, err, "no configured token audience")
	assert.Empty(t, bsatToken)
	assert.Empty(t, tokenClient.Actions(), "isolated cloud approvals must never trigger BSAT/Vault TokenRequest")

	// Conversely, KEDA_SERVICE_ACCOUNT_TOKEN_AUDIENCES must never authorize cloud workload identity.
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Namespace: "tenant", Name: "reader"},
	}).Build()
	SetConfig(&Config{
		ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{
			{Namespace: "tenant", ServiceAccountName: "reader", Audience: testGCPAudience},
		},
	})
	provider, err := ResolveServiceAccountTokenProvider(t.Context(), kubeClient, nil, kedav1alpha1.AuthPodIdentity{
		Provider: kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount, ServiceAccountName: new("reader"),
	}, "tenant", acs)
	require.ErrorContains(t, err, CloudWorkloadIdentityAudiencesEnvVar)
	assert.Nil(t, provider)
}

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
		{name: "unconfigured", namespace: "tenant", wantError: "no configured cloud workload identity token audience"},
		{name: "global audience is not delegation", namespace: "tenant", config: Config{
			CloudWorkloadIdentityAudiences: []CloudWorkloadIdentityAudience{{Audience: testGCPAudience}},
		}, wantError: "no configured cloud workload identity token audience"},
		{name: "another namespace", namespace: "other", config: Config{
			CloudWorkloadIdentityAudiences: []CloudWorkloadIdentityAudience{{Namespace: "tenant", ServiceAccountName: "reader", Audience: testGCPAudience}},
		}, wantError: "no configured cloud workload identity token audience"},
		{name: "operator account approval does not authorize tenant account", namespace: "tenant", config: Config{
			CloudWorkloadIdentityAudiences: []CloudWorkloadIdentityAudience{{Namespace: "keda", ServiceAccountName: "reader", Audience: testGCPAudience}},
		}, wantError: "no configured cloud workload identity token audience"},
		{name: "legacy mode grants no approval", namespace: "tenant", config: Config{ServiceAccountTokenMode: "legacy"}, wantError: "no configured cloud workload identity token audience"},
		{name: "API audience", namespace: "tenant", config: Config{
			CloudWorkloadIdentityAudiences: []CloudWorkloadIdentityAudience{{Namespace: "tenant", ServiceAccountName: "reader", Audience: "https://kubernetes.default.svc"}},
		}, wantError: "audience"},
		{name: "approved account", namespace: "tenant", config: Config{
			CloudWorkloadIdentityAudiences: []CloudWorkloadIdentityAudience{{Namespace: "tenant", ServiceAccountName: "reader", Audience: testGCPAudience}},
		}},
		{name: "missing account", namespace: "missing", config: Config{
			CloudWorkloadIdentityAudiences: []CloudWorkloadIdentityAudience{{Namespace: "missing", ServiceAccountName: "reader", Audience: testGCPAudience}},
		}, wantError: "failed to get pod identity service account"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			SetConfig(&tt.config)
			tokenClient := kubernetesfake.NewClientset()
			provider, err := ResolveServiceAccountTokenProvider(t.Context(), kubeClient, nil, kedav1alpha1.AuthPodIdentity{
				Provider: kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount, ServiceAccountName: new("reader"),
			}, tt.namespace, &authentication.AuthClientSet{CoreV1Interface: tokenClient.CoreV1()})
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				require.NotNil(t, provider)
				assert.Equal(t, testGCPAudience, provider.Audience)
			}
			assert.Empty(t, tokenClient.Actions(), "resolution must not mint or expose an assertion")
		})
	}
}

func TestPodIdentityServiceAccountRefresh(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	cfg := Config{CloudWorkloadIdentityAudiences: []CloudWorkloadIdentityAudience{
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
		require.NotNil(t, request.Spec.ExpirationSeconds)
		assert.Equal(t, cloudWorkloadIdentityTokenExpirySeconds, *request.Spec.ExpirationSeconds)
		return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: want}}, tokenError
	})
	ctx, cancel := context.WithCancel(t.Context())
	provider, err := ResolveServiceAccountTokenProvider(ctx, kubeClient, nil, kedav1alpha1.AuthPodIdentity{
		Provider: kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount, ServiceAccountName: new("reader"),
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
}

func TestPodIdentityServiceAccountGKEIdentityNamespace(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	SetConfig(&Config{CloudWorkloadIdentityAudiences: []CloudWorkloadIdentityAudience{
		{Namespace: "tenant", ServiceAccountName: "reader", Audience: testGKEAudience},
	}})
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Namespace: "tenant", Name: "reader"},
	}).Build()
	tokenClient := kubernetesfake.NewClientset()
	want := testServiceAccountJWT(t, testGKETokenAudience)
	tokenClient.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
		request := action.(k8stesting.CreateAction).GetObject().(*authenticationv1.TokenRequest)
		assert.Equal(t, []string{testGKETokenAudience}, request.Spec.Audiences,
			"GKE assertions must use the workload pool as audience, like the GKE metadata server")
		return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: want}}, nil
	})
	provider, err := ResolveServiceAccountTokenProvider(t.Context(), kubeClient, nil, kedav1alpha1.AuthPodIdentity{
		Provider: kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount, ServiceAccountName: new("reader"),
	}, "tenant", &authentication.AuthClientSet{CoreV1Interface: tokenClient.CoreV1()})
	require.NoError(t, err)
	assert.Equal(t, testGKEAudience, provider.Audience, "the approved audience is passed to STS")
	token, err := provider.GetToken(t.Context())
	require.NoError(t, err)
	assert.Equal(t, want, token)

	// Assertions carrying the full STS audience instead of the pool are rejected.
	want = testServiceAccountJWT(t, testGKEAudience)
	token, err = provider.GetToken(t.Context())
	require.Error(t, err)
	assert.Empty(t, token)
}

func TestPodIdentityServiceAccountDetectsAuthResourceChanges(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	SetConfig(&Config{CloudWorkloadIdentityAudiences: []CloudWorkloadIdentityAudience{
		{Namespace: "tenant", ServiceAccountName: "reader-a", Audience: testGCPAudience},
		{Namespace: "tenant", ServiceAccountName: "reader-b", Audience: testGCPAudience},
	}})
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kedav1alpha1.AddToScheme(scheme))

	identityA := kedav1alpha1.AuthPodIdentity{
		Provider:           kedav1alpha1.PodIdentityProviderGCPWorkloadServiceAccount,
		ServiceAccountName: new("reader-a"),
	}
	ta := &kedav1alpha1.TriggerAuthentication{
		ObjectMeta: metav1.ObjectMeta{Name: "auth", Namespace: "tenant", UID: "uid-1", Generation: 1},
		Spec:       kedav1alpha1.TriggerAuthenticationSpec{PodIdentity: identityA.DeepCopy()},
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		ta,
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "tenant", Name: "reader-a"}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "tenant", Name: "reader-b"}},
	).Build()

	tokenClient := kubernetesfake.NewClientset()
	want := testServiceAccountJWT(t, testGCPAudience)
	tokenClient.PrependReactor("create", "serviceaccounts", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: want}}, nil
	})
	ref := &kedav1alpha1.AuthenticationRef{Name: "auth"}
	provider, err := ResolveServiceAccountTokenProvider(t.Context(), kubeClient, ref, identityA, "tenant",
		&authentication.AuthClientSet{CoreV1Interface: tokenClient.CoreV1()})
	require.NoError(t, err)

	// Unchanged TA (and status-only updates) still allow minting.
	token, err := provider.GetToken(t.Context())
	require.NoError(t, err)
	assert.Equal(t, want, token)
	assert.Len(t, tokenClient.Actions(), 1)

	// Rotating serviceAccountName blocks minting for reader-a without issuing a TokenRequest.
	ta.Spec.PodIdentity.ServiceAccountName = new("reader-b")
	require.NoError(t, kubeClient.Update(t.Context(), ta))
	token, err = provider.GetToken(t.Context())
	require.ErrorContains(t, err, "changed")
	assert.Empty(t, token)
	assert.Len(t, tokenClient.Actions(), 1, "stale provider must not mint a token after auth rotation")

	// Replacing the TA with a new UID also blocks minting even if the spec matches again.
	require.NoError(t, kubeClient.Delete(t.Context(), ta))
	token, err = provider.GetToken(t.Context())
	require.ErrorContains(t, err, "no longer available")
	assert.Empty(t, token)
	recreated := &kedav1alpha1.TriggerAuthentication{
		ObjectMeta: metav1.ObjectMeta{Name: "auth", Namespace: "tenant", UID: "uid-2", Generation: 1},
		Spec:       kedav1alpha1.TriggerAuthenticationSpec{PodIdentity: identityA.DeepCopy()},
	}
	require.NoError(t, kubeClient.Create(t.Context(), recreated))
	token, err = provider.GetToken(t.Context())
	require.ErrorContains(t, err, "changed")
	assert.Empty(t, token)
	assert.Len(t, tokenClient.Actions(), 1)
}
