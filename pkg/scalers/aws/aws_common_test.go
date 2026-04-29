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

package aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func TestGetAwsAuthorization_OperatorWithRoleAndExternalID(t *testing.T) {
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
	assert.Equal(t, "test-external-id", meta.AwsExternalID)
	assert.False(t, meta.PodIdentityOwner)
}

func TestGetAwsAuthorization_OperatorWithoutExternalID(t *testing.T) {
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
	assert.Empty(t, meta.AwsExternalID)
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
	assert.Empty(t, meta.AwsExternalID)
	assert.False(t, meta.PodIdentityOwner)
}

func TestGetAwsAuthorization_PodWithRoleAndExternalID(t *testing.T) {
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
	assert.Equal(t, "pod-external-id", meta.AwsExternalID)
	assert.True(t, meta.PodIdentityOwner)
}

func TestGetAwsAuthorization_PodIdentityProviderAwsWithExternalID(t *testing.T) {
	authParams := map[string]string{
		"awsRoleArn":    "arn:aws:iam::123456789012:role/aws-role",
		"awsExternalId": "aws-external-id",
	}

	meta, err := GetAwsAuthorization("key", "us-east-1",
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws},
		nil, authParams, nil)

	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:iam::123456789012:role/aws-role", meta.AwsRoleArn)
	assert.Equal(t, "aws-external-id", meta.AwsExternalID)
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
	assert.Equal(t, "default-external-id", meta.AwsExternalID)
	assert.True(t, meta.PodIdentityOwner)
}

func TestCacheKeyIncludesExternalID(t *testing.T) {
	cache := newSharedConfigsCache()

	meta1 := AuthorizationMetadata{
		AwsRoleArn:    "arn:aws:iam::123456789012:role/role",
		AwsExternalID: "ext-id-1",
		AwsRegion:     "us-east-1",
	}
	meta2 := AuthorizationMetadata{
		AwsRoleArn:    "arn:aws:iam::123456789012:role/role",
		AwsExternalID: "ext-id-2",
		AwsRegion:     "us-east-1",
	}

	key1 := cache.getCacheKey(meta1)
	key2 := cache.getCacheKey(meta2)

	assert.NotEqual(t, key1, key2, "Different ExternalIDs should produce different cache keys")
}

// TestGetAwsConfig_OperatorWithRoleArn verifies that the operator identity path
// (PodIdentityOwner=false) does NOT return early when a role ARN is provided,
// and instead sets up AssumeRole credentials on the config.
func TestGetAwsConfig_OperatorWithRoleArn(t *testing.T) {
	meta := AuthorizationMetadata{
		AwsRegion:        "us-east-1",
		PodIdentityOwner: false,
		AwsRoleArn:       "arn:aws:iam::123456789012:role/test-role",
	}

	cfg, err := GetAwsConfig(context.Background(), meta)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	// Credentials must be set (AssumeRole provider) — not nil as it would be
	// if the early return `if !PodIdentityOwner` fired before the role ARN check.
	assert.NotNil(t, cfg.Credentials)
}

// TestGetAwsConfig_OperatorWithoutRoleArn verifies that the operator identity path
// returns the base config (no AssumeRole) when no role ARN is provided.
func TestGetAwsConfig_OperatorWithoutRoleArn(t *testing.T) {
	metaWithoutRole := AuthorizationMetadata{
		AwsRegion:        "us-east-1",
		PodIdentityOwner: false,
	}
	metaWithRole := AuthorizationMetadata{
		AwsRegion:        "us-east-1",
		PodIdentityOwner: false,
		AwsRoleArn:       "arn:aws:iam::123456789012:role/test-role",
	}

	cfgWithoutRole, err := GetAwsConfig(context.Background(), metaWithoutRole)
	assert.NoError(t, err)

	cfgWithRole, err := GetAwsConfig(context.Background(), metaWithRole)
	assert.NoError(t, err)

	// When no role ARN is set, credentials should be the default chain (not AssumeRole).
	// When role ARN is set, a new credentials provider is set — so they differ.
	assert.NotEqual(t, cfgWithoutRole.Credentials, cfgWithRole.Credentials,
		"operator with role ARN should have different credentials than operator without role ARN")
}