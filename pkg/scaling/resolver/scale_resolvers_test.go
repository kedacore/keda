package resolver

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	namespace  string = "test-namespace"
	trueValue  bool   = true
	falseValue bool   = false
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
