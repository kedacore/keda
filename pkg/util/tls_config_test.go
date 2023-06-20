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
	"crypto/tls"
	"crypto/x509"
	"os"
	"strings"
	"testing"
)

var randomCACert = `-----BEGIN CERTIFICATE-----
MIIDYzCCAkugAwIBAgIUHq1Lf66TAFwFxelktPk6jv3TOlkwDQYJKoZIhvcNAQEL
BQAwQTEaMBgGA1UEAwwRdW5pdHRlc3Qua2VkYS5jb20xCzAJBgNVBAYTAlVTMRYw
FAYDVQQHDA1TYW4gRnJhbnNpc2NvMB4XDTIzMDIwODE0MTgwMFoXDTI0MDEzMDE0
MTgwMFowQTEaMBgGA1UEAwwRdW5pdHRlc3Qua2VkYS5jb20xCzAJBgNVBAYTAlVT
MRYwFAYDVQQHDA1TYW4gRnJhbnNpc2NvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAvWZ1I7NQOlbiz0GR2XeO3qHehpVJeycRcbErUQmeNp3HeQRVvx2j
ZaNV2sIKn2l3BKW9jVymk3uR1lZ7fXOLD5h5EvrBb7RGxSbKMbK4jCqFHbN4p3Gv
1rz73DiCKXgisFY2lLykGFLgaXB5pALtVnrxKILS4OwndrjEudS80RGh1jP9w+Pt
7q98yM3r5qshZ56E4Qn7hq+Lsd7l6Os+eVVtBDAHbDNEiLnQfjCBBfg/3qhvqqd8
ALm+ZNEULMMc8kI165jassJMRsVvkIKOjMiTjsGSsZS6RovLf8FIEAxCtSJvbU9g
qY+WO5/C9xRlFYXUQsx7OGd2iLnNtZ+JiwIDAQABo1MwUTAdBgNVHQ4EFgQUaxIS
bJyuR5YQhO4Rh8JDkdEmlvAwHwYDVR0jBBgwFoAUaxISbJyuR5YQhO4Rh8JDkdEm
lvAwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAFiixbxuMqjIK
fRR9cxFV+LvFr6BL7zJViVK5opr+wSLKpsF7hsZV5KvdNxFslby3tVWsm0aiuhTv
BmmdGIF2cNhq+7egihRddCCTOfqek4980O1TnVstqI/clYMxkftrEO5T85k+LNts
cQbH1lUEipv8/TuwY/bdhuV/EjuQHiBBh9XyegZU3RgTORnDbfkGRnrMWbHcschP
PFwwb1T9BmyQShLXzSpJdgx+NuR+CXSu8OXMgs0P99Vle3piABDr0Qd5WPCZJHcH
syU5YTDyvkFUjf7yV0KYgsoZgTCHAuP1oiaFY6xwnQ1stpPz1/LcySMEnsXoJNVt
MdpMcBrdUw==
-----END CERTIFICATE-----
`

var rsaCertPEM = `-----BEGIN CERTIFICATE-----
MIIB0zCCAX2gAwIBAgIJAI/M7BYjwB+uMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTIwOTEyMjE1MjAyWhcNMTUwOTEyMjE1MjAyWjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBANLJ
hPHhITqQbPklG3ibCVxwGMRfp/v4XqhfdQHdcVfHap6NQ5Wok/4xIA+ui35/MmNa
rtNuC+BdZ1tMuVCPFZcCAwEAAaNQME4wHQYDVR0OBBYEFJvKs8RfJaXTH08W+SGv
zQyKn0H8MB8GA1UdIwQYMBaAFJvKs8RfJaXTH08W+SGvzQyKn0H8MAwGA1UdEwQF
MAMBAf8wDQYJKoZIhvcNAQEFBQADQQBJlffJHybjDGxRMqaRmDhX0+6v02TUKZsW
r5QuVbpQhH6u+0UgcW0jp9QwpxoPTLTWGXEWBBBurxFwiCBhkQ+V
-----END CERTIFICATE-----
`

var rsaKeyPEM = testingKey(`-----BEGIN RSA TESTING KEY-----
MIIBOwIBAAJBANLJhPHhITqQbPklG3ibCVxwGMRfp/v4XqhfdQHdcVfHap6NQ5Wo
k/4xIA+ui35/MmNartNuC+BdZ1tMuVCPFZcCAwEAAQJAEJ2N+zsR0Xn8/Q6twa4G
6OB1M1WO+k+ztnX/1SvNeWu8D6GImtupLTYgjZcHufykj09jiHmjHx8u8ZZB/o1N
MQIhAPW+eyZo7ay3lMz1V01WVjNKK9QSn1MJlb06h/LuYv9FAiEA25WPedKgVyCW
SmUwbPw8fnTcpqDWE3yTO3vKcebqMSsCIBF3UmVue8YU3jybC3NxuXq3wNm34R8T
xVLHwDXh/6NJAiEAl2oHGGLz64BuAfjKrqwz7qMYr9HCLIe/YsoWq/olzScCIQDi
D2lWusoe2/nEqfDVVWGWlyJ7yOmqaVm/iNUN9B2N2g==
-----END RSA TESTING KEY-----
`)

