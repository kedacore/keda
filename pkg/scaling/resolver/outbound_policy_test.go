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

package resolver

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func TestOutboundFilter(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		mode      string
		endpoints []string
	}{
		{name: "unset", mode: "off"},
		{name: "whitespace", value: " \n\t", mode: "off"},
		{name: "empty object", value: "{}", mode: "off"},
		{name: "empty vault object", value: "hashiCorpVault: {}", mode: "off"},
		{
			name: "omitted mode", value: `{"hashiCorpVault":{"allowedEndpoints":["https://vault.example"]}}`,
			mode: "off", endpoints: []string{"https://vault.example"},
		},
		{
			name: "JSON warn", value: `{"hashiCorpVault":{"mode":"warn","allowedEndpoints":["https://vault.example"]}}`,
			mode: "warn", endpoints: []string{"https://vault.example"},
		},
		{
			name: "YAML enforce", value: "hashiCorpVault:\n  mode: enforce\n  allowedEndpoints:\n    - https://vault.example:8200\n    - http://vault.internal\n",
			mode: "enforce", endpoints: []string{"https://vault.example:8200", "http://vault.internal"},
		},
		{name: "quoted YAML off", value: "hashiCorpVault:\n  mode: \"off\"", mode: "off"},
		{name: "enforce empty allowlist", value: `{"hashiCorpVault":{"mode":"enforce","allowedEndpoints":[]}}`, mode: "enforce", endpoints: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{}
			require.NoError(t, cfg.LoadOutboundFilter(tt.value))
			assert.Equal(t, tt.mode, cfg.OutboundEndpointPolicy)
			assert.Equal(t, tt.endpoints, cfg.AllowedOutboundEndpoints)
		})
	}
}

func TestOutboundFilterInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "malformed YAML", value: "hashiCorpVault: ["},
		{name: "null configuration", value: "null"},
		{name: "null vault", value: "hashiCorpVault: null"},
		{name: "null mode", value: "hashiCorpVault:\n  mode: null"},
		{name: "empty mode", value: `{"hashiCorpVault":{"mode":""}}`},
		{name: "unknown mode", value: `{"hashiCorpVault":{"mode":"enfroce"}}`},
		{name: "unquoted YAML off", value: "hashiCorpVault:\n  mode: off"},
		{name: "unknown section", value: `{"prometheus":{}}`},
		{name: "old vault section", value: `{"vault":{"mode":"enforce","allowedEndpoints":[]}}`},
		{name: "old and new sections", value: `{"vault":{},"hashiCorpVault":{"mode":"enforce"}}`},
		{name: "unknown field", value: `{"hashiCorpVault":{"allowedEndpoint":[]}}`},
		{name: "duplicate key", value: "hashiCorpVault:\n  mode: warn\n  mode: enforce"},
		{name: "multiple documents", value: "hashiCorpVault: {}\n---\nhashiCorpVault: {}\n"},
		{name: "missing scheme", value: `{"hashiCorpVault":{"allowedEndpoints":["vault.example:8200"]}}`},
		{name: "wildcard", value: `{"hashiCorpVault":{"allowedEndpoints":["https://*.example"]}}`},
		{name: "URL credentials", value: `{"hashiCorpVault":{"allowedEndpoints":["https://user@vault.example"]}}`},
		{name: "URL path", value: `{"hashiCorpVault":{"allowedEndpoints":["https://vault.example/login"]}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{}
			assert.Error(t, cfg.LoadOutboundFilter(tt.value))
		})
	}
}

func TestOutboundFilterEnforcement(t *testing.T) {
	previous := globalConfig
	t.Cleanup(func() { SetConfig(&previous) })
	tests := []struct {
		name      string
		mode      string
		endpoints []string
		wantError bool
	}{
		{name: "zero config allows all"},
		{name: "off ignores allowlist", mode: "off", endpoints: []string{"https://other.example"}},
		{name: "warn allows unlisted endpoint", mode: "warn", endpoints: []string{"https://other.example"}},
		{name: "enforce allows listed endpoint", mode: "enforce", endpoints: []string{"https://vault.example"}},
		{name: "enforce rejects unlisted endpoint", mode: "enforce", endpoints: []string{"https://other.example"}, wantError: true},
		{name: "enforce empty allowlist denies all", mode: "enforce", wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetConfig(&Config{OutboundEndpointPolicy: tt.mode, AllowedOutboundEndpoints: tt.endpoints})
			vh := &HashicorpVaultHandler{vault: &kedav1alpha1.HashiCorpVault{Address: "https://vault.example"}}
			err := vh.validateAddress(logr.Discard())
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
	t.Run("off still requires a Vault audience", func(t *testing.T) {
		SetConfig(&Config{OutboundEndpointPolicy: "off"})
		vh := &HashicorpVaultHandler{vault: &kedav1alpha1.HashiCorpVault{}}
		_, err := vh.kubernetesToken(t.Context())
		assert.ErrorContains(t, err, "no approved service account token audiences")
	})
}
