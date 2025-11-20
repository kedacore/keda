/*
Copyright 2021 The KEDA Authors

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

package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/youmark/pkcs8"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kedacore/keda/v2/pkg/metricsservice/utils"
)

var minTLSVersion uint16

func init() {
	var err error

	if minTLSVersion, err = initMinTLSVersion(); err != nil {
		ctrl.Log.WithName("tls_setup").Info(err.Error())
	}
}

// NewTLSConfigWithPassword returns a *tls.Config using the given ceClient cert, ceClient key,
// and CA certificate. If clientKeyPassword is not empty the provided password will be used to
// decrypt the given key. If none are appropriate, a nil *tls.Config is returned.
func NewTLSConfigWithPassword(clientCert, clientKey, clientKeyPassword, caCert string, unsafeSsl bool) (*tls.Config, error) {
	config := CreateTLSClientConfig(unsafeSsl)

	if clientCert != "" && clientKey != "" {
		key := []byte(clientKey)
		if clientKeyPassword != "" {
			var err error
			key, err = decryptClientKey(clientKey, clientKeyPassword)
			if err != nil {
				return nil, fmt.Errorf("error decrypt X509Key: %w", err)
			}
		}

		cert, err := tls.X509KeyPair([]byte(clientCert), key)
		if err != nil {
			return nil, fmt.Errorf("error parse X509KeyPair: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	if caCert != "" {
		config.RootCAs.AppendCertsFromPEM([]byte(caCert))
	}

	return config, nil
}

// NewTLSConfig returns a *tls.Config using the given ceClient cert, ceClient key,
// and CA certificate. If none are appropriate, a nil *tls.Config is returned.
func NewTLSConfig(clientCert, clientKey, caCert string, unsafeSsl bool) (*tls.Config, error) {
	return NewTLSConfigWithPassword(clientCert, clientKey, "", caCert, unsafeSsl)
}

// NewTLSConfigFromFiles returns a *tls.Config using the given the paths to key, cert and ca cert. If caCertPem is not empty,
// the cert will be loaded from this in-memory representation, otherwise the caCertFile file will be tried.
// If none are appropriate, a nil *tls.Config is returned.
func NewTLSConfigFromFiles(clientCertFile, clientKeyFile, caCertFile string, unsafeSsl bool) (*tls.Config, error) {
	return utils.LoadGrpcTLSConfig(context.Background(), caCertFile, clientCertFile, clientKeyFile, false, unsafeSsl)
}

// CreateTLSClientConfig returns a new TLS Config
// unsafeSsl parameter allows to avoid tls cert validation if it's required
func CreateTLSClientConfig(unsafeSsl bool) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: unsafeSsl,
		RootCAs:            getRootCAs(),
		MinVersion:         GetMinTLSVersion(),
	}
}

// GetMinTLSVersion return the minTLSVersion based on configurations
func GetMinTLSVersion() uint16 {
	return minTLSVersion
}

func initMinTLSVersion() (uint16, error) {
	version, _ := os.LookupEnv("KEDA_HTTP_MIN_TLS_VERSION")

	switch version {
	case "":
		minTLSVersion = tls.VersionTLS12
	case "TLS10":
		minTLSVersion = tls.VersionTLS10
	case "TLS11":
		minTLSVersion = tls.VersionTLS11
	case "TLS12":
		minTLSVersion = tls.VersionTLS12
	case "TLS13":
		minTLSVersion = tls.VersionTLS13
	default:
		return tls.VersionTLS12, fmt.Errorf("%s is not a valid value, using `TLS12`. Allowed values are: `TLS13`,`TLS12`,`TLS11`,`TLS10`", version)
	}

	return minTLSVersion, nil
}

func decryptClientKey(clientKey, clientKeyPassword string) ([]byte, error) {
	block, _ := pem.Decode([]byte(clientKey))

	key, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, []byte(clientKeyPassword))
	if err != nil {
		return nil, err
	}

	pemData, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}

	var pemPrivateBlock = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pemData,
	}

	encodedData := pem.EncodeToMemory(pemPrivateBlock)

	return encodedData, nil
}
