//go:build e2e
// +build e2e

package openstack_metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kedacore/keda/v2/pkg/scalers/openstack"
)

type Measure struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type archivePolicy struct {
	Name               string                    `json:"name"`
	AggregationMethods []string                  `json:"aggregation_methods"`
	Definition         []archivePolicyDefinition `json:"definition"`
}

type archivePolicyDefinition struct {
	Granularity interface{} `json:"granularity"`
}

type metric struct {
	ID string `json:"id"`
}

type metricCreateRequest struct {
	ArchivePolicyName string `json:"archive_policy_name"`
	Name              string `json:"name,omitempty"`
}

func CreateMetricsClient(t *testing.T, authURL, userID, password, projectID, metricsURLOverride string) openstack.Client {
	t.Helper()

	keystoneAuth, err := openstack.NewPasswordAuth(authURL, userID, password, projectID, 30)
	require.NoErrorf(t, err, "cannot create keystone auth - %s", err)

	if metricsURLOverride != "" {
		client, clientErr := keystoneAuth.RequestClient(context.Background())
		require.NoErrorf(t, clientErr, "cannot create metrics client token - %s", clientErr)
		client.URL = metricsURLOverride
		return client
	}

	client, err := keystoneAuth.RequestClient(context.Background(), "metric")
	if err != nil {
		t.Skipf("skipping OpenStack metrics test: unable to discover metric service from catalog and OPENSTACK_METRICS_URL is not set: %v", err)
		return openstack.Client{}
	}

	return client
}

func CreateClient(t *testing.T, authURL, userID, password, projectID string) openstack.Client {
	t.Helper()

	keystoneAuth, err := openstack.NewPasswordAuth(authURL, userID, password, projectID, 30)
	require.NoErrorf(t, err, "cannot create keystone auth - %s", err)

	client, err := keystoneAuth.RequestClient(context.Background())
	require.NoErrorf(t, err, "cannot create client - %s", err)

	return client
}

func CreateMetric(t *testing.T, client openstack.Client, metricName string) (string, string) {
	t.Helper()

	metricsURL := metricBaseURL(t, client.URL)
	archivePolicyName := getCompatibleArchivePolicyName(t, client, metricsURL)
	requestBody, err := json.Marshal(metricCreateRequest{
		ArchivePolicyName: archivePolicyName,
		Name:              metricName,
	})
	require.NoErrorf(t, err, "cannot marshal metric request - %s", err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, metricsURL, bytes.NewBuffer(requestBody))
	require.NoErrorf(t, err, "cannot create metric request - %s", err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", client.Token)

	resp, err := client.HTTPClient.Do(req)
	require.NoErrorf(t, err, "cannot create metric - %s", err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode, "failed to create metric")

	createdMetric := metric{}
	require.NoErrorf(t, json.NewDecoder(resp.Body).Decode(&createdMetric), "cannot decode metric response")
	require.NotEmpty(t, createdMetric.ID, "created metric id should not be empty")

	return metricsURL, createdMetric.ID
}

func DeleteMetric(t *testing.T, client openstack.Client, metricsURL, metricID string) {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, joinURL(t, metricsURL, metricID), nil)
	if err != nil {
		assert.NoErrorf(t, err, "cannot create delete metric request - %s", err)
		return
	}

	req.Header.Set("X-Auth-Token", client.Token)

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		assert.NoErrorf(t, err, "cannot delete metric - %s", err)
		return
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "failed to delete metric")
}

