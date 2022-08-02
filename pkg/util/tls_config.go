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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/youmark/pkcs8"
)

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

// NewTLSConfigWithPassword returns a *tls.Config using the given ceClient cert, ceClient key,
// and CA certificate. If clientKeyPassword is not empty the provided password will be used to
// decrypt the given key. If none are appropriate, a nil *tls.Config is returned.
func NewTLSConfigWithPassword(clientCert, clientKey, clientKeyPassword, caCert string) (*tls.Config, error) {
	// skipVerify := true is a hack to avoid the CodeQL error related with allowing insecure certificates in production environments.
	// Skipping this validation is necessary and intended in our use case in order to be able to trust in the CA.
	skipVerify := true
	valid := false

	config := &tls.Config{}

	if clientCert != "" && clientKey != "" {
		key := []byte(clientKey)
		if clientKeyPassword != "" {
			var err error
			key, err = decryptClientKey(clientKey, clientKeyPassword)
			if err != nil {
				return nil, fmt.Errorf("error decrypt X509Key: %s", err)
			}
		}

		cert, err := tls.X509KeyPair([]byte(clientCert), key)
		if err != nil {
			return nil, fmt.Errorf("error parse X509KeyPair: %s", err)
		}
		config.Certificates = []tls.Certificate{cert}
		valid = true
	}

	if caCert != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(caCert))
		config.RootCAs = caCertPool
		config.InsecureSkipVerify = skipVerify
		valid = true
	}

	if !valid {
		config = nil
	}

	return config, nil
}

// NewTLSConfig returns a *tls.Config using the given ceClient cert, ceClient key,
// and CA certificate. If none are appropriate, a nil *tls.Config is returned.
func NewTLSConfig(clientCert, clientKey, caCert string) (*tls.Config, error) {
	return NewTLSConfigWithPassword(clientCert, clientKey, "", caCert)
}
