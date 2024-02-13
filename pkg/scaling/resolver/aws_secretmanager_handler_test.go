/*
Copyright 2024 The KEDA Authors

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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/util"
)

func TestAwsSecretManagerHandler_InitializeUsingStaticCredentials(t *testing.T) {
	expectedAwsRegion := "mocked-region"
	ctrl := gomock.NewController(util.GinkgoTestReporter{})
	mockClient := mock_client.NewMockClient(ctrl)
	secret := corev1.Secret{
		Data: map[string][]byte{
			"mocked-access-key-key": []byte("secret"),
		},
	}
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, secret).Times(2).Return(nil)

	awsSecretManagerHandler := &AwsSecretManagerHandler{
		secretManager: &kedav1alpha1.AwsSecretManager{
			Region: expectedAwsRegion,
			Credentials: &kedav1alpha1.AwsSecretManagerCredentials{
				AccessKey: &kedav1alpha1.AwsSecretManagerValue{
					ValueFrom: kedav1alpha1.ValueFromSecret{
						SecretKeyRef: kedav1alpha1.SecretKeyRef{
							Name: "mocked-access-key-secret",
							Key:  "mocked-access-key-key",
						},
					},
				},
				AccessSecretKey: &kedav1alpha1.AwsSecretManagerValue{
					ValueFrom: kedav1alpha1.ValueFromSecret{
						SecretKeyRef: kedav1alpha1.SecretKeyRef{
							Name: "mocked-access-key-secret",
							Key:  "mocked-access-key-key",
						},
					},
				},
			},
		},
	}

	err := awsSecretManagerHandler.Initialize(context.Background(), mockClient, logr.Discard(), "mocked-trigger-namespace", nil, &corev1.PodSpec{})

	assert.NoError(t, err)
	assert.Equal(t, expectedAwsRegion, awsSecretManagerHandler.secretManager.Region)
}
func TestAwsSecretManagerHandler_InitializeUsingPodIdentity(t *testing.T) {
	tests := []struct {
		provider kedav1alpha1.PodIdentityProvider
		isError  bool
	}{
		{provider: kedav1alpha1.PodIdentityProviderAws, isError: false},
		{provider: kedav1alpha1.PodIdentityProviderAwsEKS, isError: true},
		{provider: kedav1alpha1.PodIdentityProviderAwsKiam, isError: true},
		{provider: kedav1alpha1.PodIdentityProviderAzure, isError: true},
		{provider: kedav1alpha1.PodIdentityProviderAzureWorkload, isError: true},
		{provider: kedav1alpha1.PodIdentityProviderGCP, isError: true},
	}

	for _, tc := range tests {
		awsSecretManagerHandler := &AwsSecretManagerHandler{
			secretManager: &kedav1alpha1.AwsSecretManager{
				Region: "expectedAwsRegion",
				PodIdentity: &kedav1alpha1.AuthPodIdentity{
					Provider: tc.provider,
				},
			},
		}

		err := awsSecretManagerHandler.Initialize(context.Background(), nil, logr.Discard(), "mocked-trigger-namespace", nil, &corev1.PodSpec{})

		if tc.isError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}
