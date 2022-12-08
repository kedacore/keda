package chttp

import (
	"context"
	"net/http"
	"time"

	kivik "github.com/go-kivik/kivik/v3"
)

// CookieAuth provides CouchDB Cookie auth services as described at
// http://docs.couchdb.org/en/2.0.0/api/server/authn.html#cookie-authentication
//
// CookieAuth stores authentication state after use, so should not be re-used.
type CookieAuth struct {
	Username string `json:"name"`
	Password string `json:"password"`

	client *Client
	// transport stores the original transport that is overridden by this auth
	// mechanism
	transport http.RoundTripper
}

var _ Authenticator = &CookieAuth{}

// Authenticate initiates a session with the CouchDB server.
func (a *CookieAuth) Authenticate(c *Client) error {
	a.client = c
	a.setCookieJar()
	a.transport = c.Transport
	if a.transport == nil {
		a.transport = http.DefaultTransport
	}
	c.Transport = a
	return nil
}

// shouldAuth returns true if there is no cookie set, or if it has expired.
func (a *CookieAuth) shouldAuth(req *http.Request) bool {
	if _, err := req.Cookie(kivik.SessionCookieName); err == nil {
		return false
	}
	cookie := a.Cookie()
	if cookie == nil {
		return true
	}
	if !cookie.Expires.IsZero() {
		return cookie.Expires.Before(time.Now().Add(time.Minute))
	}
	// If we get here, it means the server did not include an expiry time in
	// the session cookie. Some CouchDB configurations do this, but rather than
	// re-authenticating for every request, we'll let the session expire. A
	// future change might be to make a client-configurable option to set the
	// re-authentication timeout.
	return false
}

// Cookie returns the current session cookie if found, or nil if not.
func (a *CookieAuth) Cookie() *http.Cookie {
	if a.client == nil {
		return nil
	}
	for _, cookie := range a.client.Jar.Cookies(a.client.dsn) {
		if cookie.Name == kivik.SessionCookieName {
			return cookie
		}
	}
	return nil
}

var authInProgress = &struct{ name string }{"in progress"}

// RoundTrip fulfills the http.RoundTripper interface. It sets
// (re-)authenticates when the cookie has expired or is not yet set.
// It also drops the auth cookie if we receive a 401 response to ensure
// that follow up requests can try to authenticate again.
func (a *CookieAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := a.authenticate(req); err != nil {
		return nil, err
	}

	res, err := a.transport.RoundTrip(req)
	if err != nil {
		return res, err
	}

	if res != nil && res.StatusCode == http.StatusUnauthorized {
		if cookie := a.Cookie(); cookie != nil {
			// set to expire yesterday to allow us to ditch it
			cookie.Expires = time.Now().AddDate(0, 0, -1)
			a.client.Jar.SetCookies(a.client.dsn, []*http.Cookie{cookie})
		}
	}
	return res, nil
}

func (a *CookieAuth) authenticate(req *http.Request) error {
	ctx := req.Context()
	if inProg, _ := ctx.Value(authInProgress).(bool); inProg {
		return nil
	}
	if !a.shouldAuth(req) {
		return nil
	}
	a.client.authMU.Lock()
	defer a.client.authMU.Unlock()
	if c := a.Cookie(); c != nil {
		// In case another simultaneous process authenticated successfully first
		req.AddCookie(c)
		return nil
	}
	ctx = context.WithValue(ctx, authInProgress, true)
	opts := &Options{
		GetBody: BodyEncoder(a),
		Header: http.Header{
			HeaderIdempotencyKey: []string{},
		},
	}
	if _, err := a.client.DoError(ctx, http.MethodPost, "/_session", opts); err != nil {
		return err
	}
	if c := a.Cookie(); c != nil {
		req.AddCookie(c)
	}
	return nil
}
