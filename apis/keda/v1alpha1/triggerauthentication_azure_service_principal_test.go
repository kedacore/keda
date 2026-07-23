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

import "testing"

func TestValidateAzureServicePrincipal(t *testing.T) {
	credential := func() *AzureServicePrincipalCredential {
		return &AzureServicePrincipalCredential{
			ValueFrom: ValueFromSecret{
				SecretKeyRef: SecretKeyRef{Name: "azure-credentials", Key: "credential"},
			},
		}
	}

	tests := []struct {
		name             string
		servicePrincipal *AzureServicePrincipal
		wantError        bool
	}{
		{
			name: "client secret",
			servicePrincipal: &AzureServicePrincipal{
				TenantID:     "tenant-id",
				ClientID:     "client-id",
				ClientSecret: credential(),
			},
		},
		{
			name: "client certificate with password",
			servicePrincipal: &AzureServicePrincipal{
				TenantID:                  "tenant-id",
				ClientID:                  "client-id",
				ClientCertificate:         credential(),
				ClientCertificatePassword: credential(),
			},
		},
		{
			name: "private cloud",
			servicePrincipal: &AzureServicePrincipal{
				TenantID:                "tenant-id",
				ClientID:                "client-id",
				Cloud:                   "Private",
				ActiveDirectoryEndpoint: "https://login.private.example",
				ClientSecret:            credential(),
			},
		},
		{
			name: "missing tenant ID",
			servicePrincipal: &AzureServicePrincipal{
				ClientID:     "client-id",
				ClientSecret: credential(),
			},
			wantError: true,
		},
		{
			name: "missing client ID",
			servicePrincipal: &AzureServicePrincipal{
				TenantID:     "tenant-id",
				ClientSecret: credential(),
			},
			wantError: true,
		},
		{
			name: "missing credential",
			servicePrincipal: &AzureServicePrincipal{
				TenantID: "tenant-id",
				ClientID: "client-id",
			},
			wantError: true,
		},
		{
			name: "client secret and certificate",
			servicePrincipal: &AzureServicePrincipal{
				TenantID:          "tenant-id",
				ClientID:          "client-id",
				ClientSecret:      credential(),
				ClientCertificate: credential(),
			},
			wantError: true,
		},
		{
			name: "certificate password without certificate",
			servicePrincipal: &AzureServicePrincipal{
				TenantID:                  "tenant-id",
				ClientID:                  "client-id",
				ClientCertificatePassword: credential(),
			},
			wantError: true,
		},
		{
			name: "private cloud without active directory endpoint",
			servicePrincipal: &AzureServicePrincipal{
				TenantID:     "tenant-id",
				ClientID:     "client-id",
				Cloud:        "Private",
				ClientSecret: credential(),
			},
			wantError: true,
		},
		{
			name: "active directory endpoint without private cloud",
			servicePrincipal: &AzureServicePrincipal{
				TenantID:                "tenant-id",
				ClientID:                "client-id",
				ActiveDirectoryEndpoint: "https://login.private.example",
				ClientSecret:            credential(),
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateSpec(&TriggerAuthenticationSpec{AzureServicePrincipal: test.servicePrincipal})
			if test.wantError && err == nil {
				t.Fatal("expected validation to fail")
			}
			if !test.wantError && err != nil {
				t.Fatalf("expected validation to succeed: %v", err)
			}
		})
	}
}
