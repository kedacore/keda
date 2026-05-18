/*
Copyright 2024 The KEDA Authors

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

package aws

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPClientCloseIdleConnections(t *testing.T) {
	client := NewHTTPClient()

	type closeIdler interface {
		CloseIdleConnections()
	}
	_, ok := client.Transport.(closeIdler)
	assert.True(t, ok, "Transport should implement CloseIdleConnections")

	// Should not panic
	client.CloseIdleConnections()
}

func TestNewHTTPClientBlocksNon307Redirects(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound) // 302
	}))
	defer origin.Close()

	client := NewHTTPClient()
	resp, err := client.Get(origin.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// If the redirect were followed, we'd get 200 from target
	assert.Equal(t, http.StatusFound, resp.StatusCode, "302 should not be followed")
}

func TestNewHTTPClientAllows307Redirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect) // 307
	}))
	defer origin.Close()

	client := NewHTTPClient()
	resp, err := client.Get(origin.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "307 should be followed")
}

// Note on synchronization: httptest handlers write to local variables that are read after
// client.Do returns. This is safe because client.Do blocks until the response is fully
// received, which requires the handler to have completed — establishing happens-before
// without explicit synchronization. This matches the httptest pattern used throughout
// this repository (e.g. azure_pipelines_scaler_test.go, dynatrace_scaler_test.go).

func TestNewHTTPClientStripsSecurityTokenOnCrossHostRedirect(t *testing.T) {
	var receivedToken string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-Amz-Security-Token")
		w.WriteHeader(200)
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect) // 307
	}))
	defer origin.Close()

	client := NewHTTPClient()
	req, err := http.NewRequest("GET", origin.URL, nil)
	require.NoError(t, err)
	req.Header.Set("X-Amz-Security-Token", "secret-token")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The origin and target are different hosts (different ports), so token should be stripped
	assert.Empty(t, receivedToken, "X-Amz-Security-Token should be stripped on cross-host 307 redirect")
}

func TestNewHTTPClientAllows308Redirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusPermanentRedirect) // 308
	}))
	defer origin.Close()

	client := NewHTTPClient()
	resp, err := client.Get(origin.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "308 should be followed")
}

func TestNewHTTPClientPreservesTokenOnSameHostRedirect(t *testing.T) {
	var receivedToken string
	mux := http.NewServeMux()
	mux.HandleFunc("/target", func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-Amz-Security-Token")
		w.WriteHeader(200)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/target", http.StatusTemporaryRedirect) // 307 same host
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewHTTPClient()
	req, err := http.NewRequest("GET", server.URL+"/", nil)
	require.NoError(t, err)
	req.Header.Set("X-Amz-Security-Token", "secret-token")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "secret-token", receivedToken, "X-Amz-Security-Token should be preserved on same-host redirect")
}