var encryptedRsaKeyPEM = testingKey(`-----BEGIN ENCRYPTED TESTING KEY-----
MIIBvTBXBgkqhkiG9w0BBQ0wSjApBgkqhkiG9w0BBQwwHAQIuJju3iFn018CAggA
MAwGCCqGSIb3DQIJBQAwHQYJYIZIAWUDBAEqBBA7gzv+Ry86tAxCLBS3oQ+aBIIB
YGJsCG9AeftP2xcWVwGZV/R1s1qCM2pI3Zg5j+veNnvnAma6UX+bVkHIIQBDQxXs
pqz1gB0DD6VjE71icUiOZD/LhnMmpo76Ghwdf3RLF+zRz4He4vzAaYbJGKBYBL1Y
RC0v4iDyMD8d00DxLwr+lXjyxLTTVB5xtZtCPFPerpY6AiRCwpRlw8Myhhmcg0Q+
qKZ1udRbug8RzQNMFBtntGxlrib8Ti+cDy5YW/VxK0ma9TXWprolIZpjwOWgHMQK
GYtAHwRN3tl7oa7D+zfZ0Gxohw6V3MjGDXkeCj0i92SA8q5vJvEHuRWklIpXI+dc
zBYCjyoY3x6cNS2u6KtrlOFj4+KIITmJLrarnZ0PdtsNuUjRHhHH8YJFKvEijd9K
46Ayrm8Lm4yhWzgNjjHWabdejK9fXI63OOAsySHgAd+re22/daqf3tTYFSUOR4Y6
JR68ifUcDhEs2/af5oAaJsw=
-----END ENCRYPTED TESTING KEY-----
`)

// keyPEM is the same as rsaKeyPEM, but declares itself as just
// "PRIVATE KEY", not "RSA PRIVATE KEY".  https://golang.org/issue/4477
var keyPEM = testingKey(`-----BEGIN TESTING KEY-----
MIIBOwIBAAJBANLJhPHhITqQbPklG3ibCVxwGMRfp/v4XqhfdQHdcVfHap6NQ5Wo
k/4xIA+ui35/MmNartNuC+BdZ1tMuVCPFZcCAwEAAQJAEJ2N+zsR0Xn8/Q6twa4G
6OB1M1WO+k+ztnX/1SvNeWu8D6GImtupLTYgjZcHufykj09jiHmjHx8u8ZZB/o1N
MQIhAPW+eyZo7ay3lMz1V01WVjNKK9QSn1MJlb06h/LuYv9FAiEA25WPedKgVyCW
SmUwbPw8fnTcpqDWE3yTO3vKcebqMSsCIBF3UmVue8YU3jybC3NxuXq3wNm34R8T
xVLHwDXh/6NJAiEAl2oHGGLz64BuAfjKrqwz7qMYr9HCLIe/YsoWq/olzScCIQDi
D2lWusoe2/nEqfDVVWGWlyJ7yOmqaVm/iNUN9B2N2g==
-----END TESTING KEY-----
`)

var encryptedKeyPEM = testingKey(`-----BEGIN TESTING KEY-----
MIIBvTBXBgkqhkiG9w0BBQ0wSjApBgkqhkiG9w0BBQwwHAQIuJju3iFn018CAggA
MAwGCCqGSIb3DQIJBQAwHQYJYIZIAWUDBAEqBBA7gzv+Ry86tAxCLBS3oQ+aBIIB
YGJsCG9AeftP2xcWVwGZV/R1s1qCM2pI3Zg5j+veNnvnAma6UX+bVkHIIQBDQxXs
pqz1gB0DD6VjE71icUiOZD/LhnMmpo76Ghwdf3RLF+zRz4He4vzAaYbJGKBYBL1Y
RC0v4iDyMD8d00DxLwr+lXjyxLTTVB5xtZtCPFPerpY6AiRCwpRlw8Myhhmcg0Q+
qKZ1udRbug8RzQNMFBtntGxlrib8Ti+cDy5YW/VxK0ma9TXWprolIZpjwOWgHMQK
GYtAHwRN3tl7oa7D+zfZ0Gxohw6V3MjGDXkeCj0i92SA8q5vJvEHuRWklIpXI+dc
zBYCjyoY3x6cNS2u6KtrlOFj4+KIITmJLrarnZ0PdtsNuUjRHhHH8YJFKvEijd9K
46Ayrm8Lm4yhWzgNjjHWabdejK9fXI63OOAsySHgAd+re22/daqf3tTYFSUOR4Y6
JR68ifUcDhEs2/af5oAaJsw=
-----END TESTING KEY-----
`)

func testingKey(s string) string { return strings.ReplaceAll(s, "TESTING KEY", "PRIVATE KEY") }

