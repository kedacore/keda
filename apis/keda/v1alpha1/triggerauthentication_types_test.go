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

package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTriggerAuthenticationSpec_WithFilePath(t *testing.T) {
	spec := TriggerAuthenticationSpec{
		FilePath: "/mnt/auth/creds.json",
	}
	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(spec)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"filePath":"/mnt/auth/creds.json"`)

	var unmarshaled TriggerAuthenticationSpec
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, "/mnt/auth/creds.json", unmarshaled.FilePath)
}

func TestTriggerAuthenticationSpec_WithHashiCorpVaultTokenFrom(t *testing.T) {
	spec := TriggerAuthenticationSpec{
		HashiCorpVault: &HashiCorpVault{
			Address:        "http://vault.example.com",
			Authentication: VaultAuthenticationToken,
			Credential: &Credential{
				TokenFrom: &ValueFromSecret{
					SecretKeyRef: SecretKeyRef{
						Name: "vault-token",
						Key:  "token",
					},
				},
			},
			Secrets: []VaultSecret{{
				Parameter: "connection",
				Path:      "secret/data/app",
				Key:       "connectionString",
			}},
		},
	}

	data, err := json.Marshal(spec)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"tokenFrom":{"secretKeyRef":{"name":"vault-token","key":"token"}}`)

	var unmarshaled TriggerAuthenticationSpec
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	if assert.NotNil(t, unmarshaled.HashiCorpVault) && assert.NotNil(t, unmarshaled.HashiCorpVault.Credential) && assert.NotNil(t, unmarshaled.HashiCorpVault.Credential.TokenFrom) {
		assert.Equal(t, "vault-token", unmarshaled.HashiCorpVault.Credential.TokenFrom.SecretKeyRef.Name)
		assert.Equal(t, "token", unmarshaled.HashiCorpVault.Credential.TokenFrom.SecretKeyRef.Key)
	}
}
