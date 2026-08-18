package authentication

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	libs "github.com/dysnix/predictkube-libs/external/configs"
	"github.com/dysnix/predictkube-libs/external/http_transport"
	pConfig "github.com/prometheus/common/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type AuthClientSet struct {
	corev1client.CoreV1Interface
	corev1listers.SecretLister
}

const (
	AuthModesKey = "authModes"

	// InsecureOAuthWarningMessage is emitted when OAuth is configured without a client secret and without mTLS,
	// which leaves the client effectively unauthenticated.
	InsecureOAuthWarningMessage = "OAuth is configured without clientSecret and without mTLS (RFC 8705). Add a clientSecret or enable mTLS to ensure secure authentication."

	// unusedClientSecret satisfies Go's OAuth library, which requires a non-empty secret,
	// for the mTLS client authentication flow (RFC 8705) that does not use one.
	unusedClientSecret = "unused"
)

// InsecureOAuthWarning returns a warning message when OAuth is enabled without a client secret and without mTLS,
// and an empty string otherwise.
// Such a configuration is accepted so that mTLS client authentication (RFC 8705) can be terminated by a proxy,
// but it cannot be distinguished from a missing secret, so callers should surface this.
func (c *Config) InsecureOAuthWarning() string {
	if c.EnabledOAuth() && c.ClientSecret == "" && !c.EnabledTLS() {
		return InsecureOAuthWarningMessage
	}
	return ""
}

// OAuthTokenSource returns an [oauth2.TokenSource] that performs the client credentials
// flow using the Config's OAuth settings.
// The returned source caches the token and refreshes it only once it expires,
// so it should be created once per scaler and reused across requests rather than rebuilt per request.
//
// tokenClient is used for the requests to the token endpoint, so that they share the scaler's timeout and TLS settings.
// It must not be a client whose transport is wrapped by the returned token source, as token requests would then recurse back into it.
func (c *Config) OAuthTokenSource(ctx context.Context, tokenClient *http.Client) oauth2.TokenSource {
	clientSecret := c.ClientSecret
	if clientSecret == "" {
		clientSecret = unusedClientSecret
	}

	cfg := clientcredentials.Config{
		ClientID:       c.ClientID,
		ClientSecret:   clientSecret,
		TokenURL:       c.OauthTokenURI,
		Scopes:         c.Scopes,
		EndpointParams: c.EndpointParams,
	}

	if tokenClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, tokenClient)
	}

	return cfg.TokenSource(ctx)
}

// CreateHTTPRoundTripper builds an http.RoundTripper using the auth settings from the given Config (TLS, basic, bearer).
func CreateHTTPRoundTripper(roundTripperType TransportType, auth *Config, conf ...*HTTPTransport) (rt http.RoundTripper, err error) {
	unsafeSsl := false
	tlsConfig := kedautil.CreateTLSClientConfig(unsafeSsl)
	if auth != nil && (auth.CA != "" || auth.EnabledTLS()) {
		tlsConfig, err = auth.NewTLSConfig(unsafeSsl)
		if err != nil || tlsConfig == nil {
			return nil, fmt.Errorf("error creating the TLS config: %w", err)
		}
	}

	switch roundTripperType {
	case NetHTTP:
		// from official github.com/prometheus/client_golang/api package
		return &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     tlsConfig,
		}, nil
	case FastHTTP:
		// default configs
		httpConf := &libs.HTTPTransport{
			MaxIdleConnDuration: 10,
			ReadTimeout:         time.Second * 15,
			WriteTimeout:        time.Second * 15,
		}

		if len(conf) > 0 {
			httpConf = &libs.HTTPTransport{
				MaxIdleConnDuration: conf[0].MaxIdleConnDuration,
				ReadTimeout:         conf[0].ReadTimeout,
				WriteTimeout:        conf[0].WriteTimeout,
			}
		}

		var roundTripper http.RoundTripper
		if roundTripper, err = http_transport.NewHttpTransport(
			libs.SetTransportConfigs(httpConf),
			libs.SetTLS(tlsConfig),
		); err != nil {
			return nil, fmt.Errorf("error creating fast http round tripper: %w", err)
		}

		if !auth.Disabled() {
			if auth.EnabledBasicAuth() {
				rt = pConfig.NewBasicAuthRoundTripper(
					pConfig.NewInlineSecret(auth.Username),
					pConfig.NewInlineSecret(auth.Password),
					roundTripper,
				)
			}

			if auth.EnabledBearerAuth() {
				rt = pConfig.NewAuthorizationCredentialsRoundTripper(
					"Bearer",
					pConfig.NewInlineSecret(auth.BearerToken),
					roundTripper,
				)
			}
		}
		if rt == nil {
			rt = roundTripper
		}

		return rt, nil
	}

	return rt, nil
}