func TestNewTLSConfig_WithoutPassword(t *testing.T) {
	testCases := []struct {
		name   string
		cert   string
		key    string
		issuer string
		CACert string
	}{
		{
			name:   "rsaCert_WithCACert",
			cert:   rsaCertPEM,
			key:    rsaKeyPEM,
			issuer: "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
			CACert: randomCACert,
		},
		{
			name:   "Cert_WithCACert",
			cert:   rsaCertPEM,
			key:    keyPEM,
			issuer: "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
			CACert: randomCACert,
		},
		{
			name:   "rsaCert_WithoutCACert",
			cert:   rsaCertPEM,
			key:    rsaKeyPEM,
			issuer: "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
			CACert: "",
		},
		{
			name:   "Cert_WithoutCACert",
			cert:   rsaCertPEM,
			key:    keyPEM,
			issuer: "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
			CACert: "",
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			config, err := NewTLSConfig(test.cert, test.key, test.CACert, false)
			if err != nil {
				t.Errorf("Should have no error %s", err)
			}
			cert, err := x509.ParseCertificate(config.Certificates[0].Certificate[0])
			if err != nil {
				t.Errorf("Bad certificate")
			}

			if test.CACert != "" {
				caCertPool := getRootCAs()
				caCertPool.AppendCertsFromPEM([]byte(randomCACert))
				if !config.RootCAs.Equal(caCertPool) {
					t.Errorf("TLS config return different CA cert")
				}
			}

			if cert.Issuer.String() != test.issuer {
				t.Errorf("Expected Issuer %s but got %s\n", test.issuer, cert.Issuer.String())
			}
		})
	}
}
func TestNewTLSConfig_WithPassword(t *testing.T) {
	testCases := []struct {
		name     string
		cert     string
		key      string
		password string
		issuer   string
		CACert   string
		isError  bool
	}{
		{
			name:     "rsaCert_WithCACert",
			cert:     rsaCertPEM,
			key:      encryptedRsaKeyPEM,
			password: "keypass",
			issuer:   "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
			CACert:   randomCACert,
			isError:  false,
		},
		{
			name:     "Cert_WithCACert",
			cert:     rsaCertPEM,
			key:      encryptedKeyPEM,
			password: "keypass",
			issuer:   "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
			CACert:   randomCACert,
			isError:  false,
		},
		{
			name:     "rsaCert_WithoutCACert",
			cert:     rsaCertPEM,
			key:      encryptedRsaKeyPEM,
			password: "keypass",
			issuer:   "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
			CACert:   "",
			isError:  false,
		},
		{
			name:     "Cert_WithoutCACert",
			cert:     rsaCertPEM,
			key:      encryptedKeyPEM,
			password: "keypass",
			issuer:   "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
			CACert:   "",
			isError:  false,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			config, err := NewTLSConfigWithPassword(test.cert, test.key, test.password, test.CACert, false)
			switch {
			case err != nil && !test.isError:
				t.Errorf("Expected success but got error: %s", err)
			case test.isError && err == nil:
				t.Errorf("Expect error but got success")
			case err == nil:
				cert, err := x509.ParseCertificate(config.Certificates[0].Certificate[0])
				if err != nil {
					t.Errorf("Bad certificate")
				}

				if test.CACert != "" {
					caCertPool := getRootCAs()
					caCertPool.AppendCertsFromPEM([]byte(randomCACert))
					if !config.RootCAs.Equal(caCertPool) {
						t.Errorf("TLS config return different CA cert")
					}
				}
				if cert.Issuer.String() != test.issuer {
					t.Errorf("Expected Issuer %s but got %s\n", test.issuer, cert.Issuer.String())
				}
			}
		})
	}
}

type minTLSVersionTestData struct {
	envSet          bool
	envValue        string
	expectedVersion uint16
}

var minTLSVersionTestDatas = []minTLSVersionTestData{
	{
		envSet:          true,
		envValue:        "TLS10",
		expectedVersion: tls.VersionTLS10,
	},
	{
		envSet:          true,
		envValue:        "TLS11",
		expectedVersion: tls.VersionTLS11,
	},
	{
		envSet:          true,
		envValue:        "TLS12",
		expectedVersion: tls.VersionTLS12,
	},
	{
		envSet:          true,
		envValue:        "TLS13",
		expectedVersion: tls.VersionTLS13,
	},
	{
		envSet:          false,
		expectedVersion: tls.VersionTLS12,
	},
}

func TestResolveMinTLSVersion(t *testing.T) {
	defer os.Unsetenv("KEDA_HTTP_MIN_TLS_VERSION")
	for _, testData := range minTLSVersionTestDatas {
		os.Unsetenv("KEDA_HTTP_MIN_TLS_VERSION")
		if testData.envSet {
			os.Setenv("KEDA_HTTP_MIN_TLS_VERSION", testData.envValue)
		}
		minVersion, _ := initMinTLSVersion()

		if testData.expectedVersion != minVersion {
			t.Error("Failed to resolve minTLSVersion correctly", "wants", testData.expectedVersion, "got", minVersion)
		}
	}
}
