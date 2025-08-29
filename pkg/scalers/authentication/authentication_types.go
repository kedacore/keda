package authentication

import (
	"fmt"
	"net/url"
	"time"
)

// Type describes the authentication type used in a scaler
type Type string

const (
	// APIKeyAuthType is an auth type using an API key
	APIKeyAuthType Type = "apiKey"
	// BasicAuthType is an auth type using basic auth
	BasicAuthType Type = "basic"
	// TLSAuthType is an auth type using TLS
	TLSAuthType Type = "tls"
	// BearerAuthType is an auth type using a bearer token
	BearerAuthType Type = "bearer"
	// CustomAuthType is an auth type using a custom header
	CustomAuthType Type = "custom"
	// OAuthType is an auth type using a oAuth2
	OAuthType Type = "oauth"
)

// TransportType is type of http transport
type TransportType int

const (
	NetHTTP  TransportType = iota // NetHTTP standard Go net/http client.
	FastHTTP                      // FastHTTP Fast http client.
)

// AuthMeta is the metadata for the authentication types
// Deprecated: use Config instead
type AuthMeta struct {
	// bearer auth
	EnableBearerAuth bool
	BearerToken      string

	// basic auth
	EnableBasicAuth bool
	Username        string
	Password        string // +optional

	// client certification
	EnableTLS bool
	Cert      string
	Key       string
	CA        string

	// oAuth2
	EnableOAuth    bool
	OauthTokenURI  string
	Scopes         []string
	ClientID       string
	ClientSecret   string
	EndpointParams url.Values

	// custom auth header
	EnableCustomAuth bool
	CustomAuthHeader string
	CustomAuthValue  string
}

// BasicAuth is a basic authentication type
type BasicAuth struct {
	Username string `keda:"name=username, order=authParams"`
	Password string `keda:"name=password, order=authParams"`
}

// CertAuth is a client certificate authentication type
type CertAuth struct {
	Cert string `keda:"name=cert, order=authParams"`
	Key  string `keda:"name=key, order=authParams"`
	CA   string `keda:"name=ca, order=authParams"`
}

// OAuth is an oAuth2 authentication type
type OAuth struct {
	OauthTokenURI  string     `keda:"name=oauthTokenURI,  order=authParams"`
	Scopes         []string   `keda:"name=scopes,         order=authParams"`
	ClientID       string     `keda:"name=clientID,       order=authParams"`
	ClientSecret   string     `keda:"name=clientSecret,   order=authParams"`
	EndpointParams url.Values `keda:"name=endpointParams, order=authParams"`
}

// APIKeyAuth is an API key authentication type
type APIKeyAuth struct {
	APIKey       string `keda:"name=apiKey, order=triggerMetadata;authParams"`
	Method       string `keda:"name=method, order=triggerMetadata;authParams, default=header, enum=header;query"`
	KeyParamName string `keda:"name=keyParamName, order=triggerMetadata;authParams, optional"`
}

// CustomAuth is a custom header authentication type
type CustomAuth struct {
	CustomAuthHeader string `keda:"name=customAuthHeader, order=authParams"`
	CustomAuthValue  string `keda:"name=customAuthValue,  order=authParams"`
}

// Config is the configuration for the authentication types
type Config struct {
	Modes []Type `keda:"name=authModes;authMode, order=triggerMetadata;authParams, enum=apiKey;basic;tls;bearer;custom;oauth, exclusiveSet=bearer;basic;oauth, optional"`

	BearerToken string `keda:"name=bearerToken;token, order=authParams, optional"`
	BasicAuth   `keda:"optional"`
	CertAuth    `keda:"optional"`
	OAuth       `keda:"optional"`
	CustomAuth  `keda:"optional"`
	APIKeyAuth  `keda:"optional"`
}

// Disabled returns true if no auth modes are enabled
func (c *Config) Disabled() bool {
	return c == nil || len(c.Modes) == 0
}

// Enabled returns true if given auth mode is enabled
func (c *Config) Enabled(mode Type) bool {
	for _, m := range c.Modes {
		if m == mode {
			return true
		}
	}
	return false
}

// helpers for checking enabled auth modes
func (c *Config) EnabledTLS() bool        { return c.Enabled(TLSAuthType) }
func (c *Config) EnabledBasicAuth() bool  { return c.Enabled(BasicAuthType) }
func (c *Config) EnabledBearerAuth() bool { return c.Enabled(BearerAuthType) }
func (c *Config) EnabledOAuth() bool      { return c.Enabled(OAuthType) }
func (c *Config) EnabledCustomAuth() bool { return c.Enabled(CustomAuthType) }
func (c *Config) EnabledAPIKeyAuth() bool { return c.Enabled(APIKeyAuthType) }

// GetBearerToken returns the bearer token with the Bearer prefix
func (c *Config) GetBearerToken() string {
	return fmt.Sprintf("Bearer %s", c.BearerToken)
}

// Validate validates the Config and returns an error if it is invalid
func (c *Config) Validate() error {
	if c.Disabled() {
		return nil
	}
	if c.EnabledBearerAuth() && c.BearerToken == "" {
		return fmt.Errorf("bearer token is required when bearer auth is enabled")
	}
	if c.EnabledBasicAuth() && c.Username == "" {
		return fmt.Errorf("username is required when basic auth is enabled")
	}
	if c.EnabledTLS() && (c.Cert == "" || c.Key == "" || c.CA == "") {
		return fmt.Errorf("cert and key are required when tls auth is enabled")
	}
	if c.EnabledOAuth() && (c.OauthTokenURI == "" || c.ClientID == "" || c.ClientSecret == "") {
		return fmt.Errorf("oauthTokenURI, clientID and clientSecret are required when oauth is enabled")
	}
	if c.EnabledCustomAuth() && (c.CustomAuthHeader == "" || c.CustomAuthValue == "") {
		return fmt.Errorf("customAuthHeader and customAuthValue are required when custom auth is enabled")
	}
	if c.EnabledAPIKeyAuth() && c.APIKey == "" {
		return fmt.Errorf("apiKey is required when apiKey auth is enabled")
	}
	return nil
}

// ToAuthMeta converts the Config to deprecated AuthMeta
func (c *Config) ToAuthMeta() *AuthMeta {
	if c.Disabled() {
		return nil
	}
	return &AuthMeta{
		// bearer auth
		EnableBearerAuth: c.EnabledBearerAuth(),
		BearerToken:      c.BearerToken,

		// basic auth
		EnableBasicAuth: c.EnabledBasicAuth(),
		Username:        c.Username,
		Password:        c.Password,

		// client certification
		EnableTLS: c.EnabledTLS(),
		Cert:      c.Cert,
		Key:       c.Key,
		CA:        c.CA,

		// oAuth2
		EnableOAuth:    c.EnabledOAuth(),
		OauthTokenURI:  c.OauthTokenURI,
		Scopes:         c.Scopes,
		ClientID:       c.ClientID,
		ClientSecret:   c.ClientSecret,
		EndpointParams: c.EndpointParams,

		// custom auth header
		EnableCustomAuth: c.EnabledCustomAuth(),
		CustomAuthHeader: c.CustomAuthHeader,
		CustomAuthValue:  c.CustomAuthValue,
	}
}

type HTTPTransport struct {
	MaxIdleConnDuration time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
}
