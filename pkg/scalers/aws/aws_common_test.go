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

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func TestGetAwsAuthorizationParsesExternalID(t *testing.T) {
	authParams := map[string]string{
		"awsRoleArn":        "arn:aws:iam::123456789012:role/TestRole",
		"awsRoleExternalId": "my-external-id",
	}
	podIdentity := kedav1alpha1.AuthPodIdentity{
		Provider: kedav1alpha1.PodIdentityProviderAws,
	}

	meta, err := GetAwsAuthorization("test-key", "us-east-1", podIdentity, map[string]string{}, authParams, map[string]string{})

	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:iam::123456789012:role/TestRole", meta.AwsRoleArn)
	assert.Equal(t, "my-external-id", meta.AwsRoleExternalID)
}

func TestGetAwsAuthorizationWithEmptyExternalID(t *testing.T) {
	authParams := map[string]string{
		"awsRoleArn": "arn:aws:iam::123456789012:role/TestRole",
	}
	podIdentity := kedav1alpha1.AuthPodIdentity{
		Provider: kedav1alpha1.PodIdentityProviderAws,
	}

	meta, err := GetAwsAuthorization("test-key", "us-east-1", podIdentity, map[string]string{}, authParams, map[string]string{})

	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:iam::123456789012:role/TestRole", meta.AwsRoleArn)
	assert.Empty(t, meta.AwsRoleExternalID)
}

func TestGetAwsAuthorizationWithOnlyExternalID(t *testing.T) {
	authParams := map[string]string{
		"awsRoleExternalId": "my-external-id",
	}
	podIdentity := kedav1alpha1.AuthPodIdentity{
		Provider: kedav1alpha1.PodIdentityProviderAws,
	}

	meta, err := GetAwsAuthorization("test-key", "us-east-1", podIdentity, map[string]string{}, authParams, map[string]string{})

	assert.NoError(t, err)
	assert.Empty(t, meta.AwsRoleArn)
	assert.Equal(t, "my-external-id", meta.AwsRoleExternalID)
}
