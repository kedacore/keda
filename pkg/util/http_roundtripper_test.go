/*
Copyright 2026 The KEDA Authors

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
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRoundTripper struct {
	resp *http.Response
	err  error
}

func (m *mockRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return m.resp, m.err
}

func fakeResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func newRequest(ctx context.Context) *http.Request {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	return req
}

func TestInstrumentedRoundTripper_SuccessfulRequest(t *testing.T) {
	for _, statusCode := range []int{200, 201, 301, 400, 500} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			rt := NewInstrumentedRoundTripper(&mockRoundTripper{resp: fakeResponse(statusCode)})

			resp, err := rt.RoundTrip(newRequest(context.Background()))

			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, statusCode, resp.StatusCode)
		})
	}
}

func TestInstrumentedRoundTripper_TransportError(t *testing.T) {
	transportErr := errors.New("connection refused")
	rt := NewInstrumentedRoundTripper(&mockRoundTripper{err: transportErr})

	resp, err := rt.RoundTrip(newRequest(context.Background())) //nolint:bodyclose // resp is nil on error

	assert.ErrorIs(t, err, transportErr)
	assert.Nil(t, resp)
}

func TestInstrumentedRoundTripper_ResponseReturnedUnmodified(t *testing.T) {
	expected := fakeResponse(202)
	rt := NewInstrumentedRoundTripper(&mockRoundTripper{resp: expected})

	got, err := rt.RoundTrip(newRequest(context.Background()))

	require.NoError(t, err)
	defer got.Body.Close()
	assert.Same(t, expected, got)
}

func TestInstrumentedRoundTripper_NilNextUsesDefault(t *testing.T) {
	rt := NewInstrumentedRoundTripper(nil)
	irt, ok := rt.(*InstrumentedRoundTripper)
	require.True(t, ok)
	assert.Equal(t, http.DefaultTransport, irt.next)
}

func TestInstrumentedRoundTripper_ScalerContextKey_Missing(t *testing.T) {
	// When one or more of the five required context keys are absent, the
	// RoundTripper should not panic, complete normally, and skip metric recording.
	rt := NewInstrumentedRoundTripper(&mockRoundTripper{resp: fakeResponse(200)})

	resp, err := rt.RoundTrip(newRequest(context.Background()))

	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
}

func TestInstrumentedRoundTripper_AllContextKeys(t *testing.T) {
	// When all five required context keys are present the RoundTripper should
	// complete normally and forward the request to the underlying transport.
	rt := NewInstrumentedRoundTripper(&mockRoundTripper{resp: fakeResponse(200)})

	ctx := context.Background()
	ctx = context.WithValue(ctx, ScalerContextKey, "prometheus")
	ctx = context.WithValue(ctx, TriggerNameContextKey, "my-trigger")
	ctx = context.WithValue(ctx, MetricNameContextKey, "my-metric")
	ctx = context.WithValue(ctx, NamespaceContextKey, "default")
	ctx = context.WithValue(ctx, ScaledResourceContextKey, "my-so")
	resp, err := rt.RoundTrip(newRequest(ctx))

	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
}

func TestCreateHTTPClient_TransportIsInstrumented(t *testing.T) {
	client := CreateHTTPClient(0, false)
	_, ok := client.Transport.(*InstrumentedRoundTripper)
	assert.True(t, ok, "expected CreateHTTPClient to wrap transport with InstrumentedRoundTripper")
}

func TestCreateHTTPTransportWithTLSConfig_IsInstrumented(t *testing.T) {
	rt := CreateHTTPTransportWithTLSConfig(nil)
	_, ok := rt.(*InstrumentedRoundTripper)
	assert.True(t, ok, "expected CreateHTTPTransportWithTLSConfig to return an InstrumentedRoundTripper")
}
