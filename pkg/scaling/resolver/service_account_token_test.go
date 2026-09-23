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
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
)

func TestServiceAccountTokenPolicyConfiguration(t *testing.T) {
	for _, value := range []string{
		`[{"audience":"vault"},{"serviceAccountName":"reader","namespace":"tenant","audience":"metrics"}]`,
		"- audience: vault\n- serviceAccountName: reader\n  namespace: tenant\n  audience: metrics\n",
	} {
		cfg := Config{}
		require.NoError(t, cfg.LoadServiceAccountTokenAudiences(value))
		assert.Equal(t, []string{"vault", "metrics"}, cfg.serviceAccountTokenAllowedAudiences())
		audiences, err := cfg.serviceAccountTokenAudiences("tenant", "reader")
		require.NoError(t, err)
		assert.Equal(t, []string{"metrics"}, audiences, "do not mint the global audience union")
	}
	for _, value := range []string{
		"null", "{}", "[", "[]\n---\n[]", "- serviceAccountName: reader",
		`[{"name":"reader","namespace":"tenant","audience":"metrics"}]`, // Old schema is not silently treated as approval only.
		`[{"serviceAccountName":"reader","namespace":"tenant","audiences":["metrics"]}]`,
		`[{"audience":""}]`, `[{"audience":" metrics"}]`,
		`[{"audience":"metrics","mint":true}]`,
		`[{"serviceAccountName":"reader","audience":"metrics"}]`,
		`[{"namespace":"tenant","audience":"metrics"}]`,
		`[{"serviceAccountName":"*","namespace":"tenant","audience":"metrics"}]`,
		`[{"serviceAccountName":"reader","namespace":"tenant/other","audience":"metrics"}]`,
		"- audience: metrics\n  audience: vault\n",
		`[{"serviceAccountName":"reader","namespace":"tenant","audience":"metrics"},{"serviceAccountName":"reader","namespace":"tenant","audience":"vault"}]`,
		`[{"serviceAccountName":"reader","namespace":"tenant","audience":"metrics"},{"serviceAccountName":"reader","namespace":"tenant","audience":"metrics"}]`,
	} {
		t.Run(value, func(t *testing.T) {
			cfg := Config{}
			assert.Error(t, cfg.LoadServiceAccountTokenAudiences(value))
		})
	}
	for _, cfg := range []Config{
		{ServiceAccountTokenMode: "unknown"},
		{ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{Audience: ""}}},
		{ServiceAccountTokenMode: "legacy", ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{Audience: "vault"}}},
		{ServiceAccountTokenMode: "legacy", ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{ServiceAccountName: "reader", Namespace: "tenant", Audience: "metrics"}}},
	} {
		assert.Error(t, cfg.Validate())
	}
	// API audiences may be explicitly allowed; KEDA does not guess them.
	assert.NoError(t, (&Config{ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{Audience: "https://kubernetes.default.svc"}}}).Validate())
	for _, value := range []string{"", "[]"} {
		for _, mode := range []string{"enforce-audience", "legacy"} {
			cfg := Config{ServiceAccountTokenMode: mode}
			require.NoError(t, cfg.LoadServiceAccountTokenAudiences(value))
			require.NoError(t, cfg.Validate(), "unrelated auth does not require token audiences")
		}
	}
}

func TestServiceAccountTokenSharedAudience(t *testing.T) {
	cfg := Config{}
	require.NoError(t, cfg.LoadServiceAccountTokenAudiences(`
- audience: metrics
- audience: metrics
  namespace: tenant
  serviceAccountName: reader
- audience: metrics
  namespace: other
  serviceAccountName: reader
`))
	for _, namespace := range []string{"tenant", "other"} {
		audiences, err := cfg.serviceAccountTokenAudiences(namespace, "reader")
		require.NoError(t, err)
		assert.Equal(t, []string{"metrics"}, audiences)
	}
	_, err := cfg.serviceAccountTokenAudiences("", "")
	assert.Error(t, err, "approval-only entries never select a minting audience")
}

func TestServiceAccountTokenMintingRequiresExactMapping(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	client := kubernetesfake.NewClientset()
	acs := &authentication.AuthClientSet{CoreV1Interface: client.CoreV1()}
	for _, cfg := range []Config{
		{},
		{ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{Audience: "metrics"}}},
		{ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{ServiceAccountName: "reader", Namespace: "other", Audience: "metrics"}}},
		{ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{{ServiceAccountName: "other", Namespace: "tenant", Audience: "metrics"}}},
	} {
		SetConfig(&cfg)
		token, err := GenerateBoundServiceAccountToken(t.Context(), "reader", "tenant", acs)
		assert.ErrorContains(t, err, "no configured token audience")
		assert.Empty(t, token)
	}
	assert.Empty(t, client.Actions(), "must not request a default API token before rejecting configuration")
}

