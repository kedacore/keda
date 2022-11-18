// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

// Package sas provides SAS token functionality which implements TokenProvider from package auth for use with Azure
// Event Hubs and Service Bus.
package sas

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/conn"
)

type (
	// Signer provides SAS token generation for use in Service Bus and Event Hub
	Signer struct {
		KeyName string
		Key     string

		// getNow is stubabble for unit tests and is just an alias for time.Now()
		getNow func() time.Time
	}

	// TokenProvider is a SAS claims-based security token provider
	TokenProvider struct {
		// expiryDuration is only used when we're generating SAS tokens. It gets used
		// to calculate the expiration timestamp for a token. Pre-computed SAS tokens
		// passed in TokenProviderWithSAS() are not affected.
		expiryDuration time.Duration

		signer *Signer

		// sas is a precomputed SAS token. This implies that the caller has some other
		// method for generating tokens.
		sas string
	}

	// TokenProviderOption provides configuration options for SAS Token Providers
	TokenProviderOption func(*TokenProvider) error
)

// TokenProviderWithKey configures a SAS TokenProvider to use the given key name and key (secret) for signing
func TokenProviderWithKey(keyName, key string, expiryDuration time.Duration) TokenProviderOption {
	return func(provider *TokenProvider) error {

		if expiryDuration == 0 {
			expiryDuration = 2 * time.Hour
		}

		provider.expiryDuration = expiryDuration
		provider.signer = NewSigner(keyName, key)
		return nil
	}
}

// TokenProviderWithSAS configures the token provider with a pre-created SharedAccessSignature.
// auth.Token's coming back from this TokenProvider instance will always have '0' as the expiration
// date.
func TokenProviderWithSAS(sas string) TokenProviderOption {
	return func(provider *TokenProvider) error {
		provider.sas = sas
		return nil
	}
}

// NewTokenProvider builds a SAS claims-based security token provider
func NewTokenProvider(opts ...TokenProviderOption) (*TokenProvider, error) {
	provider := new(TokenProvider)

	for _, opt := range opts {
		err := opt(provider)
		if err != nil {
			return nil, err
		}
	}
	return provider, nil
}

// GetToken gets a CBS SAS token
func (t *TokenProvider) GetToken(audience string) (*auth.Token, error) {
	if t.sas != "" {
		// the expiration date doesn't matter here so we'll just set it 0.
		return auth.NewToken(auth.CBSTokenTypeSAS, t.sas, "0"), nil
	}

	signature, expiry, err := t.signer.SignWithDuration(audience, t.expiryDuration)

	if err != nil {
		return nil, err
	}

	return auth.NewToken(auth.CBSTokenTypeSAS, signature, expiry), nil
}

// NewSigner builds a new SAS signer for use in generation Service Bus and Event Hub SAS tokens
func NewSigner(keyName, key string) *Signer {
	return &Signer{
		KeyName: keyName,
		Key:     key,

		getNow: time.Now,
	}
}

// SignWithDuration signs a given for a period of time from now
func (s *Signer) SignWithDuration(uri string, interval time.Duration) (signature, expiry string, err error) {
	expiry = signatureExpiry(s.getNow().UTC(), interval)
	sig, err := s.SignWithExpiry(uri, expiry)

	if err != nil {
		return "", "", err
	}

	return sig, expiry, nil
}

// SignWithExpiry signs a given uri with a given expiry string
func (s *Signer) SignWithExpiry(uri, expiry string) (string, error) {
	audience := strings.ToLower(url.QueryEscape(uri))
	sts := stringToSign(audience, expiry)
	sig, err := s.signString(sts)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("SharedAccessSignature sr=%s&sig=%s&se=%s&skn=%s", audience, sig, expiry, s.KeyName), nil
}

// CreateConnectionStringWithSharedAccessSignature generates a new connection string with
// an embedded SharedAccessSignature and expiration.
// Ex: Endpoint=sb://<sb>.servicebus.windows.net;SharedAccessSignature=SharedAccessSignature sr=<sb>.servicebus.windows.net&sig=<base64-sig>&se=<expiry>&skn=<keyname>"
func CreateConnectionStringWithSAS(connectionString string, duration time.Duration) (string, error) {
	parsed, err := conn.ParsedConnectionFromStr(connectionString)

	if err != nil {
		return "", err
	}

	signer := NewSigner(parsed.KeyName, parsed.Key)

	sig, _, err := signer.SignWithDuration(parsed.Namespace, duration)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Endpoint=sb://%s;SharedAccessSignature=%s", parsed.Namespace, sig), nil
}

func signatureExpiry(from time.Time, interval time.Duration) string {
	t := from.Add(interval).Round(time.Second).Unix()
	return strconv.FormatInt(t, 10)
}

func stringToSign(uri, expiry string) string {
	return uri + "\n" + expiry
}

func (s *Signer) signString(str string) (string, error) {
	h := hmac.New(sha256.New, []byte(s.Key))
	_, err := h.Write([]byte(str))

	if err != nil {
		return "", err
	}

	encodedSig := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return url.QueryEscape(encodedSig), nil
}