func PostMeasure(t *testing.T, client openstack.Client, metricsURL, metricID string, value float64) {
	t.Helper()

	measure := []Measure{
		{
			Timestamp: time.Now().Format(time.RFC3339),
			Value:     value,
		},
	}

	measuresJSON, err := json.Marshal(measure)
	require.NoErrorf(t, err, "cannot marshal measures - %s", err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, joinURL(t, metricsURL, metricID, "measures"), bytes.NewBuffer(measuresJSON))
	require.NoErrorf(t, err, "cannot create request - %s", err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", client.Token)

	httpClient := client.HTTPClient
	resp, err := httpClient.Do(req)
	require.NoErrorf(t, err, "cannot post measure - %s", err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusAccepted, resp.StatusCode, "failed to post measure")
}

func getCompatibleArchivePolicyName(t *testing.T, client openstack.Client, metricsURL string) string {
	t.Helper()

	archivePoliciesURL := joinURL(t, metricsURL, "..", "archive_policy")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, archivePoliciesURL, nil)
	require.NoErrorf(t, err, "cannot create archive policy request - %s", err)

	req.Header.Set("X-Auth-Token", client.Token)

	resp, err := client.HTTPClient.Do(req)
	require.NoErrorf(t, err, "cannot list archive policies - %s", err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "failed to list archive policies")

	var archivePolicies []archivePolicy
	require.NoErrorf(t, json.NewDecoder(resp.Body).Decode(&archivePolicies), "cannot decode archive policy response")

	for _, policy := range archivePolicies {
		if !supportsAggregation(policy.AggregationMethods, "mean") {
			continue
		}

		for _, definition := range policy.Definition {
			if supportsGranularity(definition.Granularity, 300) {
				return policy.Name
			}
		}
	}

	require.FailNow(t, "no compatible archive policy found", "expected an archive policy with mean aggregation and 300-second granularity")
	return ""
}

func supportsAggregation(aggregationMethods []string, expected string) bool {
	for _, aggregationMethod := range aggregationMethods {
		if aggregationMethod == expected {
			return true
		}
	}

	return false
}

// parseGranularitySeconds parses a granularity string (e.g., "5 minutes", "300 seconds", "00:05:00", "300", "5m") into seconds.
func parseGranularitySeconds(granularity string) (int, bool) {
	trimmed := strings.TrimSpace(strings.ToLower(granularity))
	// Try parsing as float seconds directly
	if val, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return int(val), true
	}
	// Try parsing formats like "5 minutes", "300 seconds"
	fields := strings.Fields(trimmed)
	if len(fields) == 2 {
		value, err := strconv.ParseFloat(fields[0], 64)
		if err == nil {
			switch fields[1] {
			case "second", "seconds":
				return int(value), true
			case "minute", "minutes":
				return int(value * 60), true
			case "hour", "hours":
				return int(value * 3600), true
			}
		}
	}
	// Try parsing HH:MM:SS
	if strings.Count(trimmed, ":") == 2 {
		parts := strings.Split(trimmed, ":")
		hours, errH := strconv.Atoi(parts[0])
		minutes, errM := strconv.Atoi(parts[1])
		seconds, errS := strconv.Atoi(parts[2])
		if errH == nil && errM == nil && errS == nil {
			return hours*3600 + minutes*60 + seconds, true
		}
	}
	// Try Go duration string (e.g., "5m", "300s")
	if d, err := time.ParseDuration(trimmed); err == nil {
		return int(d.Seconds()), true
	}
	return 0, false
}

func supportsGranularity(rawGranularity interface{}, expectedSeconds int) bool {
	switch granularity := rawGranularity.(type) {
	case float64:
		return int(granularity) == expectedSeconds
	case string:
		if seconds, ok := parseGranularitySeconds(granularity); ok {
			return seconds == expectedSeconds
		}
	}
	return false
}

func metricBaseURL(t *testing.T, serviceURL string) string {
	t.Helper()

	parsedURL, err := url.Parse(serviceURL)
	require.NoErrorf(t, err, "metric service URL is invalid - %s", err)

	cleanPath := strings.TrimSuffix(parsedURL.Path, "/")
	switch {
	case strings.HasSuffix(cleanPath, "/v1/metric"):
		return parsedURL.String()
	case strings.HasSuffix(cleanPath, "/v1"):
		parsedURL.Path = path.Join(parsedURL.Path, "metric")
	default:
		parsedURL.Path = path.Join(parsedURL.Path, "v1", "metric")
	}

	return parsedURL.String()
}

func joinURL(t *testing.T, rawURL string, elems ...string) string {
	t.Helper()

	parsedURL, err := url.Parse(rawURL)
	require.NoErrorf(t, err, "url is invalid - %s", err)

	joinedPath := parsedURL.Path
	for _, elem := range elems {
		joinedPath = path.Join(joinedPath, elem)
	}
	parsedURL.Path = joinedPath

	return parsedURL.String()
}
