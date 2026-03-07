/*
Copyright 2025 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scalers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type ionosMonitoringMetadataTestData struct {
	metadata  map[string]string
	authParams map[string]string
	errorCase bool
}

type ionosMonitoringMetricIdentifier struct {
	metadataTestData *ionosMonitoringMetadataTestData
	triggerIndex     int
	name             string
}

var testIONOSMonitoringMetadata = []ionosMonitoringMetadataTestData{
	// empty config
	{map[string]string{}, map[string]string{}, true},
	// all required fields supplied
	{
		map[string]string{"host": "http://dummy.monitoring.de-txl.ionos.com", "query": "up", "threshold": "10"},
		map[string]string{"apiKey": "secret"},
		false,
	},
	// with optional activationThreshold
	{
		map[string]string{"host": "http://dummy.monitoring.de-txl.ionos.com", "query": "up", "threshold": "10", "activationThreshold": "5"},
		map[string]string{"apiKey": "secret"},
		false,
	},
	// ignoreNullValues=false
	{
		map[string]string{"host": "http://dummy.monitoring.de-txl.ionos.com", "query": "up", "threshold": "10", "ignoreNullValues": "false"},
		map[string]string{"apiKey": "secret"},
		false,
	},
	// missing host
	{
		map[string]string{"query": "up", "threshold": "10"},
		map[string]string{"apiKey": "secret"},
		true,
	},
	// missing query
	{
		map[string]string{"host": "http://dummy.monitoring.de-txl.ionos.com", "threshold": "10"},
		map[string]string{"apiKey": "secret"},
		true,
	},
	// missing threshold
	{
		map[string]string{"host": "http://dummy.monitoring.de-txl.ionos.com", "query": "up"},
		map[string]string{"apiKey": "secret"},
		true,
	},
	// missing apiKey
	{
		map[string]string{"host": "http://dummy.monitoring.de-txl.ionos.com", "query": "up", "threshold": "10"},
		map[string]string{},
		true,
	},
	// malformed threshold
	{
		map[string]string{"host": "http://dummy.monitoring.de-txl.ionos.com", "query": "up", "threshold": "abc"},
		map[string]string{"apiKey": "secret"},
		true,
	},
	// malformed activationThreshold
	{
		map[string]string{"host": "http://dummy.monitoring.de-txl.ionos.com", "query": "up", "threshold": "10", "activationThreshold": "abc"},
		map[string]string{"apiKey": "secret"},
		true,
	},
}

var ionosMonitoringMetricIdentifiers = []ionosMonitoringMetricIdentifier{
	{&testIONOSMonitoringMetadata[1], 0, "s0-ionos-monitoring"},
	{&testIONOSMonitoringMetadata[1], 1, "s1-ionos-monitoring"},
}

func TestIONOSMonitoringParseMetadata(t *testing.T) {
	for _, testData := range testIONOSMonitoringMetadata {
		_, err := parseIONOSMonitoringMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadata,
			AuthParams:      testData.authParams,
		})
		if err != nil && !testData.errorCase {
			t.Errorf("expected success but got error: %v (metadata: %v)", err, testData.metadata)
		}
		if testData.errorCase && err == nil {
			t.Errorf("expected error but got success (metadata: %v)", testData.metadata)
		}
	}
}

func TestIONOSMonitoringGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range ionosMonitoringMetricIdentifiers {
		meta, err := parseIONOSMonitoringMetadata(&scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadataTestData.metadata,
			AuthParams:      testData.metadataTestData.authParams,
			TriggerIndex:    testData.triggerIndex,
		})
		require.NoError(t, err, "could not parse metadata")

		scaler := ionosMonitoringScaler{metadata: meta, httpClient: nil}
		metricSpec := scaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		assert.Equal(t, testData.name, metricName, "unexpected metric name")
	}
}

func TestIONOSMonitoringExecuteQuery(t *testing.T) {
	testCases := []struct {
		name             string
		serverStatusCode int
		responseBody     interface{}
		ignoreNullValues bool
		expectedValue    float64
		expectError      bool
	}{
		{
			name:             "successful scalar result",
			serverStatusCode: http.StatusOK,
			responseBody: ionosPromQueryResult{
				Status: "success",
				Data: struct {
					ResultType string `json:"resultType"`
					Result     []struct {
						Metric struct{}      `json:"metric"`
						Value  []interface{} `json:"value"`
					} `json:"result"`
				}{
					ResultType: "vector",
					Result: []struct {
						Metric struct{}      `json:"metric"`
						Value  []interface{} `json:"value"`
					}{
						{Value: []interface{}{float64(1234567890), "42.5"}},
					},
				},
			},
			ignoreNullValues: true,
			expectedValue:    42.5,
			expectError:      false,
		},
		{
			name:             "empty result ignoreNullValues=true returns 0",
			serverStatusCode: http.StatusOK,
			responseBody: ionosPromQueryResult{
				Status: "success",
				Data: struct {
					ResultType string `json:"resultType"`
					Result     []struct {
						Metric struct{}      `json:"metric"`
						Value  []interface{} `json:"value"`
					} `json:"result"`
				}{ResultType: "vector"},
			},
			ignoreNullValues: true,
			expectedValue:    0,
			expectError:      false,
		},
		{
			name:             "empty result ignoreNullValues=false returns error",
			serverStatusCode: http.StatusOK,
			responseBody: ionosPromQueryResult{
				Status: "success",
				Data: struct {
					ResultType string `json:"resultType"`
					Result     []struct {
						Metric struct{}      `json:"metric"`
						Value  []interface{} `json:"value"`
					} `json:"result"`
				}{ResultType: "vector"},
			},
			ignoreNullValues: false,
			expectError:      true,
		},
		{
			name:             "HTTP 401 returns error",
			serverStatusCode: http.StatusUnauthorized,
			responseBody:     "Unauthorized",
			ignoreNullValues: true,
			expectError:      true,
		},
		{
			name:             "non-success status in JSON returns error",
			serverStatusCode: http.StatusOK,
			responseBody: map[string]string{
				"status":    "error",
				"errorType": "bad_data",
				"error":     "parse error",
			},
			ignoreNullValues: true,
			expectError:      true,
		},
		{
			name:             "multiple results returns error",
			serverStatusCode: http.StatusOK,
			responseBody: ionosPromQueryResult{
				Status: "success",
				Data: struct {
					ResultType string `json:"resultType"`
					Result     []struct {
						Metric struct{}      `json:"metric"`
						Value  []interface{} `json:"value"`
					} `json:"result"`
				}{
					ResultType: "vector",
					Result: []struct {
						Metric struct{}      `json:"metric"`
						Value  []interface{} `json:"value"`
					}{
						{Value: []interface{}{float64(1), "1"}},
						{Value: []interface{}{float64(2), "2"}},
					},
				},
			},
			ignoreNullValues: true,
			expectError:      true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/prometheus/api/v1/query", r.URL.Path)
				assert.NotEmpty(t, r.URL.Query().Get("query"))
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatusCode)
				b, err := json.Marshal(tt.responseBody)
				assert.NoError(t, err)
				_, _ = w.Write(b)
			}))
			defer apiStub.Close()

			ignoreStr := "true"
			if !tt.ignoreNullValues {
				ignoreStr = "false"
			}
			scaler, err := NewIONOSMonitoringScaler(&scalersconfig.ScalerConfig{
				TriggerMetadata: map[string]string{
					"host":             apiStub.URL,
					"query":            "up",
					"threshold":        "10",
					"ignoreNullValues": ignoreStr,
				},
				AuthParams:        map[string]string{"apiKey": "test-api-key"},
				TriggerIndex:      0,
				GlobalHTTPTimeout: time.Minute,
			})
			require.NoError(t, err)

			_, _, err = scaler.GetMetricsAndActivity(context.Background(), "test-metric")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if !tt.expectError {
				metrics, _, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
				require.NoError(t, err)
				assert.InDelta(t, tt.expectedValue, metrics[0].Value.AsFloat64Slow(), 0.001,
					"unexpected metric value")
			}
		})
	}
}

func TestIONOSMonitoringActivityThreshold(t *testing.T) {
	apiStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		result := ionosPromQueryResult{
			Status: "success",
			Data: struct {
				ResultType string `json:"resultType"`
				Result     []struct {
					Metric struct{}      `json:"metric"`
					Value  []interface{} `json:"value"`
				} `json:"result"`
			}{
				ResultType: "vector",
				Result: []struct {
					Metric struct{}      `json:"metric"`
					Value  []interface{} `json:"value"`
				}{{Value: []interface{}{float64(1), "3"}}},
			},
		}
		b, _ := json.Marshal(result)
		_, _ = w.Write(b)
	}))
	defer apiStub.Close()

	scaler, err := NewIONOSMonitoringScaler(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"host":                apiStub.URL,
			"query":               "up",
			"threshold":           "10",
			"activationThreshold": "5",
		},
		AuthParams:        map[string]string{"apiKey": "secret"},
		TriggerIndex:      0,
		GlobalHTTPTimeout: time.Minute,
	})
	require.NoError(t, err)

	_, active, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	require.NoError(t, err)
	// value=3 is below activationThreshold=5, so should not be active
	assert.False(t, active)
}
