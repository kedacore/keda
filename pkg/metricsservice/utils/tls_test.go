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
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
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

func handshake(client, server credentials.TransportCredentials) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()

	serverErr := make(chan error, 1)
	go func() {
		rawConn, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		conn, _, err := server.ServerHandshake(rawConn)
		if conn != nil {
			defer conn.Close()
		}
		serverErr <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	clientConn, err := net.DialTimeout("tcp", listener.Addr().String(), 2*time.Second)
	if err != nil {
		return err
	}
	conn, _, clientErr := client.ClientHandshake(ctx, "localhost", clientConn)
	if conn != nil {
		defer conn.Close()
	}
	if clientErr != nil {
		return clientErr
	}
	return <-serverErr
}

func staticTLSCredentials(t *testing.T, dir string, server bool) credentials.TransportCredentials {
	t.Helper()
	pool, err := buildCertPool(filepath.Join(dir, "ca.crt"))
	require.NoError(t, err)
	cert, err := tls.LoadX509KeyPair(filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key"))
	require.NoError(t, err)
	config := &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}}
	if server {
		config.ClientAuth = tls.RequireAndVerifyClientCert
		config.ClientCAs = pool
	} else {
		config.RootCAs = pool
	}
	return credentials.NewTLS(config)
}

func generateTestCertAndKeyWithCA(t *testing.T, dir string) (credentials.TransportCredentials, credentials.TransportCredentials) {
	t.Helper()
	generateTestCertAndKey(t, dir)
	return staticTLSCredentials(t, dir, false), staticTLSCredentials(t, dir, true)
}

func rotateFiles(dir string) error {
	dataDir := filepath.Join(dir, "..data")
	_ = os.RemoveAll(dataDir)
	return os.Mkdir(dataDir, 0700)
}

func triggerRotation(t *testing.T, dir string) (credentials.TransportCredentials, credentials.TransportCredentials) {
	t.Helper()
	client, server := generateTestCertAndKeyWithCA(t, dir)
	require.NoError(t, rotateFiles(dir))
	return client, server
}

func TestLoadGrpcTLSCredentialsConcurrentRotationRace(t *testing.T) {
	dir := t.TempDir()
	oldClient, oldServer := generateTestCertAndKeyWithCA(t, dir)
	oldCertificate, err := tls.LoadX509KeyPair(filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key"))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverCreds, err := LoadGrpcTLSCredentials(ctx, dir, true)
	require.NoError(t, err)
	clientCreds, err := LoadGrpcTLSCredentials(ctx, dir, false)
	require.NoError(t, err)
	require.NoError(t, handshake(clientCreds, serverCreds))
	require.NoError(t, handshake(oldClient, serverCreds))
	require.NoError(t, handshake(clientCreds, oldServer))

	// Rotate credentials to a new CA
	freshClient, freshServer := triggerRotation(t, dir)

	// Verify that dynamic credentials pick up the new CA material
	require.Eventually(t, func() bool {
		clientErr := handshake(clientCreds, freshServer)
		serverErr := handshake(freshClient, serverCreds)
		return clientErr == nil && serverErr == nil
	}, 5*time.Second, 50*time.Millisecond, "client and server should accept the rotated certificate material")

	probePool, err := buildCertPool(filepath.Join(dir, "ca.crt"))
	require.NoError(t, err)
	oldCA, err := x509.ParseCertificate(oldCertificate.Certificate[0])
	require.NoError(t, err)
	probePool.AddCert(oldCA)

	// Each probe trusts both generations but presents its original certificate.
	// Only the dynamic peer should reject the old certificate.
	staleClient := credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{oldCertificate},
		RootCAs:      probePool,
	})
	staleServer := credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{oldCertificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    probePool,
	})
	require.Error(t, handshake(staleClient, serverCreds), "rotated server must reject the old client certificate")
	require.Error(t, handshake(clientCreds, staleServer), "rotated client must reject the old server certificate")

	// Run concurrent handshakes while actively rotating to verify race-free synchronization
	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Goroutine performing active rotations
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				if err := rotateFiles(dir); err != nil {
					t.Errorf("rotation failed: %v", err)
					return
				}
				time.Sleep(15 * time.Millisecond)
			}
		}
	}()

	// Multiple worker goroutines performing concurrent handshakes
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					// Dynamic client <-> Dynamic server handshake should not panic or race
					_ = handshake(clientCreds, serverCreds)
					time.Sleep(5 * time.Millisecond)
				}
			}
		}()
	}

	time.Sleep(300 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestLoadGrpcTLSCredentialsWithAuthority(t *testing.T) {
	dir := t.TempDir()
	generateTestCertAndKey(t, dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverCreds, err := LoadGrpcTLSCredentials(ctx, dir, true)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	s := grpc.NewServer(grpc.Creds(serverCreds))
	go func() {
		_ = s.Serve(lis)
	}()
	defer s.Stop()

	clientCreds, err := LoadGrpcTLSCredentials(ctx, dir, false)
	require.NoError(t, err)

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(clientCreds),
		grpc.WithAuthority("localhost"),
	)
	require.NoError(t, err)
	defer conn.Close()

	conn.Connect()
	require.Eventually(t, func() bool {
		return conn.GetState() == connectivity.Ready
	}, 5*time.Second, 50*time.Millisecond, "connection with authority localhost to 127.0.0.1 must succeed")
}
