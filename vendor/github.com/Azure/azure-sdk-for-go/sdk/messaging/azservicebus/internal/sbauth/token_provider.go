// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package sbauth

import (
	"context"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/conn"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/sas"
)

// TokenProvider handles access tokens and expiration calculation for SAS
// keys (via connection strings) or TokenCredentials from Azure Identity.
type TokenProvider struct {
	tokenCred        azcore.TokenCredential
	sasTokenProvider *sas.TokenProvider
}

// NewTokenProvider creates a tokenProvider from azcore.TokenCredential.
func NewTokenProvider(tokenCredential azcore.TokenCredential) *TokenProvider {
	return &TokenProvider{tokenCred: tokenCredential}
}

// NewTokenProviderWithConnectionString creates a tokenProvider from a connection string.
func NewTokenProviderWithConnectionString(parsed *conn.ParsedConn) (*TokenProvider, error) {
	// NOTE: this is the value we've been using since forever. AFAIK, it's arbitrary.
	const defaultTokenExpiry = 2 * time.Hour

	var authOption sas.TokenProviderOption

	if parsed.SAS == "" {
		authOption = sas.TokenProviderWithKey(parsed.KeyName, parsed.Key, defaultTokenExpiry)
	} else {
		authOption = sas.TokenProviderWithSAS(parsed.SAS)
	}

	provider, err := sas.NewTokenProvider(authOption)

	if err != nil {
		return nil, err
	}

	return &TokenProvider{sasTokenProvider: provider}, nil
}

// singleUseTokenProvider allows you to wrap an *auth.Token so it can be used
// with functions that require a TokenProvider, but only actually should get
// a single token (like cbs.NegotiateClaim)
type singleUseTokenProvider auth.Token

// GetToken will return this token.
// This function makes us compatible with auth.TokenProvider.
func (tp *singleUseTokenProvider) GetToken(uri string) (*auth.Token, error) {
	return (*auth.Token)(tp), nil
}

// GetToken will retrieve a new token.
// This function makes us compatible with auth.TokenProvider.
func (tp *TokenProvider) GetToken(uri string) (*auth.Token, error) {
	token, _, err := tp.getTokenImpl(uri)
	return token, err
}

// GetToken returns a token (that is compatible as an auth.TokenProvider) and
// the calculated time when you should renew your token.
func (tp *TokenProvider) GetTokenAsTokenProvider(uri string) (*singleUseTokenProvider, time.Time, error) {
	token, renewAt, err := tp.getTokenImpl(uri)

	if err != nil {
		return nil, time.Time{}, err
	}

	return (*singleUseTokenProvider)(token), renewAt, nil
}

func (tp *TokenProvider) getTokenImpl(uri string) (*auth.Token, time.Time, error) {
	if tp.sasTokenProvider != nil {
		return tp.getSASToken(uri)
	} else {
		return tp.getAZCoreToken()
	}
}

func (tpa *TokenProvider) getAZCoreToken() (*auth.Token, time.Time, error) {
	// not sure if URI plays in here.
	accessToken, err := tpa.tokenCred.GetToken(context.TODO(), policy.TokenRequestOptions{
		Scopes: []string{
			"https://servicebus.azure.net//.default",
		},
	})

	if err != nil {
		return nil, time.Time{}, err
	}

	authToken := &auth.Token{
		TokenType: auth.CBSTokenTypeJWT,
		Token:     accessToken.Token,
		Expiry:    strconv.FormatInt(accessToken.ExpiresOn.Unix(), 10),
	}

	return authToken,
		accessToken.ExpiresOn,
		nil
}

func (tpa *TokenProvider) getSASToken(uri string) (*auth.Token, time.Time, error) {
	authToken, err := tpa.sasTokenProvider.GetToken(uri)

	if err != nil {
		return nil, time.Time{}, err
	}

	return authToken,
		// expiration is hardcoded for SAS tokens
		time.Now().Add(time.Minute * 15),
		nil
}
