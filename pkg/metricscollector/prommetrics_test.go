/*
Copyright 2026 The KEDA Authors

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

package metricscollector

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPStatusCodeLabel(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		isError    bool
		want       string
	}{
		{"transport error", 0, true, "error"},
		{"isError flag takes precedence over non-zero code", 500, true, "error"},
		{"200 OK", 200, false, "200"},
		{"201 Created", 201, false, "201"},
		{"301 Moved", 301, false, "301"},
		{"400 Bad Request", 400, false, "400"},
		{"404 Not Found", 404, false, "404"},
		{"500 Internal Server Error", 500, false, "500"},
		{"503 Service Unavailable", 503, false, "503"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := httpStatusCodeLabel(tt.statusCode, tt.isError)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPromMetrics_RecordHTTPClientRequest(t *testing.T) {
	p := &PromMetrics{}

	// Verify no panic and label combinations are created without error.
	p.RecordHTTPClientRequest(0.05, 200, false, "prometheus", "my-trigger", "my-metric", "default", "my-so")
	p.RecordHTTPClientRequest(0.1, 404, false, "redis", "redis-trigger", "redis-metric", "default", "my-so")
	p.RecordHTTPClientRequest(0.2, 500, false, "prometheus", "my-trigger", "my-metric", "default", "my-so")
	p.RecordHTTPClientRequest(0.3, 0, true, "", "", "", "", "")

	m := &dto.Metric{}

	counter, err := httpClientRequestsTotal.GetMetricWithLabelValues("default", "my-so", "prometheus", "my-trigger", "my-metric", "200")
	require.NoError(t, err)
	require.NoError(t, counter.Write(m))
	assert.EqualValues(t, 1, m.Counter.GetValue())

	counter, err = httpClientRequestsTotal.GetMetricWithLabelValues("default", "my-so", "redis", "redis-trigger", "redis-metric", "404")
	require.NoError(t, err)
	require.NoError(t, counter.Write(m))
	assert.EqualValues(t, 1, m.Counter.GetValue())

	counter, err = httpClientRequestsTotal.GetMetricWithLabelValues("default", "my-so", "prometheus", "my-trigger", "my-metric", "500")
	require.NoError(t, err)
	require.NoError(t, counter.Write(m))
	assert.EqualValues(t, 1, m.Counter.GetValue())

	counter, err = httpClientRequestsTotal.GetMetricWithLabelValues("", "", "", "", "", "error")
	require.NoError(t, err)
	require.NoError(t, counter.Write(m))
	assert.EqualValues(t, 1, m.Counter.GetValue())

	hist, err := httpClientRequestDuration.GetMetricWithLabelValues("default", "my-so", "prometheus", "my-trigger", "my-metric", "200")
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.EqualValues(t, 1, m.Histogram.GetSampleCount())
	assert.InDelta(t, 0.05, m.Histogram.GetSampleSum(), 0.001)
}
