// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package sbauth

import (
	"context"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/sas"
)

// TokenProvider handles access tokens and expiration calculation for SAS
// keys (via connection strings) or TokenCredentials from Azure Identity.
type TokenProvider struct {
	tokenCred        azcore.TokenCredential
	sasTokenProvider *sas.TokenProvider

	// InsecureDisableTLS disables TLS. This is only used if the user is connecting to localhost
	// and is using an emulator connection string. See [ConnectionStringProperties.Emulator] for
	// details.
	InsecureDisableTLS bool
}

// NewTokenProvider creates a tokenProvider from azcore.TokenCredential.
func NewTokenProvider(tokenCredential azcore.TokenCredential) *TokenProvider {
	return &TokenProvider{tokenCred: tokenCredential}
}

// NewTokenProviderWithConnectionString creates a tokenProvider from a connection string.
func NewTokenProviderWithConnectionString(props exported.ConnectionStringProperties) (*TokenProvider, error) {
	// NOTE: this is the value we've been using since forever. AFAIK, it's arbitrary.
	const defaultTokenExpiry = 2 * time.Hour

	var authOption sas.TokenProviderOption

	if props.SharedAccessSignature == nil {
		authOption = sas.TokenProviderWithKey(*props.SharedAccessKeyName, *props.SharedAccessKey, defaultTokenExpiry)
	} else {
		authOption = sas.TokenProviderWithSAS(*props.SharedAccessSignature)
	}

	provider, err := sas.NewTokenProvider(authOption)

	if err != nil {
		return nil, err
	}

	return &TokenProvider{sasTokenProvider: provider, InsecureDisableTLS: props.Emulator}, nil
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
			"https://eventhubs.azure.net//.default",
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

	// we can ignore the error here since we did the string-izing of the time
	// in the first place.
	var expiryTime time.Time

	if authToken.Expiry != "0" {
		// TODO: I'd like to just use the actual Expiry time we generated
		// Filed here https://github.com/Azure/azure-sdk-for-go/issues/20468
		expiryTime = time.Now().Add(time.Minute * 15)
	}

	return authToken,
		expiryTime,
		nil
}
