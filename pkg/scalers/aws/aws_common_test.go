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

func TestGetAwsAuthorization_OperatorWithRoleAndExternalId(t *testing.T) {
	authParams := map[string]string{
		"awsRoleArn":    "arn:aws:iam::123456789012:role/test-role",
		"awsExternalId": "test-external-id",
	}
	triggerMetadata := map[string]string{
		"identityOwner": "operator",
	}

	meta, err := GetAwsAuthorization("key", "us-east-1",
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		triggerMetadata, authParams, nil)

	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:iam::123456789012:role/test-role", meta.AwsRoleArn)
	assert.Equal(t, "test-external-id", meta.AwsExternalId)
	assert.False(t, meta.PodIdentityOwner)
}

func TestGetAwsAuthorization_OperatorWithoutExternalId(t *testing.T) {
	authParams := map[string]string{
		"awsRoleArn": "arn:aws:iam::123456789012:role/test-role",
	}
	triggerMetadata := map[string]string{
		"identityOwner": "operator",
	}

	meta, err := GetAwsAuthorization("key", "us-east-1",
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		triggerMetadata, authParams, nil)

	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:iam::123456789012:role/test-role", meta.AwsRoleArn)
	assert.Empty(t, meta.AwsExternalId)
	assert.False(t, meta.PodIdentityOwner)
}

func TestGetAwsAuthorization_OperatorWithoutRoleArn(t *testing.T) {
	triggerMetadata := map[string]string{
		"identityOwner": "operator",
	}

	meta, err := GetAwsAuthorization("key", "us-east-1",
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		triggerMetadata, map[string]string{}, nil)

	assert.NoError(t, err)
	assert.Empty(t, meta.AwsRoleArn)
	assert.Empty(t, meta.AwsExternalId)
	assert.False(t, meta.PodIdentityOwner)
}

func TestGetAwsAuthorization_PodWithRoleAndExternalId(t *testing.T) {
	authParams := map[string]string{
		"awsRoleArn":    "arn:aws:iam::123456789012:role/pod-role",
		"awsExternalId": "pod-external-id",
	}
	triggerMetadata := map[string]string{
		"identityOwner": "pod",
	}

	meta, err := GetAwsAuthorization("key", "us-east-1",
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		triggerMetadata, authParams, nil)

	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:iam::123456789012:role/pod-role", meta.AwsRoleArn)
	assert.Equal(t, "pod-external-id", meta.AwsExternalId)
	assert.True(t, meta.PodIdentityOwner)
}

func TestGetAwsAuthorization_PodIdentityProviderAwsWithExternalId(t *testing.T) {
	authParams := map[string]string{
		"awsRoleArn":    "arn:aws:iam::123456789012:role/aws-role",
		"awsExternalId": "aws-external-id",
	}

	meta, err := GetAwsAuthorization("key", "us-east-1",
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws},
		nil, authParams, nil)

	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:iam::123456789012:role/aws-role", meta.AwsRoleArn)
	assert.Equal(t, "aws-external-id", meta.AwsExternalId)
	assert.True(t, meta.UsingPodIdentity)
}

func TestGetAwsAuthorization_DefaultIdentityOwnerIsPod(t *testing.T) {
	authParams := map[string]string{
		"awsRoleArn":    "arn:aws:iam::123456789012:role/default-role",
		"awsExternalId": "default-external-id",
	}

	meta, err := GetAwsAuthorization("key", "us-east-1",
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		map[string]string{}, authParams, nil)

	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:iam::123456789012:role/default-role", meta.AwsRoleArn)
	assert.Equal(t, "default-external-id", meta.AwsExternalId)
	assert.True(t, meta.PodIdentityOwner)
}

func TestCacheKeyIncludesExternalId(t *testing.T) {
	cache := newSharedConfigsCache()

	meta1 := AuthorizationMetadata{
		AwsRoleArn:    "arn:aws:iam::123456789012:role/role",
		AwsExternalId: "ext-id-1",
		AwsRegion:     "us-east-1",
	}
	meta2 := AuthorizationMetadata{
		AwsRoleArn:    "arn:aws:iam::123456789012:role/role",
		AwsExternalId: "ext-id-2",
		AwsRegion:     "us-east-1",
	}

	key1 := cache.getCacheKey(meta1)
	key2 := cache.getCacheKey(meta2)

	assert.NotEqual(t, key1, key2, "Different ExternalIds should produce different cache keys")
}
