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

package azure

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

func TestIsServicePrincipalAuthConfigured(t *testing.T) {
	if IsServicePrincipalAuthConfigured(nil) {
		t.Fatal("expected nil auth params not to configure Azure service principal authentication")
	}
	if IsServicePrincipalAuthConfigured(map[string]string{ServicePrincipalAuthKey: "false"}) {
		t.Fatal("expected false marker not to configure Azure service principal authentication")
	}
	if !IsServicePrincipalAuthConfigured(map[string]string{ServicePrincipalAuthKey: "true"}) {
		t.Fatal("expected true marker to configure Azure service principal authentication")
	}
}

func TestNewServicePrincipalCredentialWithClientSecret(t *testing.T) {
	credential, err := NewServicePrincipalCredential(map[string]string{
		ServicePrincipalTenantIDKey:     "tenant-id",
		ServicePrincipalClientIDKey:     "client-id",
		ServicePrincipalCloudKey:        "AzureUSGovernmentCloud",
		ServicePrincipalClientSecretKey: "client-secret",
	})
	if err != nil {
		t.Fatalf("expected client secret credential creation to succeed: %v", err)
	}
	if _, ok := credential.(*azidentity.ClientSecretCredential); !ok {
		t.Fatalf("expected ClientSecretCredential, got %T", credential)
	}
}

func TestNewServicePrincipalCredentialWithClientCertificate(t *testing.T) {
	certificate := createTestClientCertificate(t)
	credential, err := NewServicePrincipalCredential(map[string]string{
		ServicePrincipalTenantIDKey:                "tenant-id",
		ServicePrincipalClientIDKey:                "client-id",
		ServicePrincipalCloudKey:                   PrivateCloud,
		ServicePrincipalActiveDirectoryEndpointKey: "https://login.private.example",
		ServicePrincipalClientCertificateKey:       certificate,
	})
	if err != nil {
		t.Fatalf("expected client certificate credential creation to succeed: %v", err)
	}
	if _, ok := credential.(*azidentity.ClientCertificateCredential); !ok {
		t.Fatalf("expected ClientCertificateCredential, got %T", credential)
	}
}

func TestGetServicePrincipalCredentialOptions(t *testing.T) {
	tests := []struct {
		name                     string
		authParams               map[string]string
		expectedAuthorityHost    string
		disableInstanceDiscovery bool
		wantError                bool
	}{
		{
			name:       "default cloud",
			authParams: map[string]string{},
		},
		{
			name: "Azure Government",
			authParams: map[string]string{
				ServicePrincipalCloudKey: "AzureUSGovernmentCloud",
			},
			expectedAuthorityHost: USGovernmentCloud.ActiveDirectoryEndpoint,
		},
		{
			name: "Azure China",
			authParams: map[string]string{
				ServicePrincipalCloudKey: "AzureChinaCloud",
			},
			expectedAuthorityHost: ChinaCloud.ActiveDirectoryEndpoint,
		},
		{
			name: "private cloud",
			authParams: map[string]string{
				ServicePrincipalCloudKey:                   PrivateCloud,
				ServicePrincipalActiveDirectoryEndpointKey: "https://login.private.example",
			},
			expectedAuthorityHost:    "https://login.private.example",
			disableInstanceDiscovery: true,
		},
		{
			name: "private cloud without authority host",
			authParams: map[string]string{
				ServicePrincipalCloudKey: PrivateCloud,
			},
			wantError: true,
		},
		{
			name: "invalid cloud",
			authParams: map[string]string{
				ServicePrincipalCloudKey: "invalid",
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options, disableInstanceDiscovery, err := getServicePrincipalCredentialOptions(test.authParams)
			if test.wantError {
				if err == nil {
					t.Fatal("expected cloud configuration to fail")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected cloud configuration to succeed: %v", err)
			}
			if options.Cloud.ActiveDirectoryAuthorityHost != test.expectedAuthorityHost {
				t.Fatalf("expected authority host %q, got %q", test.expectedAuthorityHost, options.Cloud.ActiveDirectoryAuthorityHost)
			}
			if disableInstanceDiscovery != test.disableInstanceDiscovery {
				t.Fatalf("expected DisableInstanceDiscovery=%t, got %t", test.disableInstanceDiscovery, disableInstanceDiscovery)
			}
		})
	}
}

func TestNewServicePrincipalCredentialValidation(t *testing.T) {
	tests := []struct {
		name       string
		authParams map[string]string
	}{
		{
			name: "missing tenant ID",
			authParams: map[string]string{
				ServicePrincipalClientIDKey:     "client-id",
				ServicePrincipalClientSecretKey: "client-secret",
			},
		},
		{
			name: "missing client ID",
			authParams: map[string]string{
				ServicePrincipalTenantIDKey:     "tenant-id",
				ServicePrincipalClientSecretKey: "client-secret",
			},
		},
		{
			name: "missing credential",
			authParams: map[string]string{
				ServicePrincipalTenantIDKey: "tenant-id",
				ServicePrincipalClientIDKey: "client-id",
			},
		},
		{
			name: "secret and certificate configured",
			authParams: map[string]string{
				ServicePrincipalTenantIDKey:          "tenant-id",
				ServicePrincipalClientIDKey:          "client-id",
				ServicePrincipalClientSecretKey:      "client-secret",
				ServicePrincipalClientCertificateKey: "client-certificate",
			},
		},
		{
			name: "certificate password without certificate",
			authParams: map[string]string{
				ServicePrincipalTenantIDKey:                  "tenant-id",
				ServicePrincipalClientIDKey:                  "client-id",
				ServicePrincipalClientCertificatePasswordKey: "password",
			},
		},
		{
			name: "invalid certificate",
			authParams: map[string]string{
				ServicePrincipalTenantIDKey:          "tenant-id",
				ServicePrincipalClientIDKey:          "client-id",
				ServicePrincipalClientCertificateKey: "not-a-certificate",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewServicePrincipalCredential(test.authParams); err == nil {
				t.Fatal("expected credential creation to fail")
			}
		})
	}
}

func createTestClientCertificate(t *testing.T) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "keda-test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER})

	return string(append(certificatePEM, privateKeyPEM...))
}
