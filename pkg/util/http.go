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
	"net/http"
	"time"
)

var disableKeepAlives bool

func init() {
	disableKeepAlives = getKeepAliveValue()
}

func getKeepAliveValue() bool {
	if val, err := ResolveOsEnvBool("KEDA_HTTP_DISABLE_KEEP_ALIVE", false); err == nil {
		return val
	}
	return false
}

// HTTPDoer is an interface that matches the Do method on
// (net/http).Client. It should be used in function signatures
// instead of raw *http.Clients wherever possible
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// CreateHTTPClient returns a new HTTP client with the timeout set to
// timeoutMS milliseconds, or 300 milliseconds if timeoutMS <= 0.
// unsafeSsl parameter allows to avoid tls cert validation if it's required
func CreateHTTPClient(timeout time.Duration, unsafeSsl bool) *http.Client {
	// default the timeout to 300ms
	if timeout <= 0 {
		timeout = 300 * time.Millisecond
	}
	transport := CreateHTTPTransport(unsafeSsl)
	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	return httpClient
}

// CreateHTTPTransport returns a new HTTP Transport with Proxy, Keep alives
// unsafeSsl parameter allows to avoid tls cert validation if it's required
func CreateHTTPTransport(unsafeSsl bool) *http.Transport {
	return CreateHTTPTransportWithTLSConfig(CreateTLSClientConfig(unsafeSsl))
}

// CreateHTTPTransportWithTLSConfig returns a new HTTP Transport with Proxy, Keep alives
// using given tls.Config
func CreateHTTPTransportWithTLSConfig(config *tls.Config) *http.Transport {
	transport := &http.Transport{
		TLSClientConfig: config,
		Proxy:           http.ProxyFromEnvironment,
	}
	if disableKeepAlives {
		// disable keep http connection alive
		transport.DisableKeepAlives = true
		transport.IdleConnTimeout = 100 * time.Second
	}
	return transport
}
