package util

import (
	"crypto/x509"
	"strings"
	"testing"
)

func testingKey(s string) string { return strings.ReplaceAll(s, "TESTING KEY", "PRIVATE KEY") }

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

func TestNewTLSConfig_WithoutPassword(t *testing.T) {

	testCases := []struct {
		name   string
		cert   string
		key    string
		issuer string
	}{
		{
			name:   "rsaCert",
			cert:   rsaCertPEM,
			key:    rsaKeyPEM,
			issuer: "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
		},
		{
			name:   "Cert",
			cert:   rsaCertPEM,
			key:    keyPEM,
			issuer: "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			config, err := NewTLSConfig(test.cert, test.key, "")
			if err != nil {
				t.Errorf("Should have no error %s", err)
			}
			if config == nil {
				t.Errorf("Config should not be nil")
			}
			cert, err := x509.ParseCertificate(config.Certificates[0].Certificate[0])
			if err != nil {
				t.Errorf("Bad certificate")
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
	}{
		{
			name:     "rsaCert",
			cert:     rsaCertPEM,
			key:      encryptedRsaKeyPEM,
			password: "keypass",
			issuer:   "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
		},
		{
			name:     "Cert",
			cert:     rsaCertPEM,
			key:      encryptedKeyPEM,
			password: "keypass",
			issuer:   "O=Internet Widgits Pty Ltd,ST=Some-State,C=AU",
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			config, err := NewTLSConfigWithPassword(test.cert, test.key, test.password, "")
			if err != nil {
				t.Errorf("Should have no error: %s", err)
			}
			if config == nil {
				t.Errorf("Config should not be nil")
			}
			cert, err := x509.ParseCertificate(config.Certificates[0].Certificate[0])
			if err != nil {
				t.Errorf("Bad certificate")
			}

			if cert.Issuer.String() != test.issuer {
				t.Errorf("Expected Issuer %s but got %s\n", test.issuer, cert.Issuer.String())
			}
		})
	}
}
