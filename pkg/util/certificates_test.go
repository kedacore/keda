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

package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	certCommonName  = "test-cert"
	certCommonName2 = "test-cert2"
)

func TestCustomCAsAreRegistered(t *testing.T) {
	customCAPath, err := os.MkdirTemp("", "test-ca-certdir-")
	require.NoErrorf(t, err, "error creating temporary certs dir - %s", err)
	defer os.RemoveAll(customCAPath)

	generateCA(t, certCommonName, customCAPath)

	SetCACertDirs([]string{customCAPath})

	rootCAs := getRootCAs()
	//nolint:staticcheck // func (s *CertPool) Subjects was deprecated if s was returned by SystemCertPool, Subjects
	subjects := rootCAs.Subjects()
	var rdnSequence pkix.RDNSequence
	_, err = asn1.Unmarshal(subjects[len(subjects)-1], &rdnSequence)
	if err != nil {
		t.Fatal("could not unmarshal der formatted subject")
	}
	var name pkix.Name
	name.FillFromRDNSequence(&rdnSequence)

	assert.Equal(t, certCommonName, name.CommonName, "certificate not found")
}

func TestMultipleCustomCADirs(t *testing.T) {
	customCAPath, err := os.MkdirTemp("", "test-ca-certdir-")
	require.NoErrorf(t, err, "error creating temporary certs dir - %s", err)
	defer os.RemoveAll(customCAPath)
	customCAPath2, err := os.MkdirTemp("", "test-ca-certdir2-")
	require.NoErrorf(t, err, "error creating temporary certs dir - %s", err)
	defer os.RemoveAll(customCAPath2)

	generateCA(t, certCommonName, customCAPath)
	generateCA(t, certCommonName2, customCAPath2)
	SetCACertDirs([]string{customCAPath, customCAPath2})

	rootCAs := getRootCAs()
	//nolint:staticcheck // func (s *CertPool) Subjects was deprecated if s was returned by SystemCertPool, Subjects
	subjects := rootCAs.Subjects()
	var rdnSequence pkix.RDNSequence
	for i := 0; i < 2; i++ {
		_, err = asn1.Unmarshal(subjects[len(subjects)-1-i], &rdnSequence)
		if err != nil {
			t.Fatal("could not unmarshal der formatted subject")
		}
		var name pkix.Name
		name.FillFromRDNSequence(&rdnSequence)

		if i == 1 {
			assert.Equal(t, certCommonName, name.CommonName, "certificate not found")
		} else {
			assert.Equal(t, certCommonName2, name.CommonName, "certificate not found")
		}
	}
}

func generateCA(t *testing.T, cn, dir string) {
	err := os.MkdirAll(dir, os.ModePerm)
	caCrtPath := path.Join(dir, "ca.crt")
	require.NoErrorf(t, err, "error generating the custom ca folder - %s", err)

	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
			CommonName:    cn,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoErrorf(t, err, "error generating custom CA key - %s", err)

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	require.NoErrorf(t, err, "error generating custom CA - %s", err)

	// pem encode
	crtFile, err := os.OpenFile(caCrtPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	require.NoErrorf(t, err, "error opening custom CA file - %s", err)
	err = pem.Encode(crtFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	require.NoErrorf(t, err, "error opening custom CA file - %s", err)
}
