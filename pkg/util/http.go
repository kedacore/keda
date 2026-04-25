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
	"sync"
	"time"
)

var disableKeepAlives bool

// sharedTransports holds one http.Transport per unsafeSsl value so that
// scalers using CreateHTTPClient share a single connection pool and DNS
// resolver cache for each TLS-verification mode, instead of each scaler
// instantiating its own Transport.
//
// Why: at high ScaledObject counts (measured at 30k Prom triggers), per-scaler
// Transports force a fresh TCP connection and DNS lookup every time any
// scaler's pool misses, because pools are not shared across scalers. When a
// shared metric source (e.g. the single Prom behind all scalers) has
// connections drop — on pod restart, idle eviction, or intermittent network
// issue — every affected scaler independently re-dials. The aggregate
// re-dial rate saturates the network path, and KEDA's default 3s HTTP
// timeout starts firing during connection setup. Sharing one Transport per
// TLS mode collapses N cold-dials-per-drop to 1.
var (
	sharedTransportsMu sync.RWMutex
	sharedTransports   = map[bool]*http.Transport{}
)

// Connection pool sizing on the shared Transport. Defaults aim at
// high-scale deployments (thousands of ScaledObjects fanning out to a small
// number of metric-source hosts). Go stdlib defaults are MaxIdleConnsPerHost=2
// and MaxIdleConns=100, which are far too small for this pattern.
const (
	sharedTransportMaxIdleConns        = 0 // 0 means no limit
	sharedTransportMaxIdleConnsPerHost = 1000
	sharedTransportIdleConnTimeout     = 90 * time.Second
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

// sharedHTTPTransport returns a process-wide *http.Transport for the given
// unsafeSsl value, constructing one on first use. Subsequent calls with the
// same unsafeSsl return the same Transport pointer, so all scalers using
// CreateHTTPClient share a connection pool and DNS cache per TLS mode.
func sharedHTTPTransport(unsafeSsl bool) *http.Transport {
	sharedTransportsMu.RLock()
	t, ok := sharedTransports[unsafeSsl]
	sharedTransportsMu.RUnlock()
	if ok {
		return t
	}

	sharedTransportsMu.Lock()
	defer sharedTransportsMu.Unlock()
	// Double-check after acquiring write lock in case a concurrent caller won.
	if t, ok := sharedTransports[unsafeSsl]; ok {
		return t
	}

	t = &http.Transport{
		TLSClientConfig:     CreateTLSClientConfig(unsafeSsl),
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        sharedTransportMaxIdleConns,
		MaxIdleConnsPerHost: sharedTransportMaxIdleConnsPerHost,
		IdleConnTimeout:     sharedTransportIdleConnTimeout,
	}
	if disableKeepAlives {
		t.DisableKeepAlives = true
		t.IdleConnTimeout = 100 * time.Second
	}
	sharedTransports[unsafeSsl] = t
	return t
}

// HTTPDoer is an interface that matches the Do method on
// (net/http).Client. It should be used in function signatures
// instead of raw *http.Clients wherever possible
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// CreateHTTPClient returns a new HTTP client with the timeout set to
// timeoutMS milliseconds, or 300 milliseconds if timeoutMS <= 0.
// unsafeSsl parameter allows to avoid tls cert validation if it's required.
//
// The returned *http.Client is per-caller (so its Timeout field is
// independent) but its underlying *http.Transport is shared with every other
// caller that passes the same unsafeSsl value. This means all scalers using
// CreateHTTPClient share a single connection pool and DNS resolver cache per
// TLS mode, instead of each owning its own. See the sharedHTTPTransport
// doc comment for motivation.
//
// Callers that need a private Transport (e.g. to supply a custom
// *tls.Config) should use CreateHTTPTransportWithTLSConfig directly and
// compose their own http.Client.
func CreateHTTPClient(timeout time.Duration, unsafeSsl bool) *http.Client {
	// default the timeout to 300ms
	if timeout <= 0 {
		timeout = 300 * time.Millisecond
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: sharedHTTPTransport(unsafeSsl),
	}
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
