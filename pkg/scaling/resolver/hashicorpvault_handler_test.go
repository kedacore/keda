/*
Copyright 2023 The KEDA Authors

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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	authv1 "k8s.io/api/authentication/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	mock_secretlister "github.com/kedacore/keda/v2/pkg/mock/mock_secretlister"
	mock_serviceaccounts "github.com/kedacore/keda/v2/pkg/mock/mock_serviceaccounts"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
)

const (
	vaultTestToken  = "TestToK*n"
	kedaSecretValue = "keda"
	pkiCert         = "-----BEGIN CERTIFICATE-----\nMIID\n-----END CERTIFICATE-----"
	pkiCaChain      = "-----BEGIN CERTIFICATE-----\nMIIA\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"
)

var (
	vaultTokenSelf = map[string]interface{}{
		"accessor":         "8609694a-cdbc-db9b-d345-e782dbb562ed",
		"creation_time":    1697036787,
		"creation_ttl":     0,
		"display_name":     "ldap2-tesla",
		"entity_id":        "",
		"expire_time":      nil,
		"explicit_max_ttl": 0,
		"id":               vaultTestToken,
		"issue_time":       "2023-10-11T15:06:27.602936828Z",
		"meta":             nil,
		"num_uses":         0,
		"orphan":           true,
		"path":             "auth/token/create",
		"policies":         []string{"default"},
		"renewable":        false,
		"ttl":              0,
	}
	kvV2SecretDataKeda = map[string]interface{}{
		"data": map[string]interface{}{
			"test":  kedaSecretValue,
			"array": []string{kedaSecretValue},
		},
		"metadata": map[string]interface{}{
			"version": 1,
		},
	}
	kvV1SecretDataKeda = map[string]interface{}{
		"test":  kedaSecretValue,
		"array": []string{kedaSecretValue},
	}
)

type pkiRequestTestData struct {
	name     string
	raw      string
	secret   kedav1alpha1.VaultSecret
	expected map[string]interface{}
}

var pkiRequestTestDataset = []pkiRequestTestData{
	{
		name:   "valid pki request",
		raw:    `{ "commonName": "test" }`,
		secret: kedav1alpha1.VaultSecret{},
		expected: map[string]interface{}{
			"common_name": "test",
		},
	},
	{
		name: "full pki request with all fields",
		secret: kedav1alpha1.VaultSecret{
			PkiData: kedav1alpha1.VaultPkiData{
				CommonName: "test",
				AltNames:   "test2",
				IPSans:     "192.168.1.1",
				URISans:    "test.com",
				OtherSans:  "othersans.com",
				TTL:        "24h",
				Format:     "pem",
			},
		},
		expected: map[string]interface{}{
			"common_name": "test",
			"alt_names":   "test2",
			"ip_sans":     "192.168.1.1",
			"uri_sans":    "test.com",
			"other_sans":  "othersans.com",
			"ttl":         "24h",
			"format":      "pem",
		},
	},
}

func TestGetPkiRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockCoreV1Interface := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	mockSecretLister := mock_secretlister.NewMockSecretLister(ctrl)
	authClientSet := &authentication.AuthClientSet{
		CoreV1Interface: mockCoreV1Interface,
		SecretLister:    mockSecretLister,
	}

	vault := NewHashicorpVaultHandler(nil, authClientSet, "default")

	for _, testData := range pkiRequestTestDataset {
		var secret kedav1alpha1.VaultSecret
		if testData.raw != "" {
			var pkiData kedav1alpha1.VaultPkiData
			_ = json.Unmarshal([]byte(testData.raw), &pkiData)
			secret = kedav1alpha1.VaultSecret{
				PkiData: pkiData,
			}
		} else {
			secret = testData.secret
		}
		data := vault.getPkiRequest(&secret.PkiData)
		assert.Equalf(t, testData.expected, data, "test %s: expected data does not match given secret", testData.name)
	}
}

func mockVault(t *testing.T, useRootToken bool) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]interface{}
		var auth *vaultapi.SecretAuth
		switch r.URL.Path {
		case "/v1/auth/token/lookup-self":
			data = vaultTokenSelf
			if useRootToken {
				// remove the renewable field
				delete(data, "renewable")
			}
		case "/v1/kv_v2/data/keda": //todo: more generic
			data = kvV2SecretDataKeda
		case "/v1/kv/keda": //todo: more generic
			data = kvV1SecretDataKeda
		case "/v1/pki/issue/default":
			bytes, _ := io.ReadAll(r.Body)
			str := base64.RawURLEncoding.EncodeToString(bytes)
			randomCert := fmt.Sprintf("-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----", str)
			randomKey := fmt.Sprintf("-----BEGIN END RSA PRIVATE KEY-----\n%s\n-----END END RSA PRIVATE KEY-----", str)
			data = map[string]interface{}{

				"ca_chain": []interface{}{
					"-----BEGIN CERTIFICATE-----\nMIIA\n-----END CERTIFICATE-----",
					"-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
				},
				"certificate":      randomCert,
				"expiration":       1697313631,
				"issuing_ca":       "-----BEGIN CERTIFICATE-----\nMIIDZ\n-----END CERTIFICATE-----",
				"private_key":      randomKey,
				"private_key_type": "rsa",
				"serial_number":    "4c:79:c6:2c:23:65:77:73:c2:79:49:8c:c8:fe:ad:e3:78:68:0f:86",
			}
		case "/v1/auth/kubernetes/login":
			auth = &vaultapi.SecretAuth{
				ClientToken: vaultTestToken,
			}
		default:
			t.Logf("Got request at path %s", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		secret := vaultapi.Secret{
			RequestID:     "72be5985-c24b-7083-9ca0-5957093f8b04",
			LeaseID:       "",
			LeaseDuration: 0,
			Data:          data,
			Renewable:     false,
			Warnings:      nil,
			Auth:          auth,
			WrapInfo:      nil,
		}
		var out, _ = json.Marshal(secret)
		_, _ = w.Write(out)
	}))
	return server
}

func TestHashicorpVaultHandler_getSecretValue_specify_secret_type(t *testing.T) {
	server := mockVault(t, false)
	defer server.Close()
	ctrl := gomock.NewController(t)
	mockCoreV1Interface := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	mockSecretLister := mock_secretlister.NewMockSecretLister(ctrl)
	authClientSet := &authentication.AuthClientSet{
		CoreV1Interface: mockCoreV1Interface,
		SecretLister:    mockSecretLister,
	}

	vault := kedav1alpha1.HashiCorpVault{
		Address:        server.URL,
		Authentication: kedav1alpha1.VaultAuthenticationToken,
		Credential: &kedav1alpha1.Credential{
			Token: vaultTestToken,
		},
	}
	vaultHandler := NewHashicorpVaultHandler(&vault, authClientSet, "default")
	err := vaultHandler.Initialize(logf.Log.WithName("test"))
	defer vaultHandler.Stop()
	assert.Nil(t, err)
	secrets := []kedav1alpha1.VaultSecret{{
		Parameter: "test",
		Path:      "kv_v2/data/keda",
		Key:       "test",
	}}
	assert.Equalf(t, kedav1alpha1.VaultSecretTypeGeneric, secrets[0].Type, "Expected secret to not have a vlue")
	secrets, _ = vaultHandler.ResolveSecrets(secrets)
	assert.Len(t, secrets, 1, "Supposed to get back one secret")
	secret := secrets[0]
	assert.Equalf(t, kedav1alpha1.VaultSecretTypeSecretV2, secret.Type, "Expexted secret type be %s", kedav1alpha1.VaultSecretTypeSecretV2)
	assert.Equalf(t, kedaSecretValue, secret.Value, "Expexted secret to be %s", kedaSecretValue)
	secrets = []kedav1alpha1.VaultSecret{{
		Parameter: "test",
		Path:      "kv/keda",
		Key:       "test",
	}}
	assert.Equalf(t, kedav1alpha1.VaultSecretTypeGeneric, secrets[0].Type, "Expected secret to not have a vlue")
	secrets, _ = vaultHandler.ResolveSecrets(secrets)
	assert.Len(t, secrets, 1, "Supposed to get back one secret")
	secret = secrets[0]
	assert.Equalf(t, kedav1alpha1.VaultSecretTypeSecret, secret.Type, "Expexted secret type be %s", kedav1alpha1.VaultSecretTypeSecret)
	assert.Equalf(t, kedaSecretValue, secret.Value, "Expexted secret to be %s", kedaSecretValue)
}

type resolveRequestTestData struct {
	name          string
	path          string
	key           string
	secretType    kedav1alpha1.VaultSecretType
	pkiData       kedav1alpha1.VaultPkiData
	isError       bool
	expectedValue string
}

var resolveRequestTestDataSet = []resolveRequestTestData{
	{
		name:          "existing_secret_v2",
		path:          "kv_v2/data/keda",
		key:           "test",
		isError:       false,
		expectedValue: kedaSecretValue,
	},
	{
		name:          "non_existing_secret_v2",
		path:          "kv_v2/data/kedaNotExist",
		key:           "test",
		isError:       false,
		expectedValue: "",
		secretType:    kedav1alpha1.VaultSecretTypeSecretV2,
	},
	{
		name:          "non_existing_key_in_existing_secret_v2",
		path:          "kv_v2/data/keda",
		key:           "testNotExisting",
		isError:       false,
		expectedValue: "",
		secretType:    kedav1alpha1.VaultSecretTypeSecretV2,
	},
	{
		name:          "non_string_in_existing_secret_v2",
		path:          "kv_v2/data/keda",
		key:           "array",
		isError:       false,
		expectedValue: "",
		secretType:    kedav1alpha1.VaultSecretTypeSecretV2,
	},
	{
		name:          "existing_secret_v1",
		path:          "kv/keda",
		key:           "test",
		isError:       false,
		expectedValue: kedaSecretValue,
	},
	{
		name:          "non_existing_secret_v1",
		path:          "kv/kedaNotExist",
		key:           "test",
		isError:       false,
		expectedValue: "",
		secretType:    kedav1alpha1.VaultSecretTypeSecretV2,
	},
	{
		name:          "non_existing_key_in_existing_secret_v1",
		path:          "kv/keda",
		key:           "testNotExisting",
		isError:       false,
		expectedValue: "",
		secretType:    kedav1alpha1.VaultSecretTypeSecret,
	},
	{
		name:          "non_string_in_existing_secret_v1",
		path:          "kv/keda",
		key:           "array",
		isError:       false,
		expectedValue: "",
		secretType:    kedav1alpha1.VaultSecretTypeSecret,
	},
	{
		name:          "incorrect_type",
		path:          "kv/keda",
		key:           "array",
		isError:       false,
		expectedValue: "",
		secretType:    "non_existing_type",
	},
	{
		name:          "existing_pki",
		path:          "pki/issue/default",
		key:           "private_key_type",
		isError:       false,
		secretType:    kedav1alpha1.VaultSecretTypePki,
		pkiData:       kedav1alpha1.VaultPkiData{CommonName: "test"},
		expectedValue: "rsa",
	},
	{
		name:          "existing_pki_ca_chain",
		path:          "pki/issue/default",
		key:           "ca_chain",
		isError:       false,
		secretType:    kedav1alpha1.VaultSecretTypePki,
		pkiData:       kedav1alpha1.VaultPkiData{CommonName: "test"},
		expectedValue: pkiCaChain,
	},
}

func TestHashicorpVaultHandler_ResolveSecret(t *testing.T) {
	server := mockVault(t, false)
	defer server.Close()
	ctrl := gomock.NewController(t)
	mockCoreV1Interface := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	mockSecretLister := mock_secretlister.NewMockSecretLister(ctrl)
	authClientSet := &authentication.AuthClientSet{
		CoreV1Interface: mockCoreV1Interface,
		SecretLister:    mockSecretLister,
	}

	vault := kedav1alpha1.HashiCorpVault{
		Address:        server.URL,
		Authentication: kedav1alpha1.VaultAuthenticationToken,
		Credential: &kedav1alpha1.Credential{
			Token: vaultTestToken,
		},
	}
	vaultHandler := NewHashicorpVaultHandler(&vault, authClientSet, "default")
	err := vaultHandler.Initialize(logf.Log.WithName("test"))
	defer vaultHandler.Stop()
	assert.Nil(t, err)

	for _, testData := range resolveRequestTestDataSet {
		secrets := []kedav1alpha1.VaultSecret{{
			Parameter: "test",
			Path:      testData.path,
			Key:       testData.key,
			Type:      testData.secretType,
			PkiData:   testData.pkiData,
		}}
		secrets, err := vaultHandler.ResolveSecrets(secrets)
		assert.Len(t, secrets, 1, "Supposed to get back one secret")
		secret := secrets[0]
		if testData.isError {
			assert.NotNilf(t, err, "test %s: expected error but got success, testData - %+v", testData.name, testData)
			continue
		}
		assert.Nilf(t, err, "test %s: expected success but got error - %s", testData.name, err)
		assert.Equalf(t, testData.expectedValue, secret.Value, "test %s: expected data does not match given secret", testData.name)
	}
}

func TestHashicorpVaultHandler_ResolveSecret_UsingRootToken(t *testing.T) {
	server := mockVault(t, true)
	defer server.Close()

	vault := kedav1alpha1.HashiCorpVault{
		Address:        server.URL,
		Authentication: kedav1alpha1.VaultAuthenticationToken,
		Credential: &kedav1alpha1.Credential{
			Token: vaultTestToken,
		},
	}
	ctrl := gomock.NewController(t)
	mockCoreV1Interface := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	mockSecretLister := mock_secretlister.NewMockSecretLister(ctrl)
	authClientSet := &authentication.AuthClientSet{
		CoreV1Interface: mockCoreV1Interface,
		SecretLister:    mockSecretLister,
	}

	vaultHandler := NewHashicorpVaultHandler(&vault, authClientSet, "default")
	err := vaultHandler.Initialize(logf.Log.WithName("test"))
	defer vaultHandler.Stop()
	assert.Nil(t, err)

	for _, testData := range resolveRequestTestDataSet {
		secrets := []kedav1alpha1.VaultSecret{{
			Parameter: "test",
			Path:      testData.path,
			Key:       testData.key,
			Type:      testData.secretType,
			PkiData:   testData.pkiData,
		}}
		secrets, err := vaultHandler.ResolveSecrets(secrets)
		assert.Len(t, secrets, 1, "Supposed to get back one secret")
		secret := secrets[0]
		if testData.isError {
			assert.NotNilf(t, err, "test %s: expected error but got success, testData - %+v", testData.name, testData)
			continue
		}
		assert.Nilf(t, err, "test %s: expected success but got error - %s", testData.name, err)
		assert.Equalf(t, testData.expectedValue, secret.Value, "test %s: expected data does not match given secret", testData.name)
	}
}

func TestHashicorpVaultHandler_DefaultKubernetesVaultRole(t *testing.T) {
	defaultServiceAccountPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"
	server := mockVault(t, false)
	defer server.Close()
	ctrl := gomock.NewController(t)
	mockCoreV1Interface := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	mockSecretLister := mock_secretlister.NewMockSecretLister(ctrl)
	authClientSet := &authentication.AuthClientSet{
		CoreV1Interface: mockCoreV1Interface,
		SecretLister:    mockSecretLister,
	}

	vault := kedav1alpha1.HashiCorpVault{
		Address:        server.URL,
		Authentication: kedav1alpha1.VaultAuthenticationKubernetes,
		Mount:          "my-mount",
		Role:           "my-role",
	}

	vaultHandler := NewHashicorpVaultHandler(&vault, authClientSet, "default")
	err := vaultHandler.Initialize(logf.Log.WithName("test"))
	defer vaultHandler.Stop()
	assert.Errorf(t, err, "open %s : no such file or directory", defaultServiceAccountPath)
	assert.Equal(t, vaultHandler.vault.Credential.ServiceAccount, defaultServiceAccountPath)
}

func TestHashicorpVaultHandler_ResolveSecrets_SameCertAndKey(t *testing.T) {
	server := mockVault(t, false)
	defer server.Close()
	ctrl := gomock.NewController(t)
	mockCoreV1Interface := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	mockSecretLister := mock_secretlister.NewMockSecretLister(ctrl)
	authClientSet := &authentication.AuthClientSet{
		CoreV1Interface: mockCoreV1Interface,
		SecretLister:    mockSecretLister,
	}

	vault := kedav1alpha1.HashiCorpVault{
		Address:        server.URL,
		Authentication: kedav1alpha1.VaultAuthenticationToken,
		Credential: &kedav1alpha1.Credential{
			Token: vaultTestToken,
		},
	}
	vaultHandler := NewHashicorpVaultHandler(&vault, authClientSet, "default")
	err := vaultHandler.Initialize(logf.Log.WithName("test"))
	defer vaultHandler.Stop()
	assert.Nil(t, err)
	secrets := []kedav1alpha1.VaultSecret{{
		Parameter: "certificate",
		Path:      "pki/issue/default",
		Key:       "certificate",
		Type:      kedav1alpha1.VaultSecretTypePki,
		PkiData:   kedav1alpha1.VaultPkiData{CommonName: "test"},
	}, {
		Parameter: "certificate",
		Path:      "pki/issue/default",
		Key:       "certificate",
		Type:      kedav1alpha1.VaultSecretTypePki,
		PkiData:   kedav1alpha1.VaultPkiData{CommonName: "test"},
	}}
	secrets, _ = vaultHandler.ResolveSecrets(secrets)
	assert.Len(t, secrets, 2, "Supposed to get back two secrets")
	assert.Equalf(t, secrets[0].Value, secrets[1].Value, "Refetching same path should yield same value")
}

var fetchSecretTestDataSet = []resolveRequestTestData{
	{
		name:          "existing_secret_v2",
		path:          "kv_v2/data/keda",
		key:           "test",
		isError:       false,
		expectedValue: kedaSecretValue,
	},
	{
		name:          "existing_secret_v1",
		path:          "kv/keda",
		key:           "test",
		isError:       false,
		expectedValue: kedaSecretValue,
	},
	{
		name:          "existing_pki",
		path:          "pki/issue/default",
		key:           "private_key_type",
		isError:       false,
		secretType:    kedav1alpha1.VaultSecretTypePki,
		pkiData:       kedav1alpha1.VaultPkiData{CommonName: "test"},
		expectedValue: "rsa",
	},
	{
		name:          "existing_pki_ca_chain",
		path:          "pki/issue/default",
		key:           "ca_chain",
		isError:       false,
		secretType:    kedav1alpha1.VaultSecretTypePki,
		pkiData:       kedav1alpha1.VaultPkiData{CommonName: "test"},
		expectedValue: pkiCaChain,
	},
}

func TestHashicorpVaultHandler_fetchSecret(t *testing.T) {
	server := mockVault(t, false)
	defer server.Close()

	ctrl := gomock.NewController(t)
	mockCoreV1Interface := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	mockSecretLister := mock_secretlister.NewMockSecretLister(ctrl)
	authClientSet := &authentication.AuthClientSet{
		CoreV1Interface: mockCoreV1Interface,
		SecretLister:    mockSecretLister,
	}

	vault := kedav1alpha1.HashiCorpVault{
		Address:        server.URL,
		Authentication: kedav1alpha1.VaultAuthenticationToken,
		Credential: &kedav1alpha1.Credential{
			Token: vaultTestToken,
		},
	}

	vaultHandler := NewHashicorpVaultHandler(&vault, authClientSet, "default")
	err := vaultHandler.Initialize(logf.Log.WithName("test"))
	defer vaultHandler.Stop()
	assert.Nil(t, err)

	for _, testData := range fetchSecretTestDataSet {
		secretResponse, err := vaultHandler.fetchSecret(testData.secretType, testData.path, &testData.pkiData)
		assert.Nil(t, err)

		if testData.isError {
			assert.NotNilf(t, err, "test %s: expected error but got success, testData - %+v", testData.name, testData)
		}
		secretStruct := kedav1alpha1.VaultSecret{Parameter: "test", Path: testData.path, Key: testData.key, Type: testData.secretType, PkiData: testData.pkiData}
		secret, err := vaultHandler.getSecretValue(&secretStruct, secretResponse)

		assert.Nil(t, err)
		assert.Equalf(t, testData.expectedValue, secret, "test %s: expected data does not match given secret", testData.name)
	}
}

type initializeTestData struct {
	name      string
	namespace string
	token     string
	isError   bool
}

var initialiseTestDataSet = []initializeTestData{
	{
		name:      "Namespace and Token",
		namespace: "testNamespace",
		token:     "testToken",
		isError:   false,
	},
	{
		name:      "No Namespace",
		namespace: "",
		token:     "testToken",
		isError:   false,
	},
}

func TestHashicorpVaultHandler_Initialize(t *testing.T) {
	server := mockVault(t, false)
	defer server.Close()

	ctrl := gomock.NewController(t)
	mockCoreV1Interface := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	mockSecretLister := mock_secretlister.NewMockSecretLister(ctrl)
	authClientSet := &authentication.AuthClientSet{
		CoreV1Interface: mockCoreV1Interface,
		SecretLister:    mockSecretLister,
	}

	for _, testData := range initialiseTestDataSet {
		func() {
			vault := kedav1alpha1.HashiCorpVault{
				Address:        server.URL,
				Authentication: kedav1alpha1.VaultAuthenticationToken,
				Credential: &kedav1alpha1.Credential{
					Token: testData.token,
				},
				Namespace: testData.namespace,
			}
			vaultHandler := NewHashicorpVaultHandler(&vault, authClientSet, testData.namespace)
			err := vaultHandler.Initialize(logf.Log.WithName("test"))
			defer vaultHandler.Stop()
			assert.Nil(t, err)

			if testData.isError {
				assert.NotNilf(t, err, "test %s: expected error but got success, testData - %+v", testData.name, testData)
			} else {
				assert.Equalf(t, vaultHandler.client.Address(), server.URL, "test case %s", testData.name)
				assert.Equalf(t, vaultHandler.client.Token(), testData.token, "test case %s", testData.name)
				assert.Equalf(t, vaultHandler.client.Namespace(), testData.namespace, "test case %s", testData.name)
			}
		}()
	}
}

type tokenTestData struct {
	name           string
	isError        bool
	errorMessage   string
	authentication kedav1alpha1.VaultAuthentication
	credential     kedav1alpha1.Credential
	mount          string
	role           string
}

var tokenTestDataSet = []tokenTestData{
	{
		name:           "Vault Authentication",
		isError:        false,
		authentication: kedav1alpha1.VaultAuthenticationToken,
		credential: kedav1alpha1.Credential{
			Token: vaultTestToken,
		},
		role:  "my-role",
		mount: "my-mount",
	},
	{
		name:           "Kubernetes Authentication",
		isError:        true, // Because the service account path is non-existent
		authentication: kedav1alpha1.VaultAuthenticationKubernetes,
		credential: kedav1alpha1.Credential{
			ServiceAccount: "random/path",
		},
		role:         "my-role",
		mount:        "my-mount",
		errorMessage: "open random/path: no such file or directory",
	},
	{
		name:           "Wrong Authentication Method",
		isError:        true,
		authentication: "random",
		credential: kedav1alpha1.Credential{
			ServiceAccount: "random/path",
		},
		role:         "my-role",
		mount:        "my-mount",
		errorMessage: "vault auth method random is not supported",
	},
}

func TestHashicorpVaultHandler_Token_VaultTokenAuth(t *testing.T) {
	server := mockVault(t, false)
	defer server.Close()

	ctrl := gomock.NewController(t)
	mockCoreV1Interface := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	mockSecretLister := mock_secretlister.NewMockSecretLister(ctrl)
	authClientSet := &authentication.AuthClientSet{
		CoreV1Interface: mockCoreV1Interface,
		SecretLister:    mockSecretLister,
	}

	for _, testData := range tokenTestDataSet {
		func() {
			vault := kedav1alpha1.HashiCorpVault{
				Address:        server.URL,
				Authentication: testData.authentication,
				Credential:     &testData.credential,
				Role:           testData.role,
				Mount:          testData.mount,
			}
			vaultHandler := NewHashicorpVaultHandler(&vault, authClientSet, "default")
			defer vaultHandler.Stop()

			config := vaultapi.DefaultConfig()
			client, err := vaultapi.NewClient(config)
			assert.Nil(t, err)
			token, err := vaultHandler.token(client)
			if testData.isError {
				assert.Equalf(t, vaultHandler.vault.Credential.ServiceAccount, testData.credential.ServiceAccount, "test %s: expected %s but found %s", testData.name, "random/path", vaultHandler.vault.Credential.ServiceAccount)
				assert.NotNilf(t, err, "test %s: expected error but got success, testData - %+v", testData.name, testData)
				assert.Contains(t, err.Error(), testData.errorMessage)
			} else {
				assert.Equalf(t, token, vaultTestToken, "expected %s but got %s", vaultTestToken, token)
			}
		}()
	}
}

func TestHashicorpVaultHandler_Token_ServiceAccountAuth(t *testing.T) {
	server := mockVault(t, false)
	defer server.Close()

	ctrl := gomock.NewController(t)
	mockCoreV1Interface := mock_serviceaccounts.NewMockCoreV1Interface(ctrl)
	mockSecretLister := mock_secretlister.NewMockSecretLister(ctrl)
	authClientSet := &authentication.AuthClientSet{
		CoreV1Interface: mockCoreV1Interface,
		SecretLister:    mockSecretLister,
	}

	defer ctrl.Finish()

	mockServiceAccountInterface := mockCoreV1Interface.GetServiceAccountInterface()
	tokenRequest := &authv1.TokenRequest{
		Status: authv1.TokenRequestStatus{
			Token: bsatData,
		},
	}
	mockServiceAccountInterface.EXPECT().CreateToken(gomock.Any(), gomock.Eq(bsatSAName), gomock.Any(), gomock.Any()).Return(tokenRequest, nil).AnyTimes()

	vault := kedav1alpha1.HashiCorpVault{
		Address:        server.URL,
		Authentication: kedav1alpha1.VaultAuthenticationKubernetes,
		Mount:          "kubernetes",
		Role:           "keda-role",
		Credential: &kedav1alpha1.Credential{
			ServiceAccountName: bsatSAName,
		},
	}

	vaultHandler := NewHashicorpVaultHandler(&vault, authClientSet, "default")
	defer vaultHandler.Stop()

	config := vaultapi.DefaultConfig()
	config.Address = server.URL
	client, err := vaultapi.NewClient(config)
	assert.NoError(t, err)

	token, err := vaultHandler.token(client)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}
