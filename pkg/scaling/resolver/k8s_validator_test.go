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
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestReadKubernetesServiceAccountProjectedToken(t *testing.T) {
	tests := []struct {
		name        string
		setupToken  func() string
		expectError bool
		validate    func([]byte) bool
	}{
		{
			name: "valid token",
			setupToken: func() string {
				privateKey, err := generateTestRSAKeyPair()
				if err != nil {
					t.Fatalf("failed to generate RSA keys: %v", err)
				}

				// Create valid JWT token
				claims := jwt.MapClaims{
					"iss": "kubernetes/serviceaccount",
					"sub": "system:serviceaccount:default:default",
					"exp": time.Now().Add(time.Hour).Unix(),
					"iat": time.Now().Unix(),
				}
				tokenBytes, err := createJWTToken(privateKey, claims)
				if err != nil {
					t.Fatalf("failed to create JWT token: %v", err)
				}
				tokenPath := createTempFile(t, tokenBytes)

				return tokenPath
			},
			expectError: false,
			validate: func(token []byte) bool {
				return len(token) > 0
			},
		},
		{
			name: "token file does not exist",
			setupToken: func() string {
				return "/nonexistent/token/path"
			},
			expectError: true,
		},
		{
			name: "arbitrary file content is not a valid token",
			setupToken: func() string {
				// Create an arbitrary file with random content that is not a JWT
				arbitraryContent := []byte("This is just arbitrary file content, not a JWT token at all")
				tokenPath := createTempFile(t, arbitraryContent)

				return tokenPath
			},
			expectError: true,
		},
		{
			name: "not sa token",
			setupToken: func() string {
				privateKey, err := generateTestRSAKeyPair()
				if err != nil {
					t.Fatalf("failed to generate RSA keys: %v", err)
				}

				// Create valid JWT token but not from k8s
				claims := jwt.MapClaims{
					"iss": "random-issuer",
					"sub": "1234-3212",
					"exp": time.Now().Add(time.Hour).Unix(),
					"iat": time.Now().Unix(),
				}
				tokenBytes, err := createJWTToken(privateKey, claims)
				assert.NoError(t, err)
				tokenPath := createTempFile(t, tokenBytes)

				return tokenPath
			},
			expectError: true,
		},
		{
			name: "token file exceeds maximum size",
			setupToken: func() string {
				return createTempFile(t, bytes.Repeat([]byte("x"), maxProjectedServiceAccountTokenSize+1))
			},
			expectError: true,
		},
		{
			name: "token path is not a regular file",
			setupToken: func() string {
				return t.TempDir()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenPath := tt.setupToken()
			defer os.Remove(tokenPath)

			result, err := readKubernetesServiceAccountProjectedToken(tokenPath)

			if (err != nil) != tt.expectError {
				t.Errorf("readKubernetesServiceAccountProjectedToken() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if !tt.expectError && tt.validate != nil {
				if !tt.validate(result) {
					t.Errorf("readKubernetesServiceAccountProjectedToken() returned invalid result")
				}
			}
		})
	}
}

func TestValidateK8sSATokenAudiences(t *testing.T) {
	privateKey, err := generateTestRSAKeyPair()
	assert.NoError(t, err)

	tests := []struct {
		name      string
		audiences any
		expected  []string
		wantError bool
	}{
		{name: "single matching audience", audiences: []string{"vault.example"}, expected: []string{"vault.example"}},
		{name: "string audience", audiences: "vault.example", expected: []string{"vault.example"}},
		{name: "missing audience", expected: []string{"vault.example"}, wantError: true},
		{name: "wrong audience", audiences: []string{"kube-apiserver"}, expected: []string{"vault.example"}, wantError: true},
		{name: "additional audience", audiences: []string{"vault.example", "kube-apiserver"}, expected: []string{"vault.example"}, wantError: true},
		{name: "no required audience", audiences: []string{"vault.example"}, wantError: true},
		{name: "subset of approved audiences", audiences: []string{"other.example"}, expected: []string{"vault.example", "other.example"}},
		{name: "all audiences approved", audiences: []string{"other.example", "vault.example"}, expected: []string{"vault.example", "other.example"}},
		{name: "duplicates do not expand privileges", audiences: []string{"vault.example", "vault.example"}, expected: []string{"vault.example", "other.example"}},
		{name: "one approved audience cannot hide another", audiences: []string{"vault.example", "api"}, expected: []string{"vault.example", "other.example"}, wantError: true},
		{name: "empty audience", audiences: []string{""}, expected: []string{""}, wantError: true},
		{name: "empty audience list", audiences: []string{}, expected: []string{"vault.example"}, wantError: true},
		{name: "wrong claim type", audiences: 1, expected: []string{"vault.example"}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := jwt.MapClaims{
				"iss": "kubernetes/serviceaccount",
				"sub": "system:serviceaccount:default:default",
				"exp": time.Now().Add(time.Hour).Unix(),
			}
			if tt.audiences != nil {
				claims["aud"] = tt.audiences
			}
			token, err := createJWTToken(privateKey, claims)
			assert.NoError(t, err)

			err = validateK8sSATokenAudiences(token, tt.expected)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateK8sSATokenLifetime(t *testing.T) {
	key, err := generateTestRSAKeyPair()
	assert.NoError(t, err)
	for _, lifetime := range []jwt.MapClaims{
		{}, {"exp": time.Now().Add(-time.Minute).Unix()},
		{"exp": "invalid"},
		{"exp": time.Now().Add(time.Hour).Unix(), "nbf": time.Now().Add(time.Minute).Unix()},
	} {
		lifetime["sub"] = "system:serviceaccount:default:default"
		lifetime["aud"] = []string{"vault"}
		token, err := createJWTToken(key, lifetime)
		assert.NoError(t, err)
		assert.Error(t, validateK8sSATokenAudiences(token, []string{"vault"}))
	}
}

// Helper function to generate RSA key pair for testing
func generateTestRSAKeyPair() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// Helper function to create a valid JWT token for testing
func createJWTToken(privateKey *rsa.PrivateKey, claims jwt.MapClaims) ([]byte, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return nil, err
	}
	return []byte(tokenString), nil
}

// Helper function to create temporary files for testing
func createTempFile(t *testing.T, content []byte) string {
	tmpFile, err := os.CreateTemp("", "k8s_test_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
}
