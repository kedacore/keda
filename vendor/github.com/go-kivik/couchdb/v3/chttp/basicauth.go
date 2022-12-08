package chttp

import (
	"net/http"
)

// BasicAuth provides HTTP Basic Auth for a client.
type BasicAuth struct {
	Username string
	Password string

	// transport stores the original transport that is overridden by this auth
	// mechanism
	transport http.RoundTripper
}

var _ Authenticator = &BasicAuth{}

// RoundTrip fulfills the http.RoundTripper interface. It sets HTTP Basic Auth
// on outbound requests.
func (a *BasicAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(a.Username, a.Password)
	return a.transport.RoundTrip(req)
}

// Authenticate sets HTTP Basic Auth headers for the client.
func (a *BasicAuth) Authenticate(c *Client) error {
	a.transport = c.Transport
	if a.transport == nil {
		a.transport = http.DefaultTransport
	}
	c.Transport = a
	return nil
}

// JWTAuth provides JWT based auth for a client.
type JWTAuth struct {
	Token string

	transport http.RoundTripper
}

func (a *JWTAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+a.Token)
	return a.transport.RoundTrip(req)
}

func (a *JWTAuth) Authenticate(c *Client) error {
	a.transport = c.Transport
	if a.transport == nil {
		a.transport = http.DefaultTransport
	}
	c.Transport = a
	return nil
}
