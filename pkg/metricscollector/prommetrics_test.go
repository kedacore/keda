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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
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
	httpClientRequestDuration := newHTTPClientRequestDuration(true)
	p := &PromMetrics{
		enableHighCardinalityLabels: true,
		httpClientRequestDuration:   httpClientRequestDuration,
	}

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

func TestNewPromMetrics_DisablesHighCardinalityLabels(t *testing.T) {
	previousRegistry := ctrlmetrics.Registry
	ctrlmetrics.Registry = prometheus.NewRegistry()
	t.Cleanup(func() {
		ctrlmetrics.Registry = previousRegistry
	})

	p := NewPromMetrics(false)
	p.RecordHTTPClientRequest(0.05, 200, false, "prometheus", "my-trigger", "my-metric", "default", "my-so")

	families, err := ctrlmetrics.Registry.Gather()
	require.NoError(t, err)

	names := map[string]struct{}{}
	for _, family := range families {
		names[family.GetName()] = struct{}{}
	}

	_, ok := names["keda_scaler_http_requests_total"]
	assert.True(t, ok, "keda_scaler_http_requests_total should be registered when high-cardinality labels are disabled")

	_, ok = names["keda_scaler_http_request_duration_seconds"]
	assert.True(t, ok, "keda_scaler_http_request_duration_seconds should be registered when high-cardinality labels are disabled")

	hist, err := p.httpClientRequestDuration.GetMetricWithLabelValues("prometheus", "200")
	require.NoError(t, err)
	m := &dto.Metric{}
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.EqualValues(t, 1, m.Histogram.GetSampleCount())

	_, err = p.httpClientRequestDuration.GetMetricWithLabelValues("default", "my-so", "prometheus", "my-trigger", "my-metric", "200")
	assert.Error(t, err, "high-cardinality labels should not be accepted when disabled")
}

// TestPromMetrics_LatencyDualWrite pins the dual-write of the latency gauges
// and their new histogram mirrors. A gauge keeps only the most recent
// observation, so it cannot answer what latency metrics exist to answer (p95,
// p99, how often a scaler is slow); the histogram accumulates every one. The
// gauges stay for backwards compatibility, so both must be written.
func TestPromMetrics_LatencyDualWrite(t *testing.T) {
	p := &PromMetrics{}

	const (
		namespace      = "default"
		scaledResource = "myScaledObject"
		scaler         = "cpuScaler"
		triggerIndex   = 0
		metric         = "cpu"
	)

	p.RecordScalerLatency(namespace, scaledResource, scaler, triggerIndex, metric, true, 250*time.Millisecond)
	p.RecordScalerLatency(namespace, scaledResource, scaler, triggerIndex, metric, true, 750*time.Millisecond)

	labels := getLabels(namespace, scaledResource, scaler, triggerIndex, metric, true)

	// Gauge: last observation only — the pre-existing behaviour.
	gauge := &dto.Metric{}
	require.NoError(t, scalerMetricsLatency.With(labels).Write(gauge))
	assert.InDelta(t, 0.75, gauge.GetGauge().GetValue(), 1e-9)

	// Histogram: both observations, so quantiles are computable.
	hist := &dto.Metric{}
	require.NoError(t, scalerMetricsDuration.With(labels).(prometheus.Metric).Write(hist))
	assert.Equal(t, uint64(2), hist.GetHistogram().GetSampleCount())
	assert.InDelta(t, 1.0, hist.GetHistogram().GetSampleSum(), 1e-9)
}

func TestPromMetrics_ScalableObjectLatencyDualWrite(t *testing.T) {
	p := &PromMetrics{}

	const (
		namespace = "default"
		name      = "myScaledObject2"
	)

	p.RecordScalableObjectLatency(namespace, name, true, 100*time.Millisecond)
	p.RecordScalableObjectLatency(namespace, name, true, 300*time.Millisecond)

	gauge := &dto.Metric{}
	require.NoError(t, internalLoopLatency.WithLabelValues(namespace, "scaledobject", name).Write(gauge))
	assert.InDelta(t, 0.3, gauge.GetGauge().GetValue(), 1e-9)

	hist := &dto.Metric{}
	require.NoError(t, internalLoopDuration.WithLabelValues(namespace, "scaledobject", name).(prometheus.Metric).Write(hist))
	assert.Equal(t, uint64(2), hist.GetHistogram().GetSampleCount())
	assert.InDelta(t, 0.4, hist.GetHistogram().GetSampleSum(), 1e-9)
}

// TestPromMetrics_DeleteScalerMetricsClearsHistogram guards against the new
// histogram outliving a deleted trigger and reporting stale series, which is
// exactly what DeleteScalerMetrics exists to prevent for the other scaler
// metrics.
func TestPromMetrics_DeleteScalerMetricsClearsHistogram(t *testing.T) {
	p := &PromMetrics{}

	const (
		namespace      = "delete-ns"
		scaledResource = "goingAway"
	)

	// scalerMetricsDuration is package-level and shared with the other tests in
	// this file, so assert on the delta rather than an absolute series count.
	before := testutil.CollectAndCount(scalerMetricsDuration)

	p.RecordScalerLatency(namespace, scaledResource, "cpuScaler", 0, "cpu", true, time.Second)
	require.Equal(t, before+1, testutil.CollectAndCount(scalerMetricsDuration),
		"recording should add exactly one series")

	p.DeleteScalerMetrics(namespace, scaledResource, true)
	assert.Equal(t, before, testutil.CollectAndCount(scalerMetricsDuration),
		"a deleted trigger must not leave a stale histogram series behind")
}
