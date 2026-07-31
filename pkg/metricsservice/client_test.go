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

package metricsservice

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// newTLSTestServerAndClientConn starts a gRPC server on a random local port secured
// with a self-signed certificate for 127.0.0.1, and returns a gRPC client connection
// trusting that certificate.
func newTLSTestServerAndClientConn(t *testing.T) (*grpc.Server, net.Listener, *grpc.ClientConn) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	tlsCert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(cert)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	serverCreds := credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{tlsCert}, MinVersion: tls.VersionTLS13})
	server := grpc.NewServer(grpc.Creds(serverCreds))
	go func() {
		_ = server.Serve(lis)
	}()

	clientCreds := credentials.NewTLS(&tls.Config{RootCAs: certPool, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS13})
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(clientCreds))
	require.NoError(t, err)

	return server, lis, conn
}

func TestWaitWhileConnectionReady(t *testing.T) {
	server, lis, conn := newTLSTestServerAndClientConn(t)
	defer server.Stop()
	defer conn.Close()

	client := &GrpcClient{connection: conn}
	logger := logr.Discard()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.True(t, client.WaitForConnectionReady(ctx, logger), "connection should become ready")

	done := make(chan bool, 1)
	go func() {
		done <- client.WaitWhileConnectionReady(ctx, logger)
	}()

	// Stop the server to force the connection out of the Ready state.
	server.Stop()
	_ = lis.Close()

	select {
	case result := <-done:
		require.True(t, result, "WaitWhileConnectionReady should return true once the connection leaves the Ready state")
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for WaitWhileConnectionReady to detect the connection leaving the Ready state")
	}
}

func TestWaitWhileConnectionReadyContextCancelled(t *testing.T) {
	server, lis, conn := newTLSTestServerAndClientConn(t)
	defer server.Stop()
	defer lis.Close()
	defer conn.Close()

	client := &GrpcClient{connection: conn}
	logger := logr.Discard()

	readyCtx, readyCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer readyCancel()
	require.True(t, client.WaitForConnectionReady(readyCtx, logger), "connection should become ready")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.False(t, client.WaitWhileConnectionReady(ctx, logger), "should return false when context is already cancelled")
}
