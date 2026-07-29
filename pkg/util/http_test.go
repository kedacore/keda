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
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

func TestCreateRTRemainsPrivate(t *testing.T) {
	first := CreateRT(false)
	second := CreateRT(false)

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
			config: HTTPTransportConfig{IdleConnTimeout: -time.Second},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Error(t, validateHTTPTransportConfig(testCase.config))
		})
	}
	assert.NoError(t, validateHTTPTransportConfig(HTTPTransportConfig{MaxIdleConnsPerHost: 1}))
}

func TestCreateSharedHTTPTransport(t *testing.T) {
	config := HTTPTransportConfig{
		MaxIdleConns:        25,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     time.Minute,
	}
	secure := createSharedHTTPTransport(false, config)
	insecure := createSharedHTTPTransport(true, config)
	assert.Equal(t, config.MaxIdleConns, secure.MaxIdleConns)
	assert.Equal(t, config.MaxIdleConnsPerHost, secure.MaxIdleConnsPerHost)
	assert.Equal(t, config.IdleConnTimeout, secure.IdleConnTimeout)
	assert.False(t, secure.TLSClientConfig.InsecureSkipVerify)
	assert.True(t, insecure.TLSClientConfig.InsecureSkipVerify)
}

func TestCreateSharedHTTPTransportDisablesKeepAlive(t *testing.T) {
	previousValue := disableKeepAlives
	disableKeepAlives = true
	t.Cleanup(func() { disableKeepAlives = previousValue })

	config := HTTPTransportConfig{IdleConnTimeout: time.Minute}
	transport := createSharedHTTPTransport(false, config)

	assert.True(t, transport.DisableKeepAlives)
	assert.Equal(t, config.IdleConnTimeout, transport.IdleConnTimeout)
}
