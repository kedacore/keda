// Copyright 2020 Google LLC.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package idtoken

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	newidtoken "cloud.google.com/go/auth/credentials/idtoken"
	"cloud.google.com/go/auth/oauth2adapt"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/internal"
	"google.golang.org/api/internal/credentialstype"
	"google.golang.org/api/option"
	"google.golang.org/api/option/internaloption"
	htransport "google.golang.org/api/transport/http"
)

// ClientOption is aliased so relevant options are easily found in the docs.

// ClientOption is for configuring a Google API client or transport.
type ClientOption = option.ClientOption

// CredentialsType specifies the type of JSON credentials being provided
// to a loading function such as [WithAuthCredentialsFile] or
// [WithAuthCredentialsJSON].
type CredentialsType = credentialstype.CredType

const (
	// ServiceAccount represents a service account file type.
	ServiceAccount = credentialstype.ServiceAccount
	// AuthorizedUser represents an authorized user credentials file type.
	AuthorizedUser = credentialstype.AuthorizedUser
	// ImpersonatedServiceAccount represents an impersonated service account file type.
	//
	// IMPORTANT:
	// This credential type does not validate the credential configuration. A security
	// risk occurs when a credential configuration configured with malicious urls
	// is used.
	// You should validate credential configurations provided by untrusted sources.
	// See [Security requirements when using credential configurations from an external
	// source] https://cloud.google.com/docs/authentication/external/externally-sourced-credentials
	// for more details.
	ImpersonatedServiceAccount = credentialstype.ImpersonatedServiceAccount
	// ExternalAccount represents an external account file type.
	//
	// IMPORTANT:
	// This credential type does not validate the credential configuration. A security
	// risk occurs when a credential configuration configured with malicious urls
	// is used.
	// You should validate credential configurations provided by untrusted sources.
	// See [Security requirements when using credential configurations from an external
	// source] https://cloud.google.com/docs/authentication/external/externally-sourced-credentials
	// for more details.
	ExternalAccount = credentialstype.ExternalAccount
)

// NewClient creates a HTTP Client that automatically adds an ID token to each
// request via an Authorization header. The token will have the audience
// provided and be configured with the supplied options. The parameter audience
// may not be empty.
func NewClient(ctx context.Context, audience string, opts ...ClientOption) (*http.Client, error) {
	var ds internal.DialSettings
	for _, opt := range opts {
		opt.Apply(&ds)
	}
	if err := ds.Validate(); err != nil {
		return nil, err
	}
	if ds.NoAuth {
		return nil, fmt.Errorf("idtoken: option.WithoutAuthentication not supported")
	}
	if ds.APIKey != "" {
		return nil, fmt.Errorf("idtoken: option.WithAPIKey not supported")
	}
	if ds.TokenSource != nil {
		return nil, fmt.Errorf("idtoken: option.WithTokenSource not supported")
	}

	ts, err := NewTokenSource(ctx, audience, opts...)
	if err != nil {
		return nil, err
	}
	// Skip DialSettings validation so added TokenSource will not conflict with user
	// provided credentials.
	opts = append(opts, option.WithTokenSource(ts), internaloption.SkipDialSettingsValidation())
	defaultTrans := http.DefaultTransport
	if trans, ok := defaultTrans.(*http.Transport); ok {
		defaultTrans = trans.Clone()
		defaultTrans.(*http.Transport).MaxIdleConnsPerHost = 100
	}
	t, err := htransport.NewTransport(ctx, defaultTrans, opts...)
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: t}, nil
}

// NewTokenSource creates a TokenSource that returns ID tokens with the audience
// provided and configured with the supplied options. The parameter audience may
// not be empty.
func NewTokenSource(ctx context.Context, audience string, opts ...ClientOption) (oauth2.TokenSource, error) {
	if audience == "" {
		return nil, fmt.Errorf("idtoken: must supply a non-empty audience")
	}
	var ds internal.DialSettings
	for _, opt := range opts {
		opt.Apply(&ds)
	}
	if err := ds.Validate(); err != nil {
		return nil, err
	}
	if ds.TokenSource != nil {
		return nil, fmt.Errorf("idtoken: option.WithTokenSource not supported")
	}
	if ds.ImpersonationConfig != nil {
		return nil, fmt.Errorf("idtoken: option.WithImpersonatedCredentials not supported")
	}
	if ds.IsNewAuthLibraryEnabled() {
		return newTokenSourceNewAuth(ctx, audience, &ds)
	}
	return newTokenSource(ctx, audience, &ds)
}

