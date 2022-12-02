package testhelpers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// NewMockServer creates a server to respond to API calls for unit tests
func NewMockServer(t *testing.T, mockJSONResponse string, statusCode int) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		_, err := w.Write([]byte(mockJSONResponse))

		require.NoError(t, err)
	}))

	return ts
}
