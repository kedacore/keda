package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

// NewTLSConfig returns a *tls.Config using the given ceClient cert, ceClient key,
// and CA certificate. If none are appropriate, a nil *tls.Config is returned.
func NewTLSConfig(clientCert, clientKey, caCert string) (*tls.Config, error) {
	valid := false

	config := &tls.Config{}

	if clientCert != "" && clientKey != "" {
		cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
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
		config.InsecureSkipVerify = true
		valid = true
	}

	if !valid {
		config = nil
	}

	return config, nil
}