func newTokenSourceNewAuth(ctx context.Context, audience string, ds *internal.DialSettings) (oauth2.TokenSource, error) {
	if ds.AuthCredentials != nil {
		return nil, fmt.Errorf("idtoken: option.WithTokenProvider not supported")
	}

	var credsJSON []byte
	var credsType credentialstype.CredType
	var err error

	credsFile, fileCredsType := ds.GetAuthCredentialsFile()
	if credsFile != "" {
		credsJSON, err = os.ReadFile(credsFile)
		if err != nil {
			return nil, fmt.Errorf("idtoken: cannot read credentials file: %v", err)
		}
		credsType = fileCredsType
	} else {
		credsJSON, credsType = ds.GetAuthCredentialsJSON()
	}

	if credsType != credentialstype.Unknown {
		allowed := []credentialstype.CredType{ServiceAccount, ImpersonatedServiceAccount, ExternalAccount}
		if err := credentialstype.CheckCredentialType(credsJSON, credsType, allowed...); err != nil {
			return nil, err
		}
	}

	creds, err := newidtoken.NewCredentials(&newidtoken.Options{
		Audience:        audience,
		CustomClaims:    ds.CustomClaims,
		CredentialsJSON: credsJSON, // Pass the bytes to avoid re-reading the file.
		Client:          oauth2.NewClient(ctx, nil),
		Logger:          ds.Logger,
	})
	if err != nil {
		return nil, err
	}
	return oauth2adapt.TokenSourceFromTokenProvider(creds), nil
}

func newTokenSource(ctx context.Context, audience string, ds *internal.DialSettings) (oauth2.TokenSource, error) {
	creds, err := internal.Creds(ctx, ds)
	if err != nil {
		return nil, err
	}
	if len(creds.JSON) > 0 {
		return tokenSourceFromBytes(ctx, creds.JSON, audience, ds)
	}
	// If internal.Creds did not return a response with JSON fallback to the
	// metadata service as the creds.TokenSource is not an ID token.
	if metadata.OnGCE() {
		return computeTokenSource(audience, ds)
	}
	return nil, fmt.Errorf("idtoken: couldn't find any credentials")
}

func tokenSourceFromBytes(ctx context.Context, data []byte, audience string, ds *internal.DialSettings) (oauth2.TokenSource, error) {
	credType, err := credentialstype.GetCredType(data)
	if err != nil {
		return nil, err
	}
	switch credType {
	case ServiceAccount:
		cfg, err := google.JWTConfigFromJSON(data, ds.GetScopes()...)
		if err != nil {
			return nil, err
		}
		customClaims := ds.CustomClaims
		if customClaims == nil {
			customClaims = make(map[string]interface{})
		}
		customClaims["target_audience"] = audience

		cfg.PrivateClaims = customClaims
		cfg.UseIDToken = true

		ts := cfg.TokenSource(ctx)
		tok, err := ts.Token()
		if err != nil {
			return nil, err
		}
		return oauth2.ReuseTokenSource(tok, ts), nil
	case ImpersonatedServiceAccount, ExternalAccount:
		type url struct {
			ServiceAccountImpersonationURL string `json:"service_account_impersonation_url"`
		}
		var accountURL *url
		if err := json.Unmarshal(data, &accountURL); err != nil {
			return nil, err
		}
		account := filepath.Base(accountURL.ServiceAccountImpersonationURL)
		account = strings.Split(account, ":")[0]

		config := impersonate.IDTokenConfig{
			Audience:        audience,
			TargetPrincipal: account,
			IncludeEmail:    true,
		}
		ts, err := impersonate.IDTokenSource(ctx, config, option.WithAuthCredentialsJSON(credType, data))
		if err != nil {
			return nil, err
		}
		return ts, nil
	default:
		return nil, fmt.Errorf("idtoken: unsupported credentials type: %q", credType)
	}
}

// WithCustomClaims optionally specifies custom private claims for an ID token.
func WithCustomClaims(customClaims map[string]interface{}) ClientOption {
	return withCustomClaims(customClaims)
}

