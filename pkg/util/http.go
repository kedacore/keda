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
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

var disableKeepAlives bool
var minTLSVersion uint16

func init() {
	setupLog := ctrl.Log.WithName("http_setup")
	var err error
	// This code will be removed in https://github.com/kedacore/keda/pull/4191
	// nosemgrep: trailofbits.go.invalid-usage-of-modified-variable.invalid-usage-of-modified-variable
	disableKeepAlives, err = ResolveOsEnvBool("KEDA_HTTP_DISABLE_KEEP_ALIVE", false)
	if err != nil {
		disableKeepAlives = false
	}

	minTLSVersion = initMinTLSVersion(setupLog)
}

func initMinTLSVersion(logger logr.Logger) uint16 {
	version, found := os.LookupEnv("KEDA_HTTP_MIN_TLS_VERSION")
	minVersion := tls.VersionTLS12
	if found {
		switch version {
		case "TLS13":
			minVersion = tls.VersionTLS13
		case "TLS12":
			minVersion = tls.VersionTLS12
		case "TLS11":
			minVersion = tls.VersionTLS11
		case "TLS10":
			minVersion = tls.VersionTLS10
		default:
			logger.Info(fmt.Sprintf("%s is not a valid value, using `TLS12`. Allowed values are: `TLS13`,`TLS12`,`TLS11`,`TLS10`", version))
			minVersion = tls.VersionTLS12
		}
	}
	return uint16(minVersion)
}

var rootCAs *x509.CertPool

func init() {
	setupLog := ctrl.Log.WithName("http_setup")
	disableKeepAlives = getKeepAliveValue()
	rootCAs = getRootCAs()
	minTLSVersion = initMinTLSVersion(setupLog)
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
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: unsafeSsl,
			MinVersion:         GetMinTLSVersion(),
			RootCAs:            rootCAs,
		},
		Proxy: http.ProxyFromEnvironment,
	}
	if disableKeepAlives {
		// disable keep http connection alive
		transport.DisableKeepAlives = true
		transport.IdleConnTimeout = 100 * time.Second
	}
	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	return httpClient
}

func GetMinTLSVersion() uint16 {
	return minTLSVersion
}
