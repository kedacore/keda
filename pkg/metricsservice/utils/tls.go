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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path"

	"google.golang.org/grpc/credentials"
	ctrl "sigs.k8s.io/controller-runtime"

	kedatls "github.com/kedacore/keda/v2/pkg/common/tls"
)

// LoadGrpcTLSCredentials reads the certificate from the given path and returns TLS transport credentials
func LoadGrpcTLSCredentials(certDir string, server bool) (credentials.TransportCredentials, error) {
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

	// Load certificate and private key
	cert, err := tls.LoadX509KeyPair(path.Join(certDir, "tls.crt"), path.Join(certDir, "tls.key"))
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	minTLSVersion, err := kedatls.GetMinGrpcTLSVersion()
	if err != nil {
		ctrl.Log.WithName("grpc_tls_setup").Info(err.Error())
	}
	config := &tls.Config{
		MinVersion:   minTLSVersion,
		Certificates: []tls.Certificate{cert},
	}
	if server {
		config.ClientAuth = tls.RequireAndVerifyClientCert
		config.ClientCAs = certPool
	} else {
		config.RootCAs = certPool
	}

	return credentials.NewTLS(config), nil
}