type withCustomClaims map[string]interface{}

func (w withCustomClaims) Apply(o *internal.DialSettings) {
	o.CustomClaims = w
}

// WithCredentialsFile returns a ClientOption that authenticates
// API calls with the given service account or refresh token JSON
// credentials file.
//
// Deprecated:  This function is being deprecated because of a potential security risk.
//
// This function does not validate the credential configuration. The security
// risk occurs when a credential configuration is accepted from a source that
// is not under your control and used without validation on your side.
//
// If you know that you will be loading credential configurations of a
// specific type, it is recommended to use a credential-type-specific
// option function.
// This will ensure that an unexpected credential type with potential for
// malicious intent is not loaded unintentionally. You might still have to do
// validation for certain credential types. Please follow the recommendation
// for that function. For example, if you want to load only service accounts,
// you can use [WithAuthCredentialsFile] with [ServiceAccount]:
//
//	option.WithAuthCredentialsFile(option.ServiceAccount, "/path/to/file.json")
//
// If you are loading your credential configuration from an untrusted source and have
// not mitigated the risks (e.g. by validating the configuration yourself), make
// these changes as soon as possible to prevent security risks to your environment.
//
// Regardless of the function used, it is always your responsibility to validate
// configurations received from external sources.
func WithCredentialsFile(filename string) ClientOption {
	return option.WithCredentialsFile(filename)
}

// WithAuthCredentialsFile returns a ClientOption that authenticates API calls
// with the given JSON credentials file and credential type.
//
// Important: If you accept a credential configuration (credential
// JSON/File/Stream) from an external source for authentication to Google
// Cloud Platform, you must validate it before providing it to any Google
// API or library. Providing an unvalidated credential configuration to
// Google APIs can compromise the security of your systems and data. For
// more information, refer to [Validate credential configurations from
// external sources](https://cloud.google.com/docs/authentication/external/externally-sourced-credentials).
func WithAuthCredentialsFile(credType CredentialsType, filename string) ClientOption {
	return option.WithAuthCredentialsFile(credType, filename)
}

// WithCredentialsJSON returns a ClientOption that authenticates
// API calls with the given service account or refresh token JSON
// credentials.
//
// Deprecated:  This function is being deprecated because of a potential security risk.
//
// This function does not validate the credential configuration. The security
// risk occurs when a credential configuration is accepted from a source that
// is not under your control and used without validation on your side.
//
// If you know that you will be loading credential configurations of a
// specific type, it is recommended to use a credential-type-specific
// option function.
// This will ensure that an unexpected credential type with potential for
// malicious intent is not loaded unintentionally. You might still have to do
// validation for certain credential types. Please follow the recommendation
// for that function. For example, if you want to load only service accounts,
// you can use [WithAuthCredentialsJSON] with [ServiceAccount]:
//
//	option.WithAuthCredentialsJSON(option.ServiceAccount, json)
//
// If you are loading your credential configuration from an untrusted source and have
// not mitigated the risks (e.g. by validating the configuration yourself), make
// these changes as soon as possible to prevent security risks to your environment.
//
// Regardless of the function used, it is always your responsibility to validate
// configurations received from external sources.
func WithCredentialsJSON(p []byte) ClientOption {
	return option.WithCredentialsJSON(p)
}

// WithAuthCredentialsJSON returns a ClientOption that authenticates API calls
// with the given JSON credentials and credential type.
//
// Important: If you accept a credential configuration (credential
// JSON/File/Stream) from an external source for authentication to Google
// Cloud Platform, you must validate it before providing it to any Google
// API or library. Providing an unvalidated credential configuration to
// Google APIs can compromise the security of your systems and data. For
// more information, refer to [Validate credential configurations from
// external sources](https://cloud.google.com/docs/authentication/external/externally-sourced-credentials).
func WithAuthCredentialsJSON(credType CredentialsType, json []byte) ClientOption {
	return option.WithAuthCredentialsJSON(credType, json)
}

// WithHTTPClient returns a ClientOption that specifies the HTTP client to use
// as the basis of communications. This option may only be used with services
// that support HTTP as their communication transport. When used, the
// WithHTTPClient option takes precedent over all other supplied options.
func WithHTTPClient(client *http.Client) ClientOption {
	return option.WithHTTPClient(client)
}
