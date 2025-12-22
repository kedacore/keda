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
