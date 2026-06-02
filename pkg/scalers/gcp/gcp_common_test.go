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

package gcp

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestGetGCPAuthorization(t *testing.T) {
	tests := []struct {
		name    string
		config  *scalersconfig.ScalerConfig
		wantErr error
		check   func(t *testing.T, meta *AuthorizationMetadata)
	}{
		{
			name: "PodIdentity GCP enables PodIdentityProviderEnabled",
			config: &scalersconfig.ScalerConfig{
				PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderGCP},
			},
			check: func(t *testing.T, meta *AuthorizationMetadata) {
				assert.True(t, meta.PodIdentityProviderEnabled)
				assert.Empty(t, meta.GoogleApplicationCredentials)
				assert.Empty(t, meta.GoogleApplicationCredentialsFile)
			},
		},
		{
			name: "GoogleApplicationCredentials in AuthParams sets credentials",
			config: &scalersconfig.ScalerConfig{
				PodIdentity: kedav1alpha1.AuthPodIdentity{},
				AuthParams:  map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
			},
			check: func(t *testing.T, meta *AuthorizationMetadata) {
				assert.False(t, meta.PodIdentityProviderEnabled)
				assert.Equal(t, `{"type":"service_account"}`, meta.GoogleApplicationCredentials)
			},
		},
		{
			name: "credentialsFromEnv in TriggerMetadata resolves from ResolvedEnv",
			config: &scalersconfig.ScalerConfig{
				PodIdentity:     kedav1alpha1.AuthPodIdentity{},
				TriggerMetadata: map[string]string{"credentialsFromEnv": "GCP_CREDS"},
				ResolvedEnv:     map[string]string{"GCP_CREDS": `{"type":"service_account","resolved":true}`},
			},
			check: func(t *testing.T, meta *AuthorizationMetadata) {
				assert.Equal(t, `{"type":"service_account","resolved":true}`, meta.GoogleApplicationCredentials)
				assert.Empty(t, meta.GoogleApplicationCredentialsFile)
			},
		},
		{
			name: "credentialsFromEnvFile in TriggerMetadata sets GoogleApplicationCredentialsFile",
			config: &scalersconfig.ScalerConfig{
				PodIdentity:     kedav1alpha1.AuthPodIdentity{},
				TriggerMetadata: map[string]string{"credentialsFromEnvFile": "GCP_CREDS_FILE"},
				ResolvedEnv:     map[string]string{"GCP_CREDS_FILE": "/var/secrets/gcp/key.json"},
			},
			check: func(t *testing.T, meta *AuthorizationMetadata) {
				assert.Equal(t, "/var/secrets/gcp/key.json", meta.GoogleApplicationCredentialsFile)
				assert.Empty(t, meta.GoogleApplicationCredentials)
			},
		},
		{
			name: "none of the above returns ErrGoogleApplicationCrendentialsNotFound",
			config: &scalersconfig.ScalerConfig{
				PodIdentity:     kedav1alpha1.AuthPodIdentity{},
				TriggerMetadata: map[string]string{},
				AuthParams:      map[string]string{},
				ResolvedEnv:     map[string]string{},
			},
			wantErr: ErrGoogleApplicationCrendentialsNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGCPAuthorization(tt.config)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
