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

package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc/credentials"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("grpc_server_certificates")

// LoadGrpcTLSCredentials reads the certificate from the given path and returns TLS transport credentials
func LoadGrpcTLSCredentials(ctx context.Context, certDir string, server bool) (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed client's certificate
	pemClientCA, err := os.ReadFile(path.Join(certDir, "ca.crt"))
	if err != nil {
		return nil, err
	}

	// Get the SystemCertPool, continue with an empty pool on error
	certPool, _ := x509.SystemCertPool()
	if certPool == nil {
		certPool = x509.NewCertPool()
	}
	if !certPool.AppendCertsFromPEM(pemClientCA) {
		return nil, fmt.Errorf("failed to add client CA's certificate")
	}

	// Load initial certificate and private key
	mTLSCertificate, err := tls.LoadX509KeyPair(path.Join(certDir, "tls.crt"), path.Join(certDir, "tls.key"))
	if err != nil {
		return nil, err
	}

	// Start the watcher for cert updates
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	certMutex := sync.RWMutex{}
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					continue
				}
				if event.Has(fsnotify.Write) {
					// Load certificate of the CA who signed client's certificate
					pemClientCA, err := os.ReadFile(path.Join(certDir, "ca.crt"))
					if err != nil {
						log.Error(err, "error reading grpc ca certificate")
						continue
					}
					if !certPool.AppendCertsFromPEM(pemClientCA) {
						log.Error(err, "failed to add client CA's certificate")
						continue
					}

					cert, err := tls.LoadX509KeyPair(path.Join(certDir, "tls.crt"), path.Join(certDir, "tls.key"))
					if err != nil {
						log.Error(err, "error reading grpc certificate")
						continue
					}
					certMutex.Lock()
					mTLSCertificate = cert
					certMutex.Unlock()
					log.V(1).Info("grcp mTLS certificate has been updated")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					continue
				}
				log.Error(err, "error reading grpc certificate changes")
			case <-ctx.Done():
				return
			}
		}
	}()
	err = watcher.Add(certDir)
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		MinVersion: tls.VersionTLS13,
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			certMutex.RLock()
			defer certMutex.RUnlock()
			return &mTLSCertificate, nil
		},
		GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			certMutex.RLock()
			defer certMutex.RUnlock()
			return &mTLSCertificate, nil
		},
	}
	if server {
		config.ClientAuth = tls.RequireAndVerifyClientCert
		config.ClientCAs = certPool
	} else {
		config.RootCAs = certPool
	}

	return credentials.NewTLS(config), nil
}
