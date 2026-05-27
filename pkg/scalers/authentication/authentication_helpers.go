package authentication

import (
	"fmt"
	"net"
	"net/http"
	"time"

	libs "github.com/dysnix/predictkube-libs/external/configs"
	"github.com/dysnix/predictkube-libs/external/http_transport"
	pConfig "github.com/prometheus/common/config"
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
)

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