func TestServiceAccountTokenMintingValidatesResponse(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	SetConfig(&Config{
		ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{
			{Audience: "vault"},
			{ServiceAccountName: "reader", Namespace: "tenant", Audience: "metrics"},
		},
	})
	for _, tt := range []struct {
		name      string
		token     string
		apiError  error
		wantError bool
	}{
		{name: "requested audience", token: testServiceAccountJWT(t, "metrics")},
		{name: "API audience", token: testServiceAccountJWT(t, "kube-apiserver"), wantError: true},
		{name: "mixed audiences", token: testServiceAccountJWT(t, "metrics", "kube-apiserver"), wantError: true},
		{name: "approved but not requested", token: testServiceAccountJWT(t, "vault"), wantError: true},
		{name: "no audience", token: testServiceAccountJWT(t), wantError: true},
		{name: "empty response", wantError: true},
		{name: "API failure", apiError: errors.New("token request forbidden"), wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := kubernetesfake.NewClientset()
			client.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
				assert.Equal(t, "tenant", action.GetNamespace())
				assert.Equal(t, "token", action.GetSubresource())
				request := action.(k8stesting.CreateAction).GetObject().(*authenticationv1.TokenRequest)
				assert.Equal(t, []string{"metrics"}, request.Spec.Audiences)
				assert.Positive(t, *request.Spec.ExpirationSeconds)
				return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: tt.token}}, tt.apiError
			})
			token, err := GenerateBoundServiceAccountToken(t.Context(), "reader", "tenant", &authentication.AuthClientSet{CoreV1Interface: client.CoreV1()})
			if tt.wantError {
				require.Error(t, err)
				assert.Empty(t, token)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.token, token)
			}
		})
	}
}

func TestServiceAccountTokenLegacyMinting(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	SetConfig(&Config{ServiceAccountTokenMode: "legacy"})
	want := testServiceAccountJWT(t, "kube-apiserver")
	client := kubernetesfake.NewClientset()
	client.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
		request := action.(k8stesting.CreateAction).GetObject().(*authenticationv1.TokenRequest)
		assert.Empty(t, request.Spec.Audiences)
		return true, &authenticationv1.TokenRequest{Status: authenticationv1.TokenRequestStatus{Token: want}}, nil
	})
	token, err := GenerateBoundServiceAccountToken(t.Context(), "reader", "tenant", &authentication.AuthClientSet{CoreV1Interface: client.CoreV1()})
	require.NoError(t, err)
	assert.Equal(t, want, token)
}

func TestVaultFileTokenAudiencePolicy(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	path := filepath.Join(t.TempDir(), "token")
	SetConfig(&Config{
		VaultKubernetesAuthTokenFile: path,
		ServiceAccountTokenAudiences: []ServiceAccountTokenAudience{
			{Audience: "vault"},
			{ServiceAccountName: "reader", Namespace: "tenant", Audience: "metrics"},
		},
	})
	for _, credential := range []*kedav1alpha1.Credential{nil, {ServiceAccount: path}} {
		vh := NewHashicorpVaultHandler(&kedav1alpha1.HashiCorpVault{Credential: credential}, nil, "tenant")
		for _, tt := range []struct {
			audiences []string
			wantError bool
		}{
			{audiences: []string{"vault"}},
			{audiences: []string{"metrics"}}, // Configured minting audiences also allow file tokens.
			{audiences: []string{"vault", "metrics"}},
			{audiences: []string{"kube-apiserver"}, wantError: true},
			{audiences: []string{"vault", "kube-apiserver"}, wantError: true},
			{wantError: true},
		} {
			// Replacing the contents also checks that each login observes token rotation.
			want := testServiceAccountJWT(t, tt.audiences...)
			require.NoError(t, os.WriteFile(path, []byte(want), 0o600))
			token, err := vh.kubernetesToken(t.Context())
			if tt.wantError {
				require.Error(t, err)
				assert.Empty(t, token)
			} else {
				require.NoError(t, err)
				assert.Equal(t, want, string(token))
			}
		}
	}
	SetConfig(&Config{ServiceAccountTokenMode: "legacy"})
	want := testServiceAccountJWT(t, "kube-apiserver")
	require.NoError(t, os.WriteFile(path, []byte(want), 0o600))
	vh := NewHashicorpVaultHandler(&kedav1alpha1.HashiCorpVault{Credential: &kedav1alpha1.Credential{ServiceAccount: path}}, nil, "tenant")
	token, err := vh.kubernetesToken(t.Context())
	require.NoError(t, err)
	assert.Equal(t, want, string(token))
}

func testServiceAccountJWT(t *testing.T, audiences ...string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "system:serviceaccount:tenant:reader", "aud": audiences,
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("test-only"))
	require.NoError(t, err)
	return token
}
