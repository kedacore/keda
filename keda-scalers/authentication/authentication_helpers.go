package authentication

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
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

func GetAuthConfigs(triggerMetadata, authParams map[string]string) (out *AuthMeta, err error) {
	out = &AuthMeta{}

	authModes, ok := triggerMetadata[AuthModesKey]
	// no authMode specified
	if !ok {
		return nil, nil
	}

	authTypes := strings.Split(authModes, ",")
	for _, t := range authTypes {
		authType := Type(strings.TrimSpace(t))

		switch authType {
		case BearerAuthType:
			if len(authParams["bearerToken"]) == 0 {
				return nil, errors.New("no bearer token provided")
			}
			if out.EnableBasicAuth {
				return nil, errors.New("both bearer and basic authentication can not be set")
			}
			if out.EnableOAuth {
				return nil, errors.New("both bearer and OAuth can not be set")
			}
			out.BearerToken = strings.TrimSuffix(authParams["bearerToken"], "\n")
			out.EnableBearerAuth = true
		case BasicAuthType:
			if len(authParams["username"]) == 0 {
				return nil, errors.New("no username given")
			}
			if out.EnableBearerAuth {
				return nil, errors.New("both bearer and basic authentication can not be set")
			}
			if out.EnableOAuth {
				return nil, errors.New("both bearer and OAuth can not be set")
			}

			out.Username = authParams["username"]
			// password is optional. For convenience, many application implement basic auth with
			// username as apikey and password as empty
			out.Password = authParams["password"]
			out.EnableBasicAuth = true
		case TLSAuthType:
			if len(authParams["cert"]) == 0 {
				return nil, errors.New("no cert given")
			}
			out.Cert = authParams["cert"]

			if len(authParams["key"]) == 0 {
				return nil, errors.New("no key given")
			}

			out.Key = authParams["key"]
			out.EnableTLS = true
		case CustomAuthType:
			if len(authParams["customAuthHeader"]) == 0 {
				return nil, errors.New("no custom auth header given")
			}
			out.CustomAuthHeader = strings.TrimSuffix(authParams["customAuthHeader"], "\n")

			if len(authParams["customAuthValue"]) == 0 {
				return nil, errors.New("no custom auth value given")
			}
			out.CustomAuthValue = strings.TrimSuffix(authParams["customAuthValue"], "\n")
			out.EnableCustomAuth = true
		case OAuthType:
			if out.EnableBasicAuth {
				return nil, errors.New("both oauth and basic authentication can not be set")
			}
			if out.EnableBearerAuth {
				return nil, errors.New("both oauth and bearer authentication can not be set")
			}
			out.EnableOAuth = true
			out.OauthTokenURI = authParams["oauthTokenURI"]
			out.Scopes = ParseScope(authParams["scope"])
			out.ClientID = authParams["clientID"]
			out.ClientSecret = authParams["clientSecret"]

			v, err := ParseEndpointParams(authParams["endpointParams"])
			if err != nil {
				return nil, fmt.Errorf("incorrect value for endpointParams is given: %s", authParams["endpointParams"])
			}
			out.EndpointParams = v
		default:
			return nil, fmt.Errorf("incorrect value for authMode is given: %s", t)
		}
	}

	if len(authParams["ca"]) > 0 {
		out.CA = authParams["ca"]
	}

	return out, err
}

// ParseScope parse OAuth scopes from a comma separated string
// whitespace is trimmed
func ParseScope(inputStr string) []string {
	scope := strings.TrimSpace(inputStr)
	if scope != "" {
		scopes := make([]string, 0)
		list := strings.Split(scope, ",")
		for _, sc := range list {
			sc := strings.TrimSpace(sc)
			if sc != "" {
				scopes = append(scopes, sc)
			}
		}
		if len(scopes) == 0 {
			return nil
		}
		return scopes
	}
	return nil
}

// ParseEndpointParams parse OAuth endpoint params from URL-encoded query string.
func ParseEndpointParams(inputStr string) (url.Values, error) {
	v, err := url.ParseQuery(inputStr)
	if err != nil {
		return nil, err
	}
	if len(v) == 0 {
		return nil, nil
	}
	return v, nil
}

func GetBearerToken(auth *AuthMeta) string {
	return fmt.Sprintf("Bearer %s", auth.BearerToken)
}

func NewTLSConfig(auth *AuthMeta, unsafeSsl bool) (*tls.Config, error) {
	return kedautil.NewTLSConfig(
		auth.Cert,
		auth.Key,
		auth.CA,
		unsafeSsl,
	)
}

func CreateHTTPRoundTripper(roundTripperType TransportType, auth *AuthMeta, conf ...*HTTPTransport) (rt http.RoundTripper, err error) {
	unsafeSsl := false
	tlsConfig := kedautil.CreateTLSClientConfig(unsafeSsl)
	if auth != nil && (auth.CA != "" || auth.EnableTLS) {
		tlsConfig, err = NewTLSConfig(auth, unsafeSsl)
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

		if auth != nil {
			if auth.EnableBasicAuth {
				rt = pConfig.NewBasicAuthRoundTripper(
					pConfig.NewInlineSecret(auth.Username),
					pConfig.NewInlineSecret(auth.Password),
					roundTripper,
				)
			}

			if auth.EnableBearerAuth {
				rt = pConfig.NewAuthorizationCredentialsRoundTripper(
					"Bearer",
					pConfig.NewInlineSecret(auth.BearerToken),
					roundTripper,
				)
			}
		} else {
			rt = roundTripper
		}

		return rt, nil
	}

	return rt, nil
}
