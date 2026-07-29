/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
)

// HTTPTransportConfig configures the shared HTTP transports.
type HTTPTransportConfig struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
}

var (
	disableKeepAlives   bool
	httpTransportConfig HTTPTransportConfig
	sharedSecureRT      = sync.OnceValue(func() http.RoundTripper { return createSharedRT(false) })
	sharedInsecureRT    = sync.OnceValue(func() http.RoundTripper { return createSharedRT(true) })
)

func init() {
	disableKeepAlives = getKeepAliveValue()
}

func getKeepAliveValue() bool {
	if val, err := ResolveOsEnvBool("KEDA_HTTP_DISABLE_KEEP_ALIVE", false); err == nil {
		return val
	}
	return false
}

func validateHTTPTransportConfig(config HTTPTransportConfig) error {
	if config.MaxIdleConns < 0 {
		return fmt.Errorf("http max idle connections must be non-negative")
	}
	if config.MaxIdleConnsPerHost <= 0 {
		return fmt.Errorf("http max idle connections per host must be larger than zero")
	}
	if config.IdleConnTimeout <= 0 {
		return fmt.Errorf("http idle connection timeout must be larger than zero")
	}
	return nil
}

// ConfigureHTTPTransport configures shared HTTP transports. It must be called before creating an HTTP client.
func ConfigureHTTPTransport(config HTTPTransportConfig) error {
	if err := validateHTTPTransportConfig(config); err != nil {
		return err
	}
	httpTransportConfig = config
	return nil
}

// HTTPDoer is an interface that matches the Do method on
// (net/http).Client. It should be used in function signatures
// instead of raw *http.Clients wherever possible
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// CreateHTTPClient returns an HTTP client with the timeout set to timeout,
// or 300 milliseconds if timeout <= 0. Clients with the same unsafeSsl value
// share an instrumented transport and connection pool.
func CreateHTTPClient(timeout time.Duration, unsafeSsl bool) *http.Client {
	if timeout <= 0 {
		timeout = 300 * time.Millisecond
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: CreateRT(unsafeSsl),
	}
}

func createSharedRT(unsafeSsl bool) http.RoundTripper {
	return metricscollector.NewInstrumentedRoundTripper(createHTTPTransport(CreateTLSClientConfig(unsafeSsl), httpTransportConfig))
}

// CreateRT returns a shared instrumented HTTP RoundTripper with Proxy and Keep-alive settings.
// unsafeSsl parameter allows to avoid tls cert validation if it's required.
func CreateRT(unsafeSsl bool) http.RoundTripper {
	if unsafeSsl {
		return sharedInsecureRT()
	}
	return sharedSecureRT()
}

// CreateRTWithTLSConfig returns a new instrumented HTTP RoundTripper with Proxy and
// Keep-alive settings using the given tls.Config.
func CreateRTWithTLSConfig(config *tls.Config) http.RoundTripper {
	return metricscollector.NewInstrumentedRoundTripperWithCloseIdleConnections(createHTTPTransport(config, httpTransportConfig))
}

func createHTTPTransport(tlsConfig *tls.Config, config HTTPTransportConfig) *http.Transport {
	return &http.Transport{
		TLSClientConfig:     tlsConfig,
		Proxy:               http.ProxyFromEnvironment,
		DisableKeepAlives:   disableKeepAlives,
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
	}
}
