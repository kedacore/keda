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
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestValidateSpecHashiCorpVaultTokenWarnings(t *testing.T) {
	const (
		deprecated = "hashiCorpVault.credential.token is deprecated, use hashiCorpVault.credential.tokenFrom to read the token from a secret"
		precedence = "hashiCorpVault.credential.tokenFrom takes precedence over hashiCorpVault.credential.token"
	)
	tokenFrom := &ValueFromSecret{SecretKeyRef: SecretKeyRef{Name: "vault-token", Key: "token"}}

	tests := []struct {
		name string
		spec TriggerAuthenticationSpec
		want admission.Warnings
	}{
		{
			name: "no vault",
		},
		{
			name: "vault without credential",
			spec: TriggerAuthenticationSpec{
				HashiCorpVault: &HashiCorpVault{Authentication: VaultAuthenticationKubernetes},
			},
		},
		{
			name: "token from secret",
			spec: TriggerAuthenticationSpec{
				HashiCorpVault: &HashiCorpVault{
					Authentication: VaultAuthenticationToken,
					Credential:     &Credential{TokenFrom: tokenFrom},
				},
			},
		},
		{
			name: "inline token",
			spec: TriggerAuthenticationSpec{
				HashiCorpVault: &HashiCorpVault{
					Authentication: VaultAuthenticationToken,
					Credential:     &Credential{Token: "inline-token"},
				},
			},
			want: admission.Warnings{deprecated},
		},
		{
			name: "inline token and token from secret",
			spec: TriggerAuthenticationSpec{
				HashiCorpVault: &HashiCorpVault{
					Authentication: VaultAuthenticationToken,
					Credential:     &Credential{Token: "inline-token", TokenFrom: tokenFrom},
				},
			},
			want: admission.Warnings{deprecated, precedence},
		},
		{
			// the pod identity switch returns early for providers it does not validate
			name: "inline token with a pod identity provider",
			spec: TriggerAuthenticationSpec{
				PodIdentity: &AuthPodIdentity{Provider: PodIdentityProviderNone},
				HashiCorpVault: &HashiCorpVault{
					Authentication: VaultAuthenticationToken,
					Credential:     &Credential{Token: "inline-token"},
				},
			},
			want: admission.Warnings{deprecated},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			warnings, err := validateSpec(&test.spec)
			assert.NoError(t, err)
			assert.Equal(t, test.want, warnings)
		})
	}
}
