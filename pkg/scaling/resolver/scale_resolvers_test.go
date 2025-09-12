/*
Copyright 2021 The KEDA Authors

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
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/mock/gomock"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	mock_v1 "github.com/kedacore/keda/v2/pkg/mock/mock_secretlister"
	mock_serviceaccounts "github.com/kedacore/keda/v2/pkg/mock/mock_serviceaccounts"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
)

var (
	namespace                 = "test-namespace"
	clusterNamespace          = "keda"
	triggerAuthenticationName = "triggerauth"
	secretName                = "supersecret"
	secretKey                 = "mysecretkey"
	secretData                = "secretDataHere"
	cmName                    = "supercm"
	cmKey                     = "mycmkey"
	cmData                    = "cmDataHere"
	bsatSAName                = "bsatServiceAccount"
	bsatData                  = "k8s-bsat-token"
	trueValue                 = true
	falseValue                = false
	envKey                    = "test-env-key"
	envValue                  = "test-env-value"
	dependentEnvKey           = "dependent-env-key"
	dependentEnvValue         = "$(test-env-key)-dependent-env-value"
	dependentEnvKey2          = "dependent-env-key2"
	dependentEnvValue2        = "dependent-env-value2-$(test-env-key)"
	escapedEnvKey             = "escaped-env-key"
	escapedEnvValue           = "$$(test-env-key)-escaped-env-value"
	emptyEnvKey               = "empty-env-key"
	emptyEnvValue             = "$()-empty-env-value"
	incompleteEnvKey          = "incomplete-env-key"
	incompleteValue           = "$(test-env-key-incomplete-env-value"
)

type testMetadata struct {
	isError   bool
	comment   string
	container *corev1.Container
}

var testMetadatas = []testMetadata{
	{
		isError: true,
		comment: "configmap does not exist, and it is not marked as an optional, there should be an error",
		container: &corev1.Container{
			EnvFrom: []corev1.EnvFromSource{{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "do-not-exist-not-optional",
					},
				},
			}},
		},
	},
	{
		isError: true,
		comment: "secret does not exist, and it is not marked as an optional, there should be an error",
		container: &corev1.Container{
			EnvFrom: []corev1.EnvFromSource{{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "do-not-exist-not-optional",
					},
				},
			}},
		},
	},
	{
		isError: false,
		comment: "configmap does not exist, but it is marked as an optional, there should not be an error",
		container: &corev1.Container{
			EnvFrom: []corev1.EnvFromSource{{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "do-not-exist-but-optional",
					},
					Optional: &trueValue,
				},
			}},
		},
	},
	{
		isError: false,
		comment: "secret does not exist, but it is marked as an optional, there should not be an error",
		container: &corev1.Container{
			EnvFrom: []corev1.EnvFromSource{{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "do-not-exist-but-optional",
					},
					Optional: &trueValue,
				},
			}},
		},
	},
	{
		isError: true,
		comment: "configmap does not exist, and it is not marked as an optional, there should be an error",
		container: &corev1.Container{
			EnvFrom: []corev1.EnvFromSource{{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "do-not-exist-and-not-optional-explicitly",
					},
					Optional: &falseValue,
				},
			}},
		},
	},
	{
		isError: true,
		comment: "secret does not exist, and it is not marked as an optional, there should be an error",
		container: &corev1.Container{
			EnvFrom: []corev1.EnvFromSource{{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "do-not-exist-and-not-optional-explicitly",
					},
					Optional: &falseValue,
				},
			}},
		},
	},
	{
		isError: false,
		comment: "configmap does not exist, but it is marked as an optional, there should not be an error",
		container: &corev1.Container{
			Env: []corev1.EnvVar{{
				Name: "test",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "do-not-exist-and-optional-explicitly",
						},
						Key:      "test",
						Optional: &trueValue,
					},
				},
			}},
		},
	},
	{
		isError: false,
		comment: "secret does not exist, but it is marked as an optional, there should not be an error",
		container: &corev1.Container{
			Env: []corev1.EnvVar{{
				Name: "test",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "do-not-exist-and-optional-explicitly",
						},
						Key:      "test",
						Optional: &trueValue,
					},
				},
			}},
		},
	},
	{
		isError: true,
		comment: "configmap does not exist, and it is not marked as an optional, there should be an error",
		container: &corev1.Container{
			Env: []corev1.EnvVar{{
				Name: "test",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "do-not-exist-and-not-optional",
						},
						Key:      "test",
						Optional: &falseValue,
					},
				},
			}},
		},
	},
	{
		isError: true,
		comment: "secret does not exist, and it is not marked as an optional, there should be an error",
		container: &corev1.Container{
			Env: []corev1.EnvVar{{
				Name: "test",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "do-not-exist-and-not-optional",
						},
						Key:      "test",
						Optional: &falseValue,
					},
				},
			}},
		},
	},
}

func TestResolveNonExistingConfigMapsOrSecretsEnv(t *testing.T) {
	var secretsLister corev1listers.SecretLister
	for _, testData := range testMetadatas {
		ctx := context.Background()
		_, err := resolveEnv(ctx, fake.NewClientBuilder().Build(), logf.Log.WithName("test"), testData.container, namespace, secretsLister)

		if err != nil && !testData.isError {
			t.Errorf("Expected success because %s got error, %s", testData.comment, err)
		}

		if testData.isError && err == nil {
			t.Errorf("Expected error because %s but got success, %#v", testData.comment, testData)
		}
	}
}

func TestResolveAuthRef(t *testing.T) {
	if err := corev1.AddToScheme(scheme.Scheme); err != nil {
		t.Errorf("Expected Error because: %v", err)
	}
	if err := kedav1alpha1.AddToScheme(scheme.Scheme); err != nil {
		t.Errorf("Expected Error because: %v", err)
	}
	tests := []struct {
		name                string
		existing            []runtime.Object
		soar                *kedav1alpha1.AuthenticationRef
		podSpec             *corev1.PodSpec
		expected            map[string]string
		expectedPodIdentity kedav1alpha1.AuthPodIdentity
		isError             bool
		comment             string
	}{
		{
			name:                "foo",
			expected:            make(map[string]string),
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name:                "no triggerauth exists",
			soar:                &kedav1alpha1.AuthenticationRef{Name: "notthere"},
			expected:            make(map[string]string),
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name:                "no triggerauth exists",
			soar:                &kedav1alpha1.AuthenticationRef{Name: "notthere"},
			expected:            make(map[string]string),
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name: "triggerauth exists, podidentity nil",
			existing: []runtime.Object{
				&kedav1alpha1.TriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						SecretTargetRef: []kedav1alpha1.AuthSecretTargetRef{
							{
								Parameter: "host",
								Name:      secretName,
								Key:       secretKey,
							},
						},
					},
				},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
			expected:            map[string]string{"host": ""},
		},
		{
			name: "triggerauth exists and secret",
			existing: []runtime.Object{
				&kedav1alpha1.TriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						PodIdentity: &kedav1alpha1.AuthPodIdentity{
							Provider: kedav1alpha1.PodIdentityProviderNone,
						},
						SecretTargetRef: []kedav1alpha1.AuthSecretTargetRef{
							{
								Parameter: "host",
								Name:      secretName,
								Key:       secretKey,
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      secretName,
					},
					Data: map[string][]byte{secretKey: []byte(secretData)}},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName},
			expected:            map[string]string{"host": secretData},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name: "triggerauth exists but hashicorp vault can't resolve",
			existing: []runtime.Object{
				&kedav1alpha1.TriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						HashiCorpVault: &kedav1alpha1.HashiCorpVault{
							Address:        "invalid-vault-address",
							Authentication: "token",
							Credential: &kedav1alpha1.Credential{
								Token: "my-token",
							},
							Mount: "kubernetes",
							Role:  "my-role",
							Secrets: []kedav1alpha1.VaultSecret{
								{
									Key:       "password",
									Parameter: "password",
									Path:      "secret_v2/data/my-password-path",
								},
								{
									Key:       "username",
									Parameter: "username",
									Path:      "secret_v2/data/my-username-path",
								},
							},
						},
					},
				},
			},
			isError:             true,
			comment:             "\"my-vault-address-doesnt-exist/v1/auth/token/lookup-self\": unsupported protocol scheme \"\"",
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName},
			expected:            map[string]string{},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name: "triggerauth exists and config map",
			existing: []runtime.Object{
				&kedav1alpha1.TriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						PodIdentity: &kedav1alpha1.AuthPodIdentity{
							Provider: kedav1alpha1.PodIdentityProviderNone,
						},
						ConfigMapTargetRef: []kedav1alpha1.AuthConfigMapTargetRef{
							{
								Parameter: "host",
								Name:      cmName,
								Key:       cmKey,
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      cmName,
					},
					Data: map[string]string{cmKey: cmData},
				},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName},
			expected:            map[string]string{"host": cmData},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name: "triggerauth exists secret + config map",
			existing: []runtime.Object{
				&kedav1alpha1.TriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						PodIdentity: &kedav1alpha1.AuthPodIdentity{
							Provider: kedav1alpha1.PodIdentityProviderNone,
						},
						SecretTargetRef: []kedav1alpha1.AuthSecretTargetRef{
							{
								Parameter: "host-secret",
								Name:      secretName,
								Key:       secretKey,
							},
						},
						ConfigMapTargetRef: []kedav1alpha1.AuthConfigMapTargetRef{
							{
								Parameter: "host-configmap",
								Name:      cmName,
								Key:       cmKey,
							},
						},
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      cmName,
					},
					Data: map[string]string{cmKey: cmData},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      secretName,
					},
					Data: map[string][]byte{secretKey: []byte(secretData)},
				},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName},
			expected:            map[string]string{"host-secret": secretData, "host-configmap": cmData},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name: "triggerauth exists bound service account token",
			existing: []runtime.Object{
				&kedav1alpha1.TriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						PodIdentity: &kedav1alpha1.AuthPodIdentity{
							Provider: kedav1alpha1.PodIdentityProviderNone,
						},
						BoundServiceAccountToken: []kedav1alpha1.BoundServiceAccountToken{
							{
								Parameter:          "token",
								ServiceAccountName: bsatSAName,
							},
						},
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      bsatSAName,
					},
				},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName},
			expected:            map[string]string{"token": bsatData},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name: "clustertriggerauth exists, podidentity nil",
			existing: []runtime.Object{
				&kedav1alpha1.ClusterTriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Name: triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						SecretTargetRef: []kedav1alpha1.AuthSecretTargetRef{
							{
								Parameter: "host",
								Name:      secretName,
								Key:       secretKey,
							},
						},
					},
				},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName, Kind: "ClusterTriggerAuthentication"},
			expected:            map[string]string{"host": ""},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name: "clustertriggerauth exists and secret",
			existing: []runtime.Object{
				&kedav1alpha1.ClusterTriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Name: triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						PodIdentity: &kedav1alpha1.AuthPodIdentity{
							Provider: kedav1alpha1.PodIdentityProviderNone,
						},
						SecretTargetRef: []kedav1alpha1.AuthSecretTargetRef{
							{
								Parameter: "host",
								Name:      secretName,
								Key:       secretKey,
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: clusterNamespace,
						Name:      secretName,
					},
					Data: map[string][]byte{secretKey: []byte(secretData)}},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName, Kind: "ClusterTriggerAuthentication"},
			expected:            map[string]string{"host": secretData},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name: "clustertriggerauth exists and secret + config map",
			existing: []runtime.Object{
				&kedav1alpha1.ClusterTriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Name: triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						PodIdentity: &kedav1alpha1.AuthPodIdentity{
							Provider: kedav1alpha1.PodIdentityProviderNone,
						},
						SecretTargetRef: []kedav1alpha1.AuthSecretTargetRef{
							{
								Parameter: "host",
								Name:      secretName,
								Key:       secretKey,
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: clusterNamespace,
						Name:      secretName,
					},
					Data: map[string][]byte{secretKey: []byte(secretData)},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: clusterNamespace,
						Name:      secretName,
					},
					Data: map[string]string{secretKey: secretData},
				},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName, Kind: "ClusterTriggerAuthentication"},
			expected:            map[string]string{"host": secretData},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name: "clustertriggerauth exists and secret in the wrong namespace",
			existing: []runtime.Object{
				&kedav1alpha1.ClusterTriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Name: triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						PodIdentity: &kedav1alpha1.AuthPodIdentity{
							Provider: kedav1alpha1.PodIdentityProviderNone,
						},
						SecretTargetRef: []kedav1alpha1.AuthSecretTargetRef{
							{
								Parameter: "host",
								Name:      secretName,
								Key:       secretKey,
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      secretName,
					},
					Data: map[string][]byte{secretKey: []byte(secretData)}},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName, Kind: "ClusterTriggerAuthentication"},
			expected:            map[string]string{"host": ""},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name: "clustertriggerauth exists and contains podIdentity configuration but no podSpec (target is a CRD)",
			existing: []runtime.Object{
				&kedav1alpha1.ClusterTriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Name: triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						PodIdentity: &kedav1alpha1.AuthPodIdentity{
							Provider: kedav1alpha1.PodIdentityProviderGCP,
						},
					},
				},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName, Kind: "ClusterTriggerAuthentication"},
			expected:            map[string]string{},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderGCP},
		},
		{
			name: "clustertriggerauth exists and contains podIdentity configuration as well as dummy podSpec",
			existing: []runtime.Object{
				&kedav1alpha1.ClusterTriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Name: triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						PodIdentity: &kedav1alpha1.AuthPodIdentity{
							Provider: kedav1alpha1.PodIdentityProviderGCP,
						},
					},
				},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName, Kind: "ClusterTriggerAuthentication"},
			podSpec:             &corev1.PodSpec{},
			expected:            map[string]string{},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderGCP},
		},
		{
			name: "clustertriggerauth exists bound service account token",
			existing: []runtime.Object{
				&kedav1alpha1.ClusterTriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Name: triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						PodIdentity: &kedav1alpha1.AuthPodIdentity{
							Provider: kedav1alpha1.PodIdentityProviderNone,
						},
						BoundServiceAccountToken: []kedav1alpha1.BoundServiceAccountToken{
							{
								Parameter:          "token",
								ServiceAccountName: bsatSAName,
							},
						},
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: clusterNamespace,
						Name:      bsatSAName,
					},
				},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName, Kind: "ClusterTriggerAuthentication"},
			expected:            map[string]string{"token": bsatData},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
		{
			name: "clustertriggerauth exists bound service account token but service account in the wrong namespace",
			existing: []runtime.Object{
				&kedav1alpha1.ClusterTriggerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Name: triggerAuthenticationName,
					},
					Spec: kedav1alpha1.TriggerAuthenticationSpec{
						PodIdentity: &kedav1alpha1.AuthPodIdentity{
							Provider: kedav1alpha1.PodIdentityProviderNone,
						},
						BoundServiceAccountToken: []kedav1alpha1.BoundServiceAccountToken{
							{
								Parameter:          "token",
								ServiceAccountName: bsatSAName,
							},
						},
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      bsatSAName,
					},
				},
			},
			soar:                &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName, Kind: "ClusterTriggerAuthentication"},
			expected:            map[string]string{"token": ""},
			expectedPodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		},
	}
	ctrl := gomock.NewController(t)
	var secretsLister corev1listers.SecretLister
	mockCoreV1Interface := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	mockServiceAccountInterface := mockCoreV1Interface.GetServiceAccountInterface()
	tokenRequest := &authv1.TokenRequest{
		Status: authv1.TokenRequestStatus{
			Token: bsatData,
		},
	}
	mockServiceAccountInterface.EXPECT().CreateToken(gomock.Any(), gomock.Eq(bsatSAName), gomock.Any(), gomock.Any()).Return(tokenRequest, nil).AnyTimes()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			os.Setenv("KEDA_CLUSTER_OBJECT_NAMESPACE", clusterNamespace) // Inject test cluster namespace.
			gotMap, gotPodIdentity, err := resolveAuthRef(
				ctx,
				fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(test.existing...).Build(),
				logf.Log.WithName("test"),
				test.soar,
				test.podSpec,
				namespace,
				&authentication.AuthClientSet{
					SecretLister:    secretsLister,
					CoreV1Interface: mockCoreV1Interface,
				},
			)

			if err != nil && !test.isError {
				t.Errorf("Expected success because %s got error, %s", test.comment, err)
			}

			if test.isError && err == nil {
				t.Errorf("Expected error because %s but got success, %#v", test.comment, test)
			}

			if diff := cmp.Diff(gotMap, test.expected); diff != "" {
				t.Errorf("Returned authParams are different: %s", diff)
			}
			if gotPodIdentity != test.expectedPodIdentity {
				t.Errorf("Unexpected podidentity, wanted: %q got: %q", test.expectedPodIdentity.Provider, gotPodIdentity.Provider)
			}
		})
	}
}

func TestResolveDependentEnv(t *testing.T) {
	tests := []struct {
		name      string
		expected  map[string]string
		container *corev1.Container
	}{
		{
			name:     "dependent reference env",
			expected: map[string]string{"test-env-key": "test-env-value", "dependent-env-key": "test-env-value-dependent-env-value"},
			container: &corev1.Container{
				Env: []corev1.EnvVar{
					{
						Name:  envKey,
						Value: envValue,
					},
					{
						Name:  dependentEnvKey,
						Value: dependentEnvValue,
					},
				},
			},
		},
		{
			name:     "dependent reference env2",
			expected: map[string]string{"test-env-key": "test-env-value", "dependent-env-key2": "dependent-env-value2-test-env-value"},
			container: &corev1.Container{
				Env: []corev1.EnvVar{
					{
						Name:  envKey,
						Value: envValue,
					},
					{
						Name:  dependentEnvKey2,
						Value: dependentEnvValue2,
					},
				},
			},
		},
		{
			name:     "unchanged reference env",
			expected: map[string]string{"dependent-env-key": "$(test-env-key)-dependent-env-value", "test-env-key": "test-env-value"},
			container: &corev1.Container{
				Env: []corev1.EnvVar{
					{
						Name:  dependentEnvKey,
						Value: dependentEnvValue,
					},
					{
						Name:  envKey,
						Value: envValue,
					},
				},
			},
		},
		{
			name:     "escaped reference env",
			expected: map[string]string{"test-env-key": "test-env-value", "escaped-env-key": "$(test-env-key)-escaped-env-value"},
			container: &corev1.Container{
				Env: []corev1.EnvVar{
					{
						Name:  envKey,
						Value: envValue,
					},
					{
						Name:  escapedEnvKey,
						Value: escapedEnvValue,
					},
				},
			},
		},
		{
			name:     "empty reference env",
			expected: map[string]string{"test-env-key": "test-env-value", "empty-env-key": "$()-empty-env-value"},
			container: &corev1.Container{
				Env: []corev1.EnvVar{
					{
						Name:  envKey,
						Value: envValue,
					},
					{
						Name:  emptyEnvKey,
						Value: emptyEnvValue,
					},
				},
			},
		},
		{
			name:     "incomplete reference env",
			expected: map[string]string{"test-env-key": "test-env-value", "incomplete-env-key": "$(test-env-key-incomplete-env-value"},
			container: &corev1.Container{
				Env: []corev1.EnvVar{
					{
						Name:  envKey,
						Value: envValue,
					},
					{
						Name:  incompleteEnvKey,
						Value: incompleteValue,
					},
				},
			},
		},
	}
	var secretsLister corev1listers.SecretLister
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			envMap, _ := resolveEnv(ctx, fake.NewClientBuilder().Build(), logf.Log.WithName("test"), test.container, namespace, secretsLister)
			if diff := cmp.Diff(envMap, test.expected); diff != "" {
				t.Errorf("Returned authParams are different: %s", diff)
			}
		})
	}
}

func TestEnvWithRestrictSecretAccess(t *testing.T) {
	tests := []struct {
		name      string
		expected  map[string]string
		container *corev1.Container
	}{
		{
			name: "env reference secret key",
			container: &corev1.Container{
				Env: []corev1.EnvVar{{
					Name: envKey,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secretName,
							},
							Key: secretKey,
						},
					},
				}},
			},
			expected: map[string]string{},
		},
		{
			name: "env reference secret name",
			container: &corev1.Container{
				EnvFrom: []corev1.EnvFromSource{{
					SecretRef: &corev1.SecretEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretName,
						},
					},
				}},
			},
			expected: map[string]string{},
		},
	}
	var secretsLister corev1listers.SecretLister
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			restrictSecretAccess = "true"
			ctx := context.Background()
			envMap, _ := resolveEnv(ctx, fake.NewClientBuilder().Build(), logf.Log.WithName("test"), test.container, namespace, secretsLister)
			if diff := cmp.Diff(envMap, test.expected); diff != "" {
				t.Errorf("Returned env map is different: %s", diff)
			}
		})
	}
}

func TestEnvWithRestrictedNamespace(t *testing.T) {
	envPrefix := "PREFIX_"
	tests := []struct {
		name      string
		expected  map[string]string
		container *corev1.Container
	}{
		{
			name: "env reference secret key",
			container: &corev1.Container{
				Env: []corev1.EnvVar{{
					Name: envKey,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secretName,
							},
							Key: secretKey,
						},
					},
				}},
			},
			expected: map[string]string{envKey: secretData},
		},
		{
			name: "env reference secret name",
			container: &corev1.Container{
				EnvFrom: []corev1.EnvFromSource{{
					SecretRef: &corev1.SecretEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretName,
						},
					},
				}},
			},
			expected: map[string]string{secretKey: secretData},
		},
		{
			name: "env reference secret key with prefix",
			container: &corev1.Container{
				EnvFrom: []corev1.EnvFromSource{{
					SecretRef: &corev1.SecretEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretName,
						},
					},
					Prefix: envPrefix,
				}},
			},
			expected: map[string]string{envPrefix + secretKey: secretData},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      secretName,
		},
		Data: map[string][]byte{secretKey: []byte(secretData)},
	}
	ctrl := gomock.NewController(t)
	mockSecretNamespaceLister := mock_v1.NewMockSecretNamespaceLister(ctrl)
	mockSecretNamespaceLister.EXPECT().Get(secretName).Return(secret, nil).AnyTimes()
	mockSecretLister := mock_v1.NewMockSecretLister(ctrl)
	mockSecretLister.EXPECT().Secrets(clusterNamespace).Return(mockSecretNamespaceLister).AnyTimes()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			restrictSecretAccess = "true"
			kedaNamespace = "keda"
			ctx := context.Background()
			envMap, _ := resolveEnv(ctx, fake.NewClientBuilder().Build(), logf.Log.WithName("test"), test.container, clusterNamespace, mockSecretLister)
			if diff := cmp.Diff(envMap, test.expected); diff != "" {
				t.Errorf("Returned env map is different: %s", diff)
			}
		})
	}
}
