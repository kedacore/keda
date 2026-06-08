//go:build e2e
// +build e2e

package openstack_metrics

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kedacore/keda/v2/pkg/scalers/openstack"
)

func TestResolveMetricsClientUsesOverrideURL(t *testing.T) {
	t.Parallel()

	var requestedServices [][]string
	client, err := resolveMetricsClient(context.Background(), func(_ context.Context, serviceProps ...string) (openstack.Client, error) {
		requestedServices = append(requestedServices, append([]string(nil), serviceProps...))
		return openstack.Client{URL: "https://catalog.example/v1"}, nil
	}, "https://override.example/v1")

	require.NoError(t, err)
	assert.Equal(t, "https://override.example/v1", client.URL)
	require.Len(t, requestedServices, 1)
	assert.Empty(t, requestedServices[0])
}

func TestResolveMetricsClientFallsBackToGnocchi(t *testing.T) {
	t.Parallel()

	var requestedServices [][]string
	client, err := resolveMetricsClient(context.Background(), func(_ context.Context, serviceProps ...string) (openstack.Client, error) {
		requestedServices = append(requestedServices, append([]string(nil), serviceProps...))
		if len(serviceProps) == 1 && serviceProps[0] == "metric" {
			return openstack.Client{}, errors.New("service 'metric' not found in catalog")
		}
		if len(serviceProps) == 1 && serviceProps[0] == "gnocchi" {
			return openstack.Client{URL: "https://gnocchi.example/v1"}, nil
		}
		return openstack.Client{}, errors.New("unexpected request")
	}, "")

	require.NoError(t, err)
	assert.Equal(t, "https://gnocchi.example/v1", client.URL)
	assert.Equal(t, [][]string{{"metric"}, {"gnocchi"}}, requestedServices)
}

func TestResolveMetricsClientReturnsBothLookupErrors(t *testing.T) {
	t.Parallel()

	_, err := resolveMetricsClient(context.Background(), func(_ context.Context, serviceProps ...string) (openstack.Client, error) {
		if len(serviceProps) == 1 && serviceProps[0] == "metric" {
			return openstack.Client{}, errors.New("metric lookup failed")
		}
		if len(serviceProps) == 1 && serviceProps[0] == "gnocchi" {
			return openstack.Client{}, errors.New("gnocchi lookup failed")
		}
		return openstack.Client{}, errors.New("unexpected request")
	}, "")

	require.Error(t, err)
	assert.ErrorContains(t, err, "metric lookup failed")
	assert.ErrorContains(t, err, "gnocchi lookup failed")
}
