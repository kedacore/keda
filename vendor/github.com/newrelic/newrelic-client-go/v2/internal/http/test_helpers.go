package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mock "github.com/newrelic/newrelic-client-go/v2/pkg/testhelpers"
)

// NewTestAPIClient returns a test Client instance that is configured to communicate with a mock server.
func NewTestAPIClient(t *testing.T, handler http.Handler) Client {
	ts := httptest.NewServer(handler)
	tc := mock.NewTestConfig(t, ts)

	return NewClient(tc)
}
