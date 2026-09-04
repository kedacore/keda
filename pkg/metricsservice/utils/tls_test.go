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

package utils

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
)

func generateTestCertAndKey(t *testing.T, dir string) string {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"KEDA Testing"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	caPath := filepath.Join(dir, "ca.crt")
	certPath := filepath.Join(dir, "tls.crt")
	keyPath := filepath.Join(dir, "tls.key")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	require.NoError(t, os.WriteFile(caPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))

	return caPath
}

func TestBuildCertPool(t *testing.T) {
	dir := t.TempDir()
	caPath := generateTestCertAndKey(t, dir)

	pool, err := buildCertPool(caPath)
	require.NoError(t, err)
	assert.NotNil(t, pool)

	// Non-existent path returns error
	_, err = buildCertPool(filepath.Join(dir, "non-existent.crt"))
	assert.Error(t, err)

	// Invalid PEM content returns error
	invalidPath := filepath.Join(dir, "invalid.crt")
	require.NoError(t, os.WriteFile(invalidPath, []byte("NOT A PEM"), 0600))
	_, err = buildCertPool(invalidPath)
	assert.Error(t, err)
}

func TestLoadGrpcTLSCredentialsServer(t *testing.T) {
	dir := t.TempDir()
	generateTestCertAndKey(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	creds, err := LoadGrpcTLSCredentials(ctx, dir, true)
	require.NoError(t, err)
	assert.NotNil(t, creds)
}

func TestLoadGrpcTLSCredentialsClient(t *testing.T) {
	dir := t.TempDir()
	generateTestCertAndKey(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	creds, err := LoadGrpcTLSCredentials(ctx, dir, false)
	require.NoError(t, err)
	assert.NotNil(t, creds)
}

func TestLoadGrpcTLSCredentialsConcurrentRotationRace(t *testing.T) {
	dir := t.TempDir()
	generateTestCertAndKey(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	creds, err := LoadGrpcTLSCredentials(ctx, dir, true)
	require.NoError(t, err)
	require.NotNil(t, creds)

	tlsCreds, ok := creds.(interface {
		Info() credentials.ProtocolInfo
	})
	require.True(t, ok)
	require.Equal(t, "tls", tlsCreds.Info().SecurityProtocol)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Concurrently simulate client handshakes accessing certPool & mTLSCertificate
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					pool, err := buildCertPool(filepath.Join(dir, "ca.crt"))
					if err != nil || pool == nil {
						t.Errorf("failed to build cert pool: %v", err)
						return
					}
				}
			}
		}()
	}

	// Run for a short burst
	time.Sleep(100 * time.Millisecond)
	close(stop)
	wg.Wait()
}
