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

package kafka

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func testJWT(t *testing.T, issuedAt, expiresAt time.Time) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}).SignedString([]byte("test-only")) // nosemgrep: go.jwt-go.security.jwt.hardcoded-jwt-key
	require.NoError(t, err)
	return token
}

type testProvider struct {
	*staticTokenProvider
	calls           *atomic.Int32
	missingDeadline *atomic.Bool
	at              func(time.Duration)
}

// newTestProvider returns a provider whose clock starts at t0 and is moved by at.
func newTestProvider(t0 time.Time, token string, refresh func() (string, error)) testProvider {
	tp := testProvider{calls: atomic.NewInt32(0), missingDeadline: atomic.NewBool(false)}
	var refreshFn func(context.Context) (string, error)
	if refresh != nil {
		refreshFn = func(ctx context.Context) (string, error) {
			tp.calls.Inc()
			if _, ok := ctx.Deadline(); !ok {
				tp.missingDeadline.Store(true)
			}
			return refresh()
		}
	}
	now := t0
	tp.staticTokenProvider = OAuthStaticTokenProvider(token, map[string]string{"foo": "bar"}, refreshFn).(*staticTokenProvider)
	tp.now = func() time.Time { return now }
	tp.at = func(d time.Duration) { now = t0.Add(d) }
	return tp
}

func (tp testProvider) requireToken(t *testing.T, expected string, expectedCalls int32) {
	t.Helper()
	accessToken, err := tp.Token()
	require.NoError(t, err)
	require.Equal(t, expected, accessToken.Token)
	require.Equal(t, map[string]string{"foo": "bar"}, accessToken.Extensions)
	require.Equal(t, expectedCalls, tp.calls.Load())
	require.False(t, tp.missingDeadline.Load(), "refresh must be bounded by a timeout")
}

func TestStaticTokenProviderRefreshesAtRefreshWindow(t *testing.T) {
	t0 := time.Now().Truncate(time.Second)
	token := testJWT(t, t0, t0.Add(time.Hour))
	renewed := testJWT(t, t0.Add(48*time.Minute), t0.Add(108*time.Minute))
	tp := newTestProvider(t0, token, func() (string, error) { return renewed, nil })

	tp.at(47 * time.Minute)
	tp.requireToken(t, token, 0)

	tp.at(48 * time.Minute)
	tp.requireToken(t, renewed, 1)

	// The renewed token is not refreshed again until its own refresh window
	tp.at(95 * time.Minute)
	tp.requireToken(t, renewed, 1)
	tp.at(96 * time.Minute)
	tp.requireToken(t, renewed, 2)
}

func TestStaticTokenProviderRefreshesTokenAlreadyDueAtCreation(t *testing.T) {
	t0 := time.Now().Truncate(time.Second)
	renewed := testJWT(t, t0, t0.Add(time.Hour))
	token := testJWT(t, t0.Add(-50*time.Minute), t0.Add(10*time.Minute))
	tp := newTestProvider(t0, token, func() (string, error) { return renewed, nil })
	tp.requireToken(t, renewed, 1)
}

func TestStaticTokenProviderRetriesFailedRefreshAfterDelay(t *testing.T) {
	for name, refresh := range map[string]func() (string, error){
		"error":       func() (string, error) { return "", errors.New("boom") },
		"empty token": func() (string, error) { return "", nil },
	} {
		t.Run(name, func(t *testing.T) {
			t0 := time.Now().Truncate(time.Second)
			token := testJWT(t, t0, t0.Add(time.Hour))
			tp := newTestProvider(t0, token, refresh)

			tp.at(48 * time.Minute)
			tp.requireToken(t, token, 1)
			tp.at(48*time.Minute + 9*time.Second)
			tp.requireToken(t, token, 1)
			tp.at(48*time.Minute + 10*time.Second)
			tp.requireToken(t, token, 2)
		})
	}
}

func TestStaticTokenProviderLimitsRefreshOfTokenAlreadyDue(t *testing.T) {
	// e.g. a JWT in a Secret that has not been rotated yet
	t0 := time.Now().Truncate(time.Second)
	token := testJWT(t, t0, t0.Add(time.Hour))
	tp := newTestProvider(t0, token, func() (string, error) { return token, nil })

	tp.at(48 * time.Minute)
	tp.requireToken(t, token, 1)
	tp.at(48*time.Minute + 59*time.Second)
	tp.requireToken(t, token, 1)
	tp.at(49 * time.Minute)
	tp.requireToken(t, token, 2)
}

func TestStaticTokenProviderRefreshesTokenWithUnregisteredSigningMethod(t *testing.T) {
	t0 := time.Now().Truncate(time.Second)
	encode := func(v string) string { return base64.RawURLEncoding.EncodeToString([]byte(v)) }
	token := encode(`{"alg":"ES256K","typ":"JWT"}`) + "." +
		encode(fmt.Sprintf(`{"iat":%d,"exp":%d}`, t0.Unix(), t0.Add(time.Hour).Unix())) + "." + encode("signature")
	renewed := testJWT(t, t0.Add(48*time.Minute), t0.Add(108*time.Minute))
	tp := newTestProvider(t0, token, func() (string, error) { return renewed, nil })

	tp.at(48 * time.Minute)
	tp.requireToken(t, renewed, 1)
}

func TestStaticTokenProviderWithoutRefresh(t *testing.T) {
	t0 := time.Now().Truncate(time.Second)
	jwtToken := testJWT(t, t0, t0.Add(time.Hour))

	t.Run("non-JWT token is never refreshed", func(t *testing.T) {
		tp := newTestProvider(t0, "opaque-token", func() (string, error) { return jwtToken, nil })
		tp.at(10 * time.Hour)
		tp.requireToken(t, "opaque-token", 0)
	})

	t.Run("no refresh function", func(t *testing.T) {
		tp := newTestProvider(t0, jwtToken, nil)
		tp.at(2 * time.Hour)
		tp.requireToken(t, jwtToken, 0)
	})
}

func TestStaticTokenProviderConcurrentHandshakesRefreshOnce(t *testing.T) {
	t0 := time.Now().Truncate(time.Second)
	token := testJWT(t, t0, t0.Add(time.Hour))
	renewed := testJWT(t, t0.Add(48*time.Minute), t0.Add(108*time.Minute))
	tp := newTestProvider(t0, token, func() (string, error) { return renewed, nil })
	tp.at(48 * time.Minute)

	tokens := make([]string, 10)
	var wg sync.WaitGroup
	for i := range tokens {
		wg.Go(func() {
			if accessToken, err := tp.Token(); err == nil {
				tokens[i] = accessToken.Token
			}
		})
	}
	wg.Wait()
	for _, token := range tokens {
		require.Equal(t, renewed, token)
	}
	require.Equal(t, int32(1), tp.calls.Load())
}
