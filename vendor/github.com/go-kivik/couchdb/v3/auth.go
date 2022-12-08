package couchdb

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-kivik/couchdb/v3/chttp"
	"github.com/go-kivik/kivik/v3"
)

func (c *client) Authenticate(ctx context.Context, a interface{}) error {
	if auth, ok := a.(chttp.Authenticator); ok {
		return auth.Authenticate(c.Client)
	}
	if auth, ok := a.(Authenticator); ok {
		return auth.auth(ctx, c)
	}
	return &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: errors.New("kivik: invalid authenticator")}
}

// Authenticator is a CouchDB authenticator. Direct use of the Authenticator
// interface is for advanced usage. Typically, it is sufficient to provide
// a username and password in the connecting DSN to perform authentication.
// Only use one of these provided authenticators if you have specific, special
// needs.
type Authenticator interface {
	auth(context.Context, *client) error
}

type xportAuth struct {
	http.RoundTripper
}

var _ Authenticator = &xportAuth{}

func (a *xportAuth) auth(_ context.Context, c *client) error {
	if c.Client.Client.Transport != nil {
		return &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: errors.New("kivik: HTTP client transport already set")}
	}
	c.Client.Client.Transport = a.RoundTripper
	return nil
}

// SetTransport returns an authenticator that can be used to set a client
// connection's HTTP Transport. This can be used to control proxies, TLS
// configuration, keep-alives, compression, etc.
//
// Example:
//
//     setXport := couchdb.SetTransport(&http.Transport{
//         // .. custom config
//     })
//     client, _ := kivik.New( ... )
//     client.Authenticate(setXport)
func SetTransport(t http.RoundTripper) Authenticator {
	return &xportAuth{t}
}

type authFunc func(context.Context, *client) error

func (a authFunc) auth(ctx context.Context, c *client) error {
	return a(ctx, c)
}

// BasicAuth provides support for HTTP Basic authentication.
func BasicAuth(user, password string) Authenticator {
	auth := chttp.BasicAuth{Username: user, Password: password}
	return authFunc(func(ctx context.Context, c *client) error {
		return auth.Authenticate(c.Client)
	})
}

// CookieAuth provides support for CouchDB cookie-based authentication.
func CookieAuth(user, password string) Authenticator {
	auth := chttp.CookieAuth{Username: user, Password: password}
	return authFunc(func(ctx context.Context, c *client) error {
		return auth.Authenticate(c.Client)
	})
}

// JWTAuth provides support for CouchDB JWT-based authentication. Kivik does
// no validation on the JWT token; it is passed verbatim to the server.
//
// See https://docs.couchdb.org/en/latest/api/server/authn.html#jwt-authentication
func JWTAuth(token string) Authenticator {
	auth := chttp.JWTAuth{Token: token}
	return authFunc(func(ctx context.Context, c *client) error {
		return auth.Authenticate(c.Client)
	})
}

// ProxyAuth provides support for Proxy authentication.
//
// The `secret` argument represents the `couch_httpd_auth/secret` value
// configured on the CouchDB server. See https://docs.couchdb.org/en/stable/config/auth.html#couch_httpd_auth/secret
// If `secret` is the empty string, the X-Auth-CouchDB-Token header will not be
// set, to support disabling the `proxy_use_secret` server setting. See https://docs.couchdb.org/en/stable/config/auth.html#couch_httpd_auth/proxy_use_secret
//
// The optional `headers` map may be passed to use non-standard header names.
// For instance, to use `X-User` in place of the `X-Auth-CouchDB-Username`
// header, pass a value of {"X-Auth-CouchDB-UserName": "X-User"}.
// The relevant headers are X-Auth-CouchDB-UserName, X-Auth-CouchDB-Roles, and
// X-Auth-CouchDB-Token.
//
// See https://docs.couchdb.org/en/stable/api/server/authn.html?highlight=proxy%20auth#proxy-authentication
func ProxyAuth(user, secret string, roles []string, headers ...map[string]string) Authenticator {
	headerOverrides := http.Header{}
	for _, h := range headers {
		for k, v := range h {
			headerOverrides.Set(k, v)
		}
	}
	auth := chttp.ProxyAuth{Username: user, Secret: secret, Roles: roles, Headers: headerOverrides}
	return authFunc(func(ctx context.Context, c *client) error {
		return auth.Authenticate(c.Client)
	})
}

type rawCookie struct {
	cookie *http.Cookie
	next   http.RoundTripper
}

var (
	_ Authenticator     = &rawCookie{}
	_ http.RoundTripper = &rawCookie{}
)

func (a *rawCookie) auth(_ context.Context, c *client) error {
	if c.Client.Client.Transport != nil {
		return &kivik.Error{HTTPStatus: http.StatusBadRequest, Err: errors.New("kivik: HTTP client transport already set")}
	}
	a.next = c.Client.Client.Transport
	if a.next == nil {
		a.next = http.DefaultTransport
	}
	c.Client.Client.Transport = a
	return nil
}

func (a *rawCookie) RoundTrip(r *http.Request) (*http.Response, error) {
	r.AddCookie(a.cookie)
	return a.next.RoundTrip(r)
}

// SetCookie adds cookie to all outbound requests. This is useful when using
// kivik as a proxy.
func SetCookie(cookie *http.Cookie) Authenticator {
	return &rawCookie{cookie: cookie}
}
