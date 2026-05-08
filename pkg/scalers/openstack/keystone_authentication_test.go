package openstack

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetServiceURLFallsBackToServiceNameWhenServiceTypeLookupFails(t *testing.T) {
	catalogServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "/v3/auth/catalog", request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, err := writer.Write([]byte(`{"catalog":[{"type":"custom-metric","id":"svc-1","name":"metric","endpoints":[{"url":"https://metrics.example.test/v1","interface":"public","region":"RegionOne","region_id":"RegionOne","id":"ep-1"}]}]}`))
		assert.NoError(t, err)
	}))
	defer catalogServer.Close()

	keystone := &KeystoneAuthRequest{
		AuthURL:           catalogServer.URL,
		HTTPClientTimeout: 5 * time.Second,
		serviceTypesLookup: func(_ context.Context, _ string) ([]string, error) {
			return nil, fmt.Errorf("project is not an official OpenStack project")
		},
	}

	serviceURL, err := keystone.getServiceURL(context.Background(), "token", "metric", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://metrics.example.test/v1", serviceURL)
}

func TestGetServiceURLMatchesProvidedServiceTypeWhenAliasLookupFails(t *testing.T) {
	catalogServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "/v3/auth/catalog", request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, err := writer.Write([]byte(`{"catalog":[{"type":"metric","id":"svc-1","name":"telemetry-metrics","endpoints":[{"url":"https://metrics.example.test/v1","interface":"public","region":"RegionOne","region_id":"RegionOne","id":"ep-1"}]}]}`))
		assert.NoError(t, err)
	}))
	defer catalogServer.Close()

	keystone := &KeystoneAuthRequest{
		AuthURL:           catalogServer.URL,
		HTTPClientTimeout: 5 * time.Second,
		serviceTypesLookup: func(_ context.Context, _ string) ([]string, error) {
			return nil, fmt.Errorf("project is not an official OpenStack project")
		},
	}

	serviceURL, err := keystone.getServiceURL(context.Background(), "token", "metric", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://metrics.example.test/v1", serviceURL)
}
