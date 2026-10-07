/*
Copyright 2023 The KEDA Authors

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
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type redirectTestTransport func(*http.Request) (*http.Response, error)

func (f redirectTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestCheckSameOriginRedirect(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		for _, tc := range []struct {
			name         string
			destinations []string
			wantRequests int
			wantError    string
		}{
			{name: "relative path", destinations: []string{"/metrics"}, wantRequests: 2},
			{name: "same origin", destinations: []string{"https://metrics.example/metrics"}, wantRequests: 2},
			{name: "case insensitive host", destinations: []string{"https://METRICS.example/metrics"}, wantRequests: 2},
			{name: "explicit default port", destinations: []string{"https://metrics.example:443/metrics"}, wantRequests: 2},
			{name: "different host", destinations: []string{"https://other.example/metrics"}, wantRequests: 1, wantError: "different origin"},
			{name: "subdomain", destinations: []string{"https://other.metrics.example/metrics"}, wantRequests: 1, wantError: "different origin"},
			{name: "different port", destinations: []string{"https://metrics.example:8443/metrics"}, wantRequests: 1, wantError: "different origin"},
			{name: "downgrade", destinations: []string{"http://metrics.example/metrics"}, wantRequests: 1, wantError: "different origin"},
			{name: "user information", destinations: []string{"https://user:password@metrics.example/metrics"}, wantRequests: 1, wantError: "user information"},
			{name: "redirect chain", destinations: []string{"/next", "http://other.example/metrics"}, wantRequests: 2, wantError: "different origin"},
			{name: "redirect loop", destinations: strings.Fields(strings.Repeat("/loop ", 10)), wantRequests: 10, wantError: "stopped after 10 redirects"},
		} {
			t.Run(http.StatusText(status)+"/"+tc.name, func(t *testing.T) {
				requests := 0
				client := &http.Client{
					CheckRedirect: CheckSameOriginRedirect,
					// OAuth transports add authorization even when net/http would
					// strip it from a request to a different host.
					Transport: &oauth2.Transport{
						Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-cloud-token"}),
						Base: redirectTestTransport(func(request *http.Request) (*http.Response, error) {
							require.Equal(t, "Bearer test-cloud-token", request.Header.Get("Authorization"))
							response := &http.Response{
								StatusCode: http.StatusOK, Header: make(http.Header),
								Body: io.NopCloser(strings.NewReader("")), Request: request,
							}
							if requests < len(tc.destinations) {
								response.StatusCode = status
								response.Header.Set("Location", tc.destinations[requests])
							}
							requests++
							return response, nil
						}),
					},
				}
				request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://metrics.example/start", nil)
				require.NoError(t, err)
				response, err := client.Do(request)
				if response != nil {
					require.NoError(t, response.Body.Close())
				}
				if tc.wantError != "" {
					require.ErrorContains(t, err, tc.wantError)
				} else {
					require.NoError(t, err)
				}
				assert.Equal(t, tc.wantRequests, requests, "a forbidden redirect must not reach the transport")
			})
		}
	}
}

func TestCreateHTTPClientWhenNegativeTimeout(t *testing.T) {
	client := CreateHTTPClient(-1*time.Minute, false)

	assert.Equal(t, 300*time.Millisecond, client.Timeout)
}

func TestCreateHTTPClientWhenValidTimeout(t *testing.T) {
	client := CreateHTTPClient(1*time.Minute, false)

	assert.Equal(t, 1*time.Minute, client.Timeout)
}

func TestCreateHTTPClientSharesTransport(t *testing.T) {
	first := CreateHTTPClient(time.Second, false)
	second := CreateHTTPClient(time.Minute, false)

	assert.NotSame(t, first, second)
	assert.Same(t, first.Transport, second.Transport)
	assert.Equal(t, time.Second, first.Timeout)
	assert.Equal(t, time.Minute, second.Timeout)
}

func TestCreateHTTPClientSeparatesTLSModes(t *testing.T) {
	secure := CreateHTTPClient(time.Second, false)
	insecure := CreateHTTPClient(time.Second, true)

	assert.NotSame(t, secure.Transport, insecure.Transport)
}

func TestCreateHTTPClientSharesTransportConcurrently(t *testing.T) {
	const goroutines = 100
	transports := make(chan http.RoundTripper, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			transports <- CreateHTTPClient(time.Second, false).Transport
		}()
	}
	wg.Wait()
	close(transports)

	var first http.RoundTripper
	for transport := range transports {
		if first == nil {
			first = transport
			continue
		}
		assert.Same(t, first, transport)
	}
}

func TestCreateRTSharesTransport(t *testing.T) {
	first := CreateRT(false)
	second := CreateRT(false)

	assert.Same(t, first, second)
}

func TestCreateRTWithTLSConfigRemainsPrivate(t *testing.T) {
	first := CreateRTWithTLSConfig(nil)
	second := CreateRTWithTLSConfig(nil)

	assert.NotSame(t, first, second)
}

func TestHTTPTransportConfigValidation(t *testing.T) {
	testCases := []struct {
		name   string
		config HTTPTransportConfig
	}{
		{
			name:   "negative max idle connections",
			config: HTTPTransportConfig{MaxIdleConns: -1},
		},
		{
			name:   "negative max idle connections per host",
			config: HTTPTransportConfig{MaxIdleConnsPerHost: -1},
		},
		{
			name:   "negative idle connection timeout",
			config: HTTPTransportConfig{MaxIdleConnsPerHost: 1, IdleConnTimeout: -time.Second},
		},
		{
			name:   "zero idle connection timeout",
			config: HTTPTransportConfig{MaxIdleConnsPerHost: 1},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Error(t, validateHTTPTransportConfig(testCase.config))
		})
	}
	assert.NoError(t, validateHTTPTransportConfig(HTTPTransportConfig{MaxIdleConnsPerHost: 1, IdleConnTimeout: time.Second}))
}

func TestCreateHTTPTransport(t *testing.T) {
	config := HTTPTransportConfig{
		MaxIdleConns:        25,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     time.Minute,
	}
	secure := createHTTPTransport(CreateTLSClientConfig(false), config)
	insecure := createHTTPTransport(CreateTLSClientConfig(true), config)
	assert.Equal(t, config.MaxIdleConns, secure.MaxIdleConns)
	assert.Equal(t, config.MaxIdleConnsPerHost, secure.MaxIdleConnsPerHost)
	assert.Equal(t, config.IdleConnTimeout, secure.IdleConnTimeout)
	assert.False(t, secure.TLSClientConfig.InsecureSkipVerify)
	assert.True(t, insecure.TLSClientConfig.InsecureSkipVerify)
}

func TestCreateHTTPTransportAppliesPoolConfig(t *testing.T) {
	config := HTTPTransportConfig{
		MaxIdleConns:        25,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     time.Minute,
	}
	transport := createHTTPTransport(&tls.Config{MinVersion: tls.VersionTLS12}, config)

	assert.Equal(t, config.MaxIdleConns, transport.MaxIdleConns)
	assert.Equal(t, config.MaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)
	assert.Equal(t, config.IdleConnTimeout, transport.IdleConnTimeout)
}

func TestCreateHTTPTransportDisablesKeepAlive(t *testing.T) {
	previousValue := disableKeepAlives
	disableKeepAlives = true
	t.Cleanup(func() { disableKeepAlives = previousValue })

	config := HTTPTransportConfig{IdleConnTimeout: time.Minute}
	transport := createHTTPTransport(CreateTLSClientConfig(false), config)

	assert.True(t, transport.DisableKeepAlives)
	assert.Equal(t, config.IdleConnTimeout, transport.IdleConnTimeout)
}
