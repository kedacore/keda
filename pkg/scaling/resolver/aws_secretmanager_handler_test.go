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

// import (
// 	"context"
// 	"testing"

// 	"github.com/aws/aws-sdk-go-v2/aws"
// 	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
// 	"github.com/stretchr/testify/assert"
// 	corev1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// 	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
// )

// type MockSecretManagerClient struct {
// 	GetSecretValueFn func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
// }

// func (m *MockSecretManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
// 	return m.GetSecretValueFn(ctx, params, optFns...)
// }

// type MockLogger struct {
// 	ErrorFn func(err error, msg string, keysAndValues ...interface{})
// }

// // Error method for MockLogger
// func (m *MockLogger) Error(err error, msg string, keysAndValues ...interface{}) {
// 	m.ErrorFn(err, msg, keysAndValues...)
// }

// type MockSecretLister struct {
// 	GetFn func(name, namespace string) (*corev1.Secret, error)
// }

// func (m *MockSecretLister) Get(name, namespace string) (*corev1.Secret, error) {
// 	return m.GetFn(name, namespace)
// }

// func TestAwsSecretManagerHandler_Read(t *testing.T) {
// 	expectedSecretValue := "mocked-secret-value"

// 	mockLogger := &MockLogger{
// 		ErrorFn: func(err error, msg string, keysAndValues ...interface{}) {
// 			t.Errorf("Unexpected error in logger: %v", err)
// 		},
// 	}

// 	mockSecretManagerClient := &MockSecretManagerClient{
// 		GetSecretValueFn: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
// 			assert.Equal(t, "mocked-secret-name", aws.ToString(params.SecretId))
// 			return &secretsmanager.GetSecretValueOutput{
// 				SecretString: aws.String(expectedSecretValue),
// 			}, nil
// 		},
// 	}

// 	awsSecretManagerHandler := &AwsSecretManagerHandler{
// 		session: mockSecretManagerClient,
// 	}

// 	secretValue, err := awsSecretManagerHandler.Read(context.Background(), mockLogger, "mocked-secret-name", "", "")

// 	assert.NoError(t, err)
// 	assert.Equal(t, expectedSecretValue, secretValue)
// }

// func TestAwsSecretManagerHandler_Initialize(t *testing.T) {
// 	expectedAwsRegion := "mocked-region"

// 	mockLogger := &MockLogger{
// 		ErrorFn: func(err error, msg string, keysAndValues ...interface{}) {
// 			t.Errorf("Unexpected error in logger: %v", err)
// 		},
// 	}

// 	// Create a mock client
// 	mockClient := &MockSecretLister{
// 		GetFn: func(name, namespace string) (*corev1.Secret, error) {
// 			assert.Equal(t, "mocked-secret-name", name)
// 			assert.Equal(t, "mocked-namespace", namespace)
// 			return &corev1.Secret{
// 				ObjectMeta: metav1.ObjectMeta{Name: "mocked-secret-name", Namespace: "mocked-namespace"},
// 				Data:       map[string][]byte{"key": []byte("mocked-value")},
// 			}, nil
// 		},
// 	}

// 	// Create an AwsSecretManagerHandler with the mock client
// 	awsSecretManagerHandler := &AwsSecretManagerHandler{
// 		secretManager: &kedav1alpha1.AwsSecretManager{
// 			Region: "mocked-region",
// 			Credentials: &kedav1alpha1.AwsSecretManagerCredentials{
// 				AccessKey: &kedav1alpha1.AwsSecretManagerValue{
// 					ValueFrom: kedav1alpha1.ValueFromSecret{
// 						SecretKeyRef: kedav1alpha1.SecretKeyRef{
// 							Name: "mocked-access-key-secret",
// 							Key:  "mocked-access-key-key",
// 						},
// 					},
// 				},
// 				AccessSecretKey: &kedav1alpha1.AwsSecretManagerValue{
// 					ValueFrom: kedav1alpha1.ValueFromSecret{
// 						SecretKeyRef: kedav1alpha1.SecretKeyRef{
// 							Name: "mocked-access-key-secret",
// 							Key:  "mocked-access-key-key",
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	err := awsSecretManagerHandler.Initialize(context.Background(), mockClient, mockLogger, "mocked-trigger-namespace", mockClient, &corev1.PodSpec{})

// 	assert.NoError(t, err)
// 	assert.Equal(t, mocked-secret-name, awsSecretManagerHandler.awsMetadata.AwsRegion)
// }
