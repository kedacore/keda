// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

// Package auth provides an abstraction over claims-based security for Azure Event Hub and Service Bus.
package auth

const (
	// CBSTokenTypeJWT is the type of token to be used for JWTs. For example Azure Active Directory tokens.
	CBSTokenTypeJWT TokenType = "jwt"
	// CBSTokenTypeSAS is the type of token to be used for SAS tokens.
	CBSTokenTypeSAS TokenType = "servicebus.windows.net:sastoken"
)

type (
	// TokenType represents types of tokens known for claims-based auth
	TokenType string

	// Token contains all of the information to negotiate authentication
	Token struct {
		// TokenType is the type of CBS token
		TokenType TokenType
		Token     string
		Expiry    string
	}

	// TokenProvider abstracts the fetching of authentication tokens
	TokenProvider interface {
		GetToken(uri string) (*Token, error)
	}
)

// NewToken constructs a new auth token
func NewToken(tokenType TokenType, token, expiry string) *Token {
	return &Token{
		TokenType: tokenType,
		Token:     token,
		Expiry:    expiry,
	}
}
