package chttp

import (
	"net/http/cookiejar"

	"golang.org/x/net/publicsuffix"
)

// Authenticator is an interface that provides authentication to a server.
type Authenticator interface {
	Authenticate(*Client) error
}

func (a *CookieAuth) setCookieJar() {
	// If a jar is already set, just use it
	if a.client.Jar != nil {
		return
	}
	// cookiejar.New never returns an error
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	a.client.Jar = jar
}
