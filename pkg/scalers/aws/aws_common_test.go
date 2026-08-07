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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func TestGetAwsAuthorization(t *testing.T) {
	tests := []struct {
		name            string
		podIdentity     kedav1alpha1.AuthPodIdentity
		triggerMetadata map[string]string
		authParams      map[string]string
		resolvedEnv     map[string]string
		wantErr         error
		check           func(t *testing.T, meta AuthorizationMetadata)
	}{
		{
			name:        "PodIdentity AWS provider sets UsingPodIdentity",
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws},
			authParams:  map[string]string{},
			check: func(t *testing.T, meta AuthorizationMetadata) {
				assert.True(t, meta.UsingPodIdentity)
				assert.Empty(t, meta.AwsRoleArn)
			},
		},
		{
			name:        "PodIdentity AWS with awsRoleArn sets both fields",
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws},
			authParams:  map[string]string{"awsRoleArn": "arn:aws:iam::123456789012:role/MyRole"},
			check: func(t *testing.T, meta AuthorizationMetadata) {
				assert.True(t, meta.UsingPodIdentity)
				assert.Equal(t, "arn:aws:iam::123456789012:role/MyRole", meta.AwsRoleArn)
			},
		},
		{
			name:            "identityOwner=operator sets PodIdentityOwner false",
			podIdentity:     kedav1alpha1.AuthPodIdentity{},
			triggerMetadata: map[string]string{"identityOwner": "operator"},
			authParams:      map[string]string{},
			check: func(t *testing.T, meta AuthorizationMetadata) {
				assert.False(t, meta.PodIdentityOwner)
			},
		},
		{
			name:            "identityOwner=pod with awsRoleArn sets AwsRoleArn without access keys",
			podIdentity:     kedav1alpha1.AuthPodIdentity{},
			triggerMetadata: map[string]string{"identityOwner": "pod"},
			authParams:      map[string]string{"awsRoleArn": "arn:aws:iam::123456789012:role/MyRole"},
			check: func(t *testing.T, meta AuthorizationMetadata) {
				assert.True(t, meta.PodIdentityOwner)
				assert.Equal(t, "arn:aws:iam::123456789012:role/MyRole", meta.AwsRoleArn)
				assert.Empty(t, meta.AwsAccessKeyID)
				assert.Empty(t, meta.AwsSecretAccessKey)
			},
		},
		{
			name:            "identityOwner=pod with awsAccessKeyID sets key and secret",
			podIdentity:     kedav1alpha1.AuthPodIdentity{},
			triggerMetadata: map[string]string{"identityOwner": "pod"},
			authParams: map[string]string{
				"awsAccessKeyID":     "AKIAIOSFODNN7EXAMPLE",
				"awsSecretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			check: func(t *testing.T, meta AuthorizationMetadata) {
				assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", meta.AwsAccessKeyID)
				assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", meta.AwsSecretAccessKey)
			},
		},
		{
			name:            "identityOwner=pod with awsAccessKeyID but no secret returns error",
			podIdentity:     kedav1alpha1.AuthPodIdentity{},
			triggerMetadata: map[string]string{"identityOwner": "pod"},
			authParams: map[string]string{
				"awsAccessKeyID": "AKIAIOSFODNN7EXAMPLE",
			},
			wantErr: ErrAwsNoSecretAccessKey,
		},
		{
			name:            "identityOwner=pod awsAccessKeyId lowercase d fallback sets key",
			podIdentity:     kedav1alpha1.AuthPodIdentity{},
			triggerMetadata: map[string]string{"identityOwner": "pod"},
			authParams: map[string]string{
				"awsAccessKeyId":     "AKIAIOSFODNN7EXAMPLE",
				"awsSecretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			check: func(t *testing.T, meta AuthorizationMetadata) {
				assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", meta.AwsAccessKeyID)
			},
		},
		{
			name:            "empty pod identity with triggerMetadata awsAccessKeyID sets key",
			podIdentity:     kedav1alpha1.AuthPodIdentity{},
			triggerMetadata: map[string]string{"awsAccessKeyID": "AKIAIOSFODNN7EXAMPLE", "awsSecretAccessKeyFromEnv": "MY_SECRET_ENV"},
			authParams:      map[string]string{},
			resolvedEnv:     map[string]string{"MY_SECRET_ENV": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"},
			check: func(t *testing.T, meta AuthorizationMetadata) {
				assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", meta.AwsAccessKeyID)
				assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", meta.AwsSecretAccessKey)
			},
		},
		{
			name:        "identityOwner=pod with awsAccessKeyIDFromEnv resolves key from env",
			podIdentity: kedav1alpha1.AuthPodIdentity{},
			triggerMetadata: map[string]string{
				"identityOwner":             "pod",
				"awsAccessKeyIDFromEnv":     "AWS_KEY_ENV",
				"awsSecretAccessKeyFromEnv": "AWS_SECRET_ENV",
			},
			authParams: map[string]string{},
			resolvedEnv: map[string]string{
				"AWS_KEY_ENV":    "AKIAIOSFODNN7EXAMPLE",
				"AWS_SECRET_ENV": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			check: func(t *testing.T, meta AuthorizationMetadata) {
				assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", meta.AwsAccessKeyID)
				assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", meta.AwsSecretAccessKey)
			},
		},
		{
			name:            "missing both keys returns ErrAwsNoAccessKey",
			podIdentity:     kedav1alpha1.AuthPodIdentity{},
			triggerMetadata: map[string]string{},
			authParams:      map[string]string{},
			resolvedEnv:     map[string]string{},
			wantErr:         ErrAwsNoAccessKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.triggerMetadata == nil {
				tt.triggerMetadata = map[string]string{}
			}
			if tt.authParams == nil {
				tt.authParams = map[string]string{}
			}
			if tt.resolvedEnv == nil {
				tt.resolvedEnv = map[string]string{}
			}

			meta, err := GetAwsAuthorization("test-key", "us-east-1", tt.podIdentity, tt.triggerMetadata, tt.authParams, tt.resolvedEnv)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, meta)
			}
		})
	}
}
