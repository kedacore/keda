package util

import (
	"net/http"
	"time"
)

// HTTPDoer is an interface that matches the Do method on
// (net/http).Client. It should be used in function signatures
// instead of raw *http.Clients wherever possible
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// CreateHTTPClient returns a new HTTP client with the timeout set to
// timeoutMS milliseconds, or 300 milliseconds if timeoutMS <= 0
func CreateHTTPClient(timeout time.Duration) *http.Client {
	// default the timeout to 300ms
	if timeout <= 0 {
		timeout = 300 * time.Millisecond
	}
	httpClient := &http.Client{
		Timeout: timeout,
	}

	return httpClient
}
