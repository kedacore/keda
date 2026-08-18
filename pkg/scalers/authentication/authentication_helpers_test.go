package authentication

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func TestInsecureOAuthWarning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *Config
		wantWarning bool
	}{
		{
			name:        "oauth with client secret",
			config:      &Config{Modes: []Type{OAuthType}, OAuth: OAuth{ClientSecret: "secret"}},
			wantWarning: false,
		},
		{
			name:        "oauth without client secret but with mTLS",
			config:      &Config{Modes: []Type{OAuthType, TLSAuthType}},
			wantWarning: false,
		},
		{
			name:        "oauth without client secret and without mTLS",
			config:      &Config{Modes: []Type{OAuthType}},
			wantWarning: true,
		},
		{
			name:        "oauth not enabled",
			config:      &Config{Modes: []Type{BearerAuthType}},
			wantWarning: false,
		},
		{
			// Scalers embed the Config as an optional pointer, so it may be nil.
			name:        "no auth config at all",
			config:      nil,
			wantWarning: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			msg := test.config.InsecureOAuthWarning()
			if test.wantWarning {
				assert.Equal(t, InsecureOAuthWarningMessage, msg)
			} else {
				assert.Empty(t, msg)
			}
		})
	}
}

// newTokenServer returns a token endpoint handing out client credentials tokens,
// the number of requests it has served and the form values of the most recent request.
func newTokenServer(t *testing.T) (*httptest.Server, *atomic.Int64, *url.Values) {
	t.Helper()

	var requests atomic.Int64
	lastForm := &url.Values{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if !assert.NoError(t, r.ParseForm()) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		*lastForm = r.Form
		if username, password, ok := r.BasicAuth(); ok {
			lastForm.Set("client_id", username)
			lastForm.Set("client_secret", password)
		}

		w.Header().Set("Content-Type", "application/json")
		// expires_in keeps the token valid for the duration of the test,
		// so that a second request to the endpoint can only come from a cache miss.
		_, err := w.Write([]byte(`{"access_token":"fake_token","token_type":"Bearer","expires_in":3600}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	return server, &requests, lastForm
}

func TestOAuthTokenSource(t *testing.T) {
	t.Parallel()

	server, requests, lastForm := newTokenServer(t)

	config := &Config{
		Modes: []Type{OAuthType},
		OAuth: OAuth{
			OauthTokenURI:  server.URL,
			ClientID:       "my-client",
			ClientSecret:   "my-secret",
			Scopes:         []string{"scope-a", "scope-b"},
			EndpointParams: url.Values{"audience": []string{"my-audience"}},
		},
	}

	tokenSource := config.OAuthTokenSource(t.Context(), server.Client())

	token, err := tokenSource.Token()
	require.NoError(t, err)
	assert.Equal(t, "fake_token", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)

	assert.Equal(t, "client_credentials", lastForm.Get("grant_type"))
	assert.Equal(t, "my-client", lastForm.Get("client_id"))
	assert.Equal(t, "my-secret", lastForm.Get("client_secret"))
	assert.Equal(t, "scope-a scope-b", lastForm.Get("scope"))
	assert.Equal(t, "my-audience", lastForm.Get("audience"))
	assert.Equal(t, int64(1), requests.Load())

	// A cached, unexpired token must not trigger another request to the token endpoint.
	_, err = tokenSource.Token()
	require.NoError(t, err)
	assert.Equal(t, int64(1), requests.Load(), "token should be cached until it expires")
}

func TestOAuthTokenSourceWithoutClientSecret(t *testing.T) {
	server, _, lastForm := newTokenServer(t)

	config := &Config{
		Modes: []Type{OAuthType, TLSAuthType},
		OAuth: OAuth{
			OauthTokenURI: server.URL,
			ClientID:      "my-client",
		},
	}

	_, err := config.OAuthTokenSource(t.Context(), server.Client()).Token()
	require.NoError(t, err)

	// mTLS client authentication (RFC 8705) carries no secret,
	// but Go's OAuth library requires a non-empty one, so a placeholder is sent instead.
	assert.Equal(t, "my-client", lastForm.Get("client_id"))
	assert.Equal(t, unusedClientSecret, lastForm.Get("client_secret"))
}

func TestOAuthTokenSourceUsesGivenClient(t *testing.T) {
	t.Parallel()

	server, requests, _ := newTokenServer(t)

	config := &Config{
		Modes: []Type{OAuthType},
		OAuth: OAuth{OauthTokenURI: server.URL, ClientID: "my-client", ClientSecret: "my-secret"},
	}

	// A client whose transport always fails proves the token request is routed through the client passed in,
	// rather than through http.DefaultClient.
	failing := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, assert.AnError
	})}

	_, err := config.OAuthTokenSource(t.Context(), failing).Token()
	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, int64(0), requests.Load())
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
