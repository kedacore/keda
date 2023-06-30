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

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	mock_v1 "github.com/kedacore/keda/v2/pkg/mock/mock_secretlister"
)

var (
	namespace                 = "test-namespace"
	clusterNamespace          = "keda"
	triggerAuthenticationName = "triggerauth"
	secretName                = "supersecret"
	secretKey                 = "mysecretkey"
	secretData                = "secretDataHere"
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
	}{
		{
			name:     "foo",
			expected: make(map[string]string),
		},
		{
			name:     "no triggerauth exists",
			soar:     &kedav1alpha1.AuthenticationRef{Name: "notthere"},
			expected: make(map[string]string),
		},
		{
			name:     "no triggerauth exists",
			soar:     &kedav1alpha1.AuthenticationRef{Name: "notthere"},
			expected: make(map[string]string),
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
			soar:     &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName},
			expected: map[string]string{"host": ""},
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
			soar:     &kedav1alpha1.AuthenticationRef{Name: triggerAuthenticationName, Kind: "ClusterTriggerAuthentication"},
			expected: map[string]string{"host": ""},
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
	}
	var secretsLister corev1listers.SecretLister
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			os.Setenv("KEDA_CLUSTER_OBJECT_NAMESPACE", clusterNamespace) // Inject test cluster namespace.
			gotMap, gotPodIdentity := resolveAuthRef(
				ctx,
				fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(test.existing...).Build(),
				logf.Log.WithName("test"),
				test.soar,
				test.podSpec,
				namespace,
				secretsLister)
			if diff := cmp.Diff(gotMap, test.expected); diff != "" {
				t.Errorf("Returned authParams are different: %s", diff)
			}
			if gotPodIdentity != test.expectedPodIdentity {
				t.Errorf("Unexpected podidentity, wanted: %q got: %q", test.expectedPodIdentity, gotPodIdentity)
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
		test := test
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
		test := test
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
		test := test
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
