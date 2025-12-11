/*
Copyright 2025 The KEDA Authors

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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

const (
	akeylessTestToken     = "test-token-12345"
	akeylessTestAccessId  = "p-123456789012a3" // 14 char with 'a' (access_key) as second-to-last
	akeylessTestAccessKey = "test-access-key"
	testSecretValue       = "test-secret-value"
	testSecretPath        = "kv/prod/servicebus/password"
)

func mockAkeylessServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle POST requests with JSON body
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var reqBody map[string]interface{}
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Logf("Error decoding request body: %v", err)
			}
		}

		switch r.URL.Path {
		case "/api/v2/auth":
			// Mock authentication response
			authResp := map[string]interface{}{
				"token":      akeylessTestToken,
				"expiration": "2025-12-31T23:59:59Z",
			}
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(authResp); err != nil {
				t.Logf("Error encoding auth response: %v", err)
			}

		case "/api/v2/describe-item":
			// Mock describe item response
			// The SDK sends the item name in the request body as "name"
			var itemName string
			var ok bool

			if itemName, ok = reqBody["name"].(string); !ok {
				// Log the actual request body for debugging
				t.Logf("Describe item request body: %+v", reqBody)
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "item name not found in request",
				}); err != nil {
					t.Logf("Error encoding describe item response: %v", err)
				}
				return
			}

			// Return different item types based on path
			var itemType string
			switch itemName {
			case testSecretPath:
				itemType = "STATIC_SECRET"
			case "dynamic-secret/path":
				itemType = "DYNAMIC_SECRET"
			case "rotated-secret/path":
				itemType = "ROTATED_SECRET"
			default:
				w.WriteHeader(http.StatusNotFound)
				if err := json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "item not found",
				}); err != nil {
					t.Logf("Error encoding describe item response: %v", err)
				}
				return
			}

			describeResp := map[string]interface{}{
				"item_type": itemType,
			}
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(describeResp); err != nil {
				t.Logf("Error encoding describe item response: %v", err)
			}

		case "/api/v2/get-secret-value":
			// Mock get secret value response
			// The SDK sends names as an array
			var paths []interface{}
			var ok bool

			if paths, ok = reqBody["names"].([]interface{}); !ok {
				// Try alternative field name
				if paths, ok = reqBody["Names"].([]interface{}); !ok {
					t.Logf("Get secret value request body: %+v", reqBody)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			}

			if len(paths) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			path := paths[0].(string)
			secretResp := map[string]interface{}{
				path: testSecretValue,
			}
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(secretResp); err != nil {
				t.Logf("Error encoding get secret value response: %v", err)
			}

		case "/api/v2/get-dynamic-secret-value":
			// Mock dynamic secret response
			var _ string
			var ok bool

			if _, ok = reqBody["name"].(string); !ok {
				if _, ok = reqBody["Name"].(string); !ok {
					t.Logf("Get dynamic secret value request body: %+v", reqBody)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			}

			// Return JSON value for dynamic secrets
			dynamicValue := map[string]string{
				"username": "testuser",
				"password": "testpass",
			}
			valueJSON, _ := json.Marshal(dynamicValue)

			secretResp := map[string]interface{}{
				"value": string(valueJSON),
				"error": "",
			}
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(secretResp); err != nil {
				t.Logf("Error encoding get dynamic secret value response: %v", err)
			}

		case "/api/v2/get-rotated-secret-value":
			// Mock rotated secret response
			// Accept any request to this endpoint (SDK may send different field names)
			secretResp := map[string]interface{}{
				"value": map[string]string{
					"username": "rotateduser",
					"password": "rotatedpass",
				},
			}
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(secretResp); err != nil {
				t.Logf("Error encoding get rotated secret value response: %v", err)
			}

		default:
			t.Logf("Got request at path %s with method %s", r.URL.Path, r.Method)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server
}

func TestAkeylessHandler_Initialize_AccessKey(t *testing.T) {
	server := mockAkeylessServer(t)
	defer server.Close()

	accessKey := akeylessTestAccessKey
	akeyless := &kedav1alpha1.Akeyless{
		GatewayURL: server.URL + "/api/v2",
		AccessID:   akeylessTestAccessId,
		AccessKey:  &accessKey,
		Secrets: []kedav1alpha1.AkeylessSecret{
			{
				Parameter: "queuePassword",
				Path:      testSecretPath,
			},
		},
	}

	handler := &AkeylessHandler{
		akeyless: akeyless,
		logger:   logf.Log.WithName("test"),
	}
	ctx := context.Background()
	err := handler.Initialize(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, handler.client)
	assert.Equal(t, akeylessTestToken, handler.token)
}

func TestAkeylessHandler_Initialize_DefaultGatewayUrl(t *testing.T) {
	accessKey := akeylessTestAccessKey
	akeyless := &kedav1alpha1.Akeyless{
		AccessID:  akeylessTestAccessId,
		AccessKey: &accessKey,
		Secrets: []kedav1alpha1.AkeylessSecret{
			{
				Parameter: "queuePassword",
				Path:      testSecretPath,
			},
		},
	}

	handler := &AkeylessHandler{
		akeyless: akeyless,
		logger:   logf.Log.WithName("test"),
	}

	// Should use default gateway URL when not provided
	assert.Empty(t, akeyless.GatewayURL)

	// Note: This will fail authentication but tests the default URL logic
	ctx := context.Background()
	err := handler.Initialize(ctx)

	// We expect an error because we're not mocking the default URL
	assert.Error(t, err)
	// But the gateway URL should be set to default
	assert.Equal(t, publicGatewayURL, akeyless.GatewayURL)
}

func TestAkeylessHandler_Initialize_MissingAccessId(t *testing.T) {
	accessKey := akeylessTestAccessKey
	akeyless := &kedav1alpha1.Akeyless{
		GatewayURL: "https://test.akeyless.io",
		AccessKey:  &accessKey,
		Secrets: []kedav1alpha1.AkeylessSecret{
			{
				Parameter: "queuePassword",
				Path:      testSecretPath,
			},
		},
	}

	handler := &AkeylessHandler{
		akeyless: akeyless,
		logger:   logf.Log.WithName("test"),
	}
	ctx := context.Background()
	err := handler.Initialize(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accessId is required")
}

func TestAkeylessHandler_Initialize_InvalidGatewayUrl(t *testing.T) {
	accessKey := akeylessTestAccessKey
	akeyless := &kedav1alpha1.Akeyless{
		GatewayURL: "not-a-valid-url",
		AccessID:   akeylessTestAccessId,
		AccessKey:  &accessKey,
		Secrets: []kedav1alpha1.AkeylessSecret{
			{
				Parameter: "queuePassword",
				Path:      testSecretPath,
			},
		},
	}

	handler := &AkeylessHandler{
		akeyless: akeyless,
		logger:   logf.Log.WithName("test"),
	}
	ctx := context.Background()
	err := handler.Initialize(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid gateway URL")
}

func TestAkeylessHandler_GetSecretsValue_StaticSecret(t *testing.T) {
	server := mockAkeylessServer(t)
	defer server.Close()

	accessKey := akeylessTestAccessKey
	akeyless := &kedav1alpha1.Akeyless{
		GatewayURL: server.URL + "/api/v2",
		AccessID:   akeylessTestAccessId,
		AccessKey:  &accessKey,
		Secrets: []kedav1alpha1.AkeylessSecret{
			{
				Parameter: "queuePassword",
				Path:      testSecretPath,
			},
		},
	}

	handler := &AkeylessHandler{
		akeyless: akeyless,
		logger:   logf.Log.WithName("test"),
	}
	ctx := context.Background()
	err := handler.Initialize(ctx)
	require.NoError(t, err)

	secretResults := make(map[string]string)
	result, err := handler.GetSecretsValue(ctx, secretResults)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, testSecretValue, result["queuePassword"])
}

func TestAkeylessHandler_GetSecretsValue_StaticSecretWithKey(t *testing.T) {
	server := mockAkeylessServer(t)
	defer server.Close()

	accessKey := akeylessTestAccessKey
	akeyless := &kedav1alpha1.Akeyless{
		GatewayURL: server.URL + "/api/v2",
		AccessID:   akeylessTestAccessId,
		AccessKey:  &accessKey,
		Secrets: []kedav1alpha1.AkeylessSecret{
			{
				Parameter: "password",
				Path:      testSecretPath,
				Key:       "password",
			},
		},
	}

	// Mock a JSON response for static secret with key
	jsonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/auth":
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"token":      akeylessTestToken,
				"expiration": "2025-12-31T23:59:59Z",
			}); err != nil {
				t.Logf("Error encoding auth response: %v", err)
			}
		case "/api/v2/describe-item":
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"item_type": "STATIC_SECRET",
			}); err != nil {
				t.Logf("Error encoding describe item response: %v", err)
			}
		case "/api/v2/get-secret-value":
			// Return JSON string
			jsonValue := `{"username":"testuser","password":"testpass"}`
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				testSecretPath: jsonValue,
			}); err != nil {
				t.Logf("Error encoding get secret value response: %v", err)
			}
		}
	}))
	defer jsonServer.Close()

	akeyless.GatewayURL = jsonServer.URL + "/api/v2"
	handler := &AkeylessHandler{
		akeyless: akeyless,
		logger:   logf.Log.WithName("test"),
	}
	ctx := context.Background()
	err := handler.Initialize(ctx)
	require.NoError(t, err)

	secretResults := make(map[string]string)
	result, err := handler.GetSecretsValue(ctx, secretResults)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "testpass", result["password"])
}

func TestAkeylessHandler_GetSecretsValue_MultipleSecrets(t *testing.T) {
	server := mockAkeylessServer(t)
	defer server.Close()

	accessKey := akeylessTestAccessKey
	akeyless := &kedav1alpha1.Akeyless{
		GatewayURL: server.URL + "/api/v2",
		AccessID:   akeylessTestAccessId,
		AccessKey:  &accessKey,
		Secrets: []kedav1alpha1.AkeylessSecret{
			{
				Parameter: "secret1",
				Path:      testSecretPath,
			},
			{
				Parameter: "secret2",
				Path:      testSecretPath,
			},
		},
	}

	handler := &AkeylessHandler{
		akeyless: akeyless,
		logger:   logf.Log.WithName("test"),
	}
	ctx := context.Background()
	err := handler.Initialize(ctx)
	require.NoError(t, err)

	secretResults := make(map[string]string)
	result, err := handler.GetSecretsValue(ctx, secretResults)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, testSecretValue, result["secret1"])
	assert.Equal(t, testSecretValue, result["secret2"])
}

func TestAkeylessHandler_GetSecretsValue_NonExistentSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v2/auth":
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"token":      akeylessTestToken,
				"expiration": "2025-12-31T23:59:59Z",
			}); err != nil {
				t.Logf("Error encoding auth response: %v", err)
			}
		case "/api/v2/describe-item":
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	accessKey := akeylessTestAccessKey
	akeyless := &kedav1alpha1.Akeyless{
		GatewayURL: server.URL + "/api/v2",
		AccessID:   akeylessTestAccessId,
		AccessKey:  &accessKey,
		Secrets: []kedav1alpha1.AkeylessSecret{
			{
				Parameter: "queuePassword",
				Path:      "non-existent/path",
			},
		},
	}

	handler := &AkeylessHandler{
		akeyless: akeyless,
		logger:   logf.Log.WithName("test"),
	}
	ctx := context.Background()
	err := handler.Initialize(ctx)
	require.NoError(t, err)

	secretResults := make(map[string]string)
	_, err = handler.GetSecretsValue(ctx, secretResults)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get secret type")
}

func TestExtractAccessTypeChar(t *testing.T) {
	tests := []struct {
		name      string
		accessId  string
		want      string
		wantError bool
	}{
		{
			name:      "valid 14 char access ID with access_key type",
			accessId:  "p-123456789012a3",
			want:      "a",
			wantError: false,
		},
		{
			name:      "valid 12 char access ID with k8s type",
			accessId:  "p-1234567890k1",
			want:      "k",
			wantError: false,
		},
		{
			name:      "invalid format - missing prefix",
			accessId:  "1234567890123a",
			want:      "",
			wantError: true,
		},
		{
			name:      "invalid format - wrong prefix",
			accessId:  "x-1234567890123a",
			want:      "",
			wantError: true,
		},
		{
			name:      "invalid format - too short",
			accessId:  "p-123",
			want:      "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractAccessTypeChar(tt.accessId)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetAccessTypeDisplayName(t *testing.T) {
	tests := []struct {
		name      string
		typeChar  string
		want      string
		wantError bool
	}{
		{
			name:      "access_key type",
			typeChar:  "a",
			want:      authAccessKey,
			wantError: false,
		},
		{
			name:      "k8s type",
			typeChar:  "k",
			want:      authK8s,
			wantError: false,
		},
		{
			name:      "aws_iam type",
			typeChar:  "w",
			want:      authAwsIam,
			wantError: false,
		},
		{
			name:      "gcp type",
			typeChar:  "g",
			want:      authGcp,
			wantError: false,
		},
		{
			name:      "azure_ad type",
			typeChar:  "z",
			want:      authAzureAd,
			wantError: false,
		},
		{
			name:      "unknown type",
			typeChar:  "x",
			want:      "",
			wantError: true,
		},
		{
			name:      "empty type",
			typeChar:  "",
			want:      "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAccessTypeDisplayName(tt.typeChar)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIsValidAccessIdFormat(t *testing.T) {
	tests := []struct {
		name     string
		accessId string
		want     bool
	}{
		{
			name:     "valid 14 char",
			accessId: "p-123456789012a3",
			want:     true,
		},
		{
			name:     "valid 12 char",
			accessId: "p-1234567890k1",
			want:     true,
		},
		{
			name:     "invalid - missing prefix",
			accessId: "1234567890123a",
			want:     false,
		},
		{
			name:     "invalid - wrong prefix",
			accessId: "x-1234567890123a",
			want:     false,
		},
		{
			name:     "invalid - too short",
			accessId: "p-123",
			want:     false,
		},
		{
			name:     "invalid - too long",
			accessId: "p-123456789012345a6",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidAccessIDFormat(tt.accessId)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAkeylessHandler_parseStaticSecretValue(t *testing.T) {
	handler := &AkeylessHandler{
		logger: logf.Log.WithName("test"),
	}

	tests := []struct {
		name           string
		path           string
		key            string
		secretValueStr string
		want           string
		wantError      bool
		errorContains  string
	}{
		{
			name:           "plain string value without key",
			path:           "test/path",
			key:            "",
			secretValueStr: "plain-secret-value",
			want:           "plain-secret-value",
			wantError:      false,
		},
		{
			name:           "JSON string without key - returns stringified JSON",
			path:           "test/path",
			key:            "",
			secretValueStr: `{"username":"testuser","password":"testpass"}`,
			want:           `{"password":"testpass","username":"testuser"}`,
			wantError:      false,
		},
		{
			name:           "JSON string with key - returns value for key",
			path:           "test/path",
			key:            "password",
			secretValueStr: `{"username":"testuser","password":"testpass"}`,
			want:           "testpass",
			wantError:      false,
		},
		{
			name:           "JSON string with non-existent key - returns error",
			path:           "test/path",
			key:            "nonexistent",
			secretValueStr: `{"username":"testuser","password":"testpass"}`,
			want:           "",
			wantError:      true,
			errorContains:  "key 'nonexistent' not found",
		},
		{
			name:           "invalid JSON string - returns as-is",
			path:           "test/path",
			key:            "",
			secretValueStr: `not-valid-json{`,
			want:           `not-valid-json{`,
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := handler.parseStaticSecretValue(tt.path, tt.key, tt.secretValueStr)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				// For JSON comparison, we need to handle unordered keys
				if tt.key == "" && (tt.secretValueStr[0] == '{' || tt.secretValueStr[0] == '[') {
					// Compare as JSON objects
					var gotJSON, wantJSON map[string]interface{}
					require.NoError(t, json.Unmarshal([]byte(got), &gotJSON))
					require.NoError(t, json.Unmarshal([]byte(tt.want), &wantJSON))
					assert.Equal(t, wantJSON, gotJSON)
				} else {
					assert.Equal(t, tt.want, got)
				}
			}
		})
	}
}

func TestAkeylessHandler_extractSecretValueFromMap(t *testing.T) {
	handler := &AkeylessHandler{
		logger: logf.Log.WithName("test"),
	}

	tests := []struct {
		name          string
		path          string
		key           string
		mapValue      map[string]string
		secretType    string
		want          string
		wantError     bool
		errorContains string
	}{
		{
			name:       "no key provided - returns stringified JSON",
			path:       "test/path",
			key:        "",
			mapValue:   map[string]string{"username": "testuser", "password": "testpass"},
			secretType: "dynamic",
			want:       `{"password":"testpass","username":"testuser"}`,
			wantError:  false,
		},
		{
			name:       "key provided and exists - returns value",
			path:       "test/path",
			key:        "password",
			mapValue:   map[string]string{"username": "testuser", "password": "testpass"},
			secretType: "dynamic",
			want:       "testpass",
			wantError:  false,
		},
		{
			name:          "key provided but doesn't exist - returns error",
			path:          "test/path",
			key:           "nonexistent",
			mapValue:      map[string]string{"username": "testuser", "password": "testpass"},
			secretType:    "dynamic",
			want:          "",
			wantError:     true,
			errorContains: "key 'nonexistent' not found",
		},
		{
			name:       "rotated secret with no key",
			path:       "test/path",
			key:        "",
			mapValue:   map[string]string{"username": "rotateduser", "password": "rotatedpass"},
			secretType: "rotated",
			want:       `{"password":"rotatedpass","username":"rotateduser"}`,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := handler.extractSecretValueFromMap(tt.path, tt.key, tt.mapValue, tt.secretType)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				// For JSON comparison, handle unordered keys
				if tt.key == "" {
					var gotJSON, wantJSON map[string]interface{}
					require.NoError(t, json.Unmarshal([]byte(got), &gotJSON))
					require.NoError(t, json.Unmarshal([]byte(tt.want), &wantJSON))
					assert.Equal(t, wantJSON, gotJSON)
				} else {
					assert.Equal(t, tt.want, got)
				}
			}
		})
	}
}

func TestAkeylessHandler_getStaticSecretValue(t *testing.T) {
	server := mockAkeylessServer(t)
	defer server.Close()

	accessKey := akeylessTestAccessKey
	akeyless := &kedav1alpha1.Akeyless{
		GatewayURL: server.URL + "/api/v2",
		AccessID:   akeylessTestAccessId,
		AccessKey:  &accessKey,
		Secrets: []kedav1alpha1.AkeylessSecret{
			{
				Parameter: "queuePassword",
				Path:      testSecretPath,
			},
		},
	}

	handler := &AkeylessHandler{
		akeyless: akeyless,
		logger:   logf.Log.WithName("test"),
	}
	ctx := context.Background()
	err := handler.Initialize(ctx)
	require.NoError(t, err)

	secret := kedav1alpha1.AkeylessSecret{
		Parameter: "queuePassword",
		Path:      testSecretPath,
	}

	value, err := handler.getStaticSecretValue(ctx, secret)
	assert.NoError(t, err)
	assert.Equal(t, testSecretValue, value)
}

func TestAkeylessHandler_getDynamicSecretValue(t *testing.T) {
	server := mockAkeylessServer(t)
	defer server.Close()

	accessKey := akeylessTestAccessKey
	akeyless := &kedav1alpha1.Akeyless{
		GatewayURL: server.URL + "/api/v2",
		AccessID:   akeylessTestAccessId,
		AccessKey:  &accessKey,
		Secrets: []kedav1alpha1.AkeylessSecret{
			{
				Parameter: "dynamicSecret",
				Path:      "dynamic-secret/path",
			},
		},
	}

	handler := &AkeylessHandler{
		akeyless: akeyless,
		logger:   logf.Log.WithName("test"),
	}
	ctx := context.Background()
	err := handler.Initialize(ctx)
	require.NoError(t, err)

	secret := kedav1alpha1.AkeylessSecret{
		Parameter: "dynamicSecret",
		Path:      "dynamic-secret/path",
		Key:       "username",
	}

	value, err := handler.getDynamicSecretValue(ctx, secret)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", value)
}

func TestAkeylessHandler_getRotatedSecretValue(t *testing.T) {
	server := mockAkeylessServer(t)
	defer server.Close()

	accessKey := akeylessTestAccessKey
	akeyless := &kedav1alpha1.Akeyless{
		GatewayURL: server.URL + "/api/v2",
		AccessID:   akeylessTestAccessId,
		AccessKey:  &accessKey,
		Secrets: []kedav1alpha1.AkeylessSecret{
			{
				Parameter: "rotatedSecret",
				Path:      "rotated-secret/path",
			},
		},
	}

	handler := &AkeylessHandler{
		akeyless: akeyless,
		logger:   logf.Log.WithName("test"),
	}
	ctx := context.Background()
	err := handler.Initialize(ctx)
	require.NoError(t, err)

	secret := kedav1alpha1.AkeylessSecret{
		Parameter: "rotatedSecret",
		Path:      "rotated-secret/path",
		Key:       "username",
	}

	value, err := handler.getRotatedSecretValue(ctx, secret)
	assert.NoError(t, err)
	assert.Equal(t, "rotateduser", value)
}
