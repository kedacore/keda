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
	"os"
	"strings"

	"github.com/youmark/pkcs8"
	ctrl "sigs.k8s.io/controller-runtime"
)

var minTLSVersion uint16
var tlsCipherList []uint16
var serviceMinTLSVersion uint16
var serviceTLSCipherList []uint16

func init() {
	var err error

	if minTLSVersion, err = ParseTLSVersion(os.Getenv("KEDA_HTTP_MIN_TLS_VERSION"), tls.VersionTLS12); err != nil {
		ctrl.Log.WithName("tls_setup").Info("Error parsing environment variable KEDA_HTTP_MIN_TLS_VERSION", "value", os.Getenv("KEDA_HTTP_MIN_TLS_VERSION"), "error", err.Error())
	}
	tlsCipherList = ParseTLSCipherList(os.Getenv("KEDA_HTTP_TLS_CIPHER_LIST"))

	if envvar, found := os.LookupEnv("KEDA_SERVICE_MIN_TLS_VERSION"); found {
		if serviceMinTLSVersion, err = ParseTLSVersion(envvar, tls.VersionTLS13); err != nil {
			ctrl.Log.WithName("tls_setup").Info("Error parsing environment variable KEDA_HTTP_MIN_TLS_VERSION", "value", os.Getenv("KEDA_SERVICE_MIN_TLS_VERSION"), "error", err.Error())
		}
	} else {
		serviceMinTLSVersion = minTLSVersion // fall back since the old behavior was for the webhook to use KEDA_HTTP_MIN_TLS_VERSION
	}

	if envvar, found := os.LookupEnv("KEDA_SERVICE_TLS_CIPHER_LIST"); found {
		serviceTLSCipherList = ParseTLSCipherList(envvar)
	} else {
		serviceTLSCipherList = tlsCipherList
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

// CreateTLSClientConfig returns a new TLS Config
// unsafeSsl parameter allows to avoid tls cert validation if it's required
func CreateTLSClientConfig(unsafeSsl bool) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: unsafeSsl,
		RootCAs:            getRootCAs(),
		MinVersion:         GetMinTLSVersion(),
		CipherSuites:       GetTLSCipherList(),
	}
}

// GetMinTLSVersion return the minTLSVersion based on configurations
func GetMinTLSVersion() uint16 {
	return minTLSVersion
}

// GetTLSCipherList returns the TLS cipher list based on configurations
func GetTLSCipherList() []uint16 {
	return tlsCipherList
}

// GetServiceMinTLSVersion return the minimum TLS version that KEDA services are configured to use
func GetServiceMinTLSVersion() uint16 {
	return serviceMinTLSVersion
}

// GetServiceTLSCipherList return the TLS ciphersuites that KEDA services are configured to use
func GetServiceTLSCipherList() []uint16 {
	return serviceTLSCipherList
}

// ParseTLSCipherList parses a colon or comma-separated list of TLS cipher suite names
// (as returned by crypto/tls CipherSuites()) into a slice of cipher suite IDs.
// Unknown names are logged. Returns nil if no valid ciphers are found.
func ParseTLSCipherList(ciphers string) []uint16 {
	reverseCipherMap := make(map[string]uint16)
	for _, c := range tls.CipherSuites() {
		reverseCipherMap[c.Name] = c.ID
	}
	var ciphersuites []uint16
	for c := range strings.SplitSeq(strings.ReplaceAll(ciphers, ",", ":"), ":") {
		c = strings.TrimSpace(c)
		if id, ok := reverseCipherMap[c]; ok {
			ciphersuites = append(ciphersuites, id)
		} else {
			ctrl.Log.WithName("tls_setup").Info("Unrecognized TLS ciphersuite name while parsing list of ciphersuites", "value", c)
		}
	}
	if len(ciphersuites) == 0 {
		return nil
	}
	return ciphersuites
}

// ParseTLSVersion converts a TLS version string to a TLS version value
func ParseTLSVersion(version string, defaultVersion uint16) (uint16, error) {
	var ver uint16
	switch version {
	case "":
		ver = defaultVersion
	case "TLS10":
		ver = tls.VersionTLS10
	case "TLS11":
		ver = tls.VersionTLS11
	case "TLS12":
		ver = tls.VersionTLS12
	case "TLS13":
		ver = tls.VersionTLS13
	default:
		return defaultVersion, fmt.Errorf("%s is not a valid value, using `%s`. Allowed values are: `TLS13`,`TLS12`,`TLS11`,`TLS10`", version, strings.ReplaceAll(strings.ReplaceAll(tls.VersionName(defaultVersion), " ", ""), ".", ""))
	}

	return ver, nil
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
