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

// tokenRequest records what the token endpoint received on its most recent request.
type tokenRequest struct {
	// form holds the request's form values, with any credentials from the Authorization header folded in,
	// so that assertions on client_id and client_secret hold for both OAuth auth styles.
	form url.Values
	// authorization is the raw Authorization header, empty when the request carried none.
	authorization string
}

// newTokenServer returns a token endpoint handing out client credentials tokens,
// the number of requests it has served and the most recent request it received.
func newTokenServer(t *testing.T) (*httptest.Server, *atomic.Int64, *tokenRequest) {
	t.Helper()

	var requests atomic.Int64
	last := &tokenRequest{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if !assert.NoError(t, r.ParseForm()) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		*last = tokenRequest{form: r.Form, authorization: r.Header.Get("Authorization")}
		if username, password, ok := r.BasicAuth(); ok {
			last.form.Set("client_id", username)
			last.form.Set("client_secret", password)
		}

		w.Header().Set("Content-Type", "application/json")
		// expires_in keeps the token valid for the duration of the test,
		// so that a second request to the endpoint can only come from a cache miss.
		_, err := w.Write([]byte(`{"access_token":"fake_token","token_type":"Bearer","expires_in":3600}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	return server, &requests, last
}

func TestOAuthTokenSource(t *testing.T) {
	t.Parallel()

	server, requests, last := newTokenServer(t)

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

	assert.Equal(t, "client_credentials", last.form.Get("grant_type"))
	assert.Equal(t, "my-client", last.form.Get("client_id"))
	assert.Equal(t, "my-secret", last.form.Get("client_secret"))
	assert.Equal(t, "scope-a scope-b", last.form.Get("scope"))
	assert.Equal(t, "my-audience", last.form.Get("audience"))
	assert.Equal(t, int64(1), requests.Load())

	// A cached, unexpired token must not trigger another request to the token endpoint.
	_, err = tokenSource.Token()
	require.NoError(t, err)
	assert.Equal(t, int64(1), requests.Load(), "token should be cached until it expires")
}

func TestOAuthTokenSourceWithoutClientSecret(t *testing.T) {
	server, _, last := newTokenServer(t)

	config := &Config{
		Modes: []Type{OAuthType, TLSAuthType},
		OAuth: OAuth{
			OauthTokenURI: server.URL,
			ClientID:      "my-client",
		},
	}

	_, err := config.OAuthTokenSource(t.Context(), server.Client()).Token()
	require.NoError(t, err)

	// mTLS client authentication (RFC 8705) identifies the client by its certificate,
	// so the token request must carry the client ID but no second client authentication method:
	// neither a client_secret parameter nor an Authorization header.
	assert.Equal(t, "my-client", last.form.Get("client_id"))
	assert.Empty(t, last.form.Get("client_secret"))
	assert.Empty(t, last.authorization)
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
