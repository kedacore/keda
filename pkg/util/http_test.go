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

// TestCreateHTTPClient_SharesTransport ensures clients built with the same
// unsafeSsl value share a single *http.Transport so all scalers reuse one
// connection pool and DNS cache per TLS mode.
func TestCreateHTTPClient_SharesTransport(t *testing.T) {
	c1 := CreateHTTPClient(time.Second, false)
	c2 := CreateHTTPClient(30*time.Second, false)

	t1, ok := c1.Transport.(*http.Transport)
	assert.True(t, ok, "expected *http.Transport")
	t2, ok := c2.Transport.(*http.Transport)
	assert.True(t, ok, "expected *http.Transport")

	// Same pointer: shared Transport.
	assert.Same(t, t1, t2, "clients with matching unsafeSsl must share a Transport")

	// Timeouts remain independent (per-client).
	assert.NotEqual(t, c1.Timeout, c2.Timeout)
}

// TestCreateHTTPClient_SeparatesByUnsafeSsl ensures unsafeSsl=true and
// unsafeSsl=false get different Transports (they carry different TLS config).
func TestCreateHTTPClient_SeparatesByUnsafeSsl(t *testing.T) {
	cSafe := CreateHTTPClient(time.Second, false)
	cUnsafe := CreateHTTPClient(time.Second, true)

	tSafe := cSafe.Transport.(*http.Transport)
	tUnsafe := cUnsafe.Transport.(*http.Transport)

	assert.NotSame(t, tSafe, tUnsafe, "unsafeSsl variants must use distinct Transports")
	// Sanity: TLS config reflects the mode.
	assert.False(t, tSafe.TLSClientConfig.InsecureSkipVerify)
	assert.True(t, tUnsafe.TLSClientConfig.InsecureSkipVerify)
}

// TestSharedHTTPTransport_Concurrent asserts the internal sharedHTTPTransport
// cache is safe under concurrent access (run under -race). All goroutines
// calling with the same unsafeSsl must observe the same Transport pointer.
func TestSharedHTTPTransport_Concurrent(t *testing.T) {
	const goroutines = 1000
	var (
		wg        sync.WaitGroup
		resultsMu sync.Mutex
		results   = make([]*http.Transport, 0, goroutines)
	)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			tr := sharedHTTPTransport(false)
			resultsMu.Lock()
			results = append(results, tr)
			resultsMu.Unlock()
		}()
	}
	wg.Wait()

	first := results[0]
	for _, tr := range results {
		assert.Same(t, first, tr, "all concurrent calls must return the same Transport")
	}
}

// TestSharedHTTPTransport_PoolSizing sanity-checks that the shared Transport
// carries the connection-pool sizing intended for high-fanout deployments.
// Goal: make sure nobody accidentally reverts to the Go stdlib defaults
// (MaxIdleConnsPerHost=2), which is what caused the original problem.
func TestSharedHTTPTransport_PoolSizing(t *testing.T) {
	tr := sharedHTTPTransport(false)
	assert.GreaterOrEqual(t, tr.MaxIdleConnsPerHost, 100,
		"shared Transport must allow many idle connections per host to avoid re-dial churn")
	assert.Equal(t, sharedTransportIdleConnTimeout, tr.IdleConnTimeout)
}
