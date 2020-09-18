package resolver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	kedav1alpha1 "github.com/kedacore/keda/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	namespace                 = "test-namespace"
	triggerAuthenticationName = "triggerauth"
	secretName                = "supersecret"
	secretKey                 = "mysecretkey"
	secretData                = "secretDataHere"
	trueValue                 = true
	falseValue                = false
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
}

func TestResolveNonExistingConfigMapsOrSecretsEnv(t *testing.T) {

	for _, testData := range testMetadatas {
		_, err := resolveEnv(fake.NewFakeClient(), logf.Log.WithName("test"), testData.container, namespace)

		if err != nil && !testData.isError {
			t.Errorf("Expected success because %s got error, %s", testData.comment, err)
		}

		if testData.isError && err == nil {
			t.Errorf("Expected error because %s but got success, %#v", testData.comment, testData)
		}
	}
}

func TestResolveAuthRef(t *testing.T) {
	corev1.AddToScheme(scheme.Scheme)
	kedav1alpha1.AddToScheme(scheme.Scheme)
	tests := []struct {
		name                string
		existing            []runtime.Object
		soar                *kedav1alpha1.ScaledObjectAuthRef
		podSpec             *corev1.PodSpec
		expected            map[string]string
		expectedPodIdentity string
	}{
		{
			name:     "foo",
			expected: make(map[string]string),
		},
		{
			name:     "no triggerauth exists",
			soar:     &kedav1alpha1.ScaledObjectAuthRef{Name: "notthere"},
			expected: make(map[string]string),
		},
		{
			name:     "no triggerauth exists",
			soar:     &kedav1alpha1.ScaledObjectAuthRef{Name: "notthere"},
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
			soar:     &kedav1alpha1.ScaledObjectAuthRef{Name: triggerAuthenticationName},
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
			soar:                &kedav1alpha1.ScaledObjectAuthRef{Name: triggerAuthenticationName},
			expected:            map[string]string{"host": secretData},
			expectedPodIdentity: "none",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotMap, gotPodIdentity := ResolveAuthRef(fake.NewFakeClientWithScheme(scheme.Scheme, test.existing...), logf.Log.WithName("test"), test.soar, test.podSpec, namespace)
			if diff := cmp.Diff(gotMap, test.expected); diff != "" {
				t.Errorf("Returned authParams are different: %s", diff)
			}
			if gotPodIdentity != test.expectedPodIdentity {
				t.Errorf("Unexpected podidentity, wanted: %q got: %q", test.expectedPodIdentity, gotPodIdentity)
			}
		})
	}
}
