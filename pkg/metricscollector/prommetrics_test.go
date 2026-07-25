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

	hist, err := httpClientRequestDuration.GetMetricWithLabelValues("prometheus", "200")
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.EqualValues(t, 1, m.Histogram.GetSampleCount())
	assert.InDelta(t, 0.05, m.Histogram.GetSampleSum(), 0.001)
}

func TestPromMetrics_RecordScalerLatency_DefaultLabels(t *testing.T) {
	p := &PromMetrics{}

	// scaler carries the user-defined trigger name, triggerType the bounded scaler type
	p.RecordScalerLatency("latency-ns", "latency-so", "my-cron-trigger", "cron", 0, "latency-metric", true, 250*time.Millisecond)

	// the deprecated gauge keeps the full label set, with the trigger name as scaler
	m := &dto.Metric{}
	gauge, err := scalerMetricsLatency.GetMetricWith(getLabels("latency-ns", "latency-so", "my-cron-trigger", 0, "latency-metric", true))
	require.NoError(t, err)
	require.NoError(t, gauge.Write(m))
	assert.InDelta(t, 0.25, m.Gauge.GetValue(), 0.001)

	// the histogram mirror is labeled by the bounded trigger type only by default,
	// even when the user gave the trigger a custom name
	hist, err := scalerMetricsDuration.GetMetricWithLabelValues("cron")
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.EqualValues(t, 1, m.Histogram.GetSampleCount())
	assert.InDelta(t, 0.25, m.Histogram.GetSampleSum(), 0.001)

	// no histogram series was created for the user-defined trigger name
	assert.Zero(t, scalerMetricsDuration.DeletePartialMatch(prometheus.Labels{"scaler": "my-cron-trigger"}))

	// the full label set is not accepted without the high-cardinality flag
	_, err = scalerMetricsDuration.GetMetricWith(getLabels("latency-ns", "latency-so", "my-cron-trigger", 0, "latency-metric", true))
	assert.Error(t, err)

	// deleting the scaled resource keeps the shared reduced-label series intact
	p.DeleteScalerMetrics("latency-ns", "latency-so", true)
	hist, err = scalerMetricsDuration.GetMetricWithLabelValues("cron")
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.EqualValues(t, 1, m.Histogram.GetSampleCount())
}

func TestPromMetrics_RecordScalableObjectLatency_DefaultLabels(t *testing.T) {
	p := &PromMetrics{}

	p.RecordScalableObjectLatency("loop-ns", "loop-so", true, 100*time.Millisecond)
	p.RecordScalableObjectLatency("loop-ns", "loop-sj", false, 200*time.Millisecond)

	m := &dto.Metric{}
	hist, err := internalLoopDuration.GetMetricWithLabelValues("scaledobject")
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.EqualValues(t, 1, m.Histogram.GetSampleCount())
	assert.InDelta(t, 0.1, m.Histogram.GetSampleSum(), 0.001)

	hist, err = internalLoopDuration.GetMetricWithLabelValues("scaledjob")
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.EqualValues(t, 1, m.Histogram.GetSampleCount())
	assert.InDelta(t, 0.2, m.Histogram.GetSampleSum(), 0.001)
}

func TestPromMetrics_DurationHistograms_HighCardinalityLabels(t *testing.T) {
	prevMode := highCardinalityMetricLabels
	prevScalerHist := scalerMetricsDuration
	prevLoopHist := internalLoopDuration
	t.Cleanup(func() {
		highCardinalityMetricLabels = prevMode
		scalerMetricsDuration = prevScalerHist
		internalLoopDuration = prevLoopHist
	})
	highCardinalityMetricLabels = true
	scalerMetricsDuration = newScalerMetricsDurationHistogram(true)
	internalLoopDuration = newInternalLoopDurationHistogram(true)

	p := &PromMetrics{}
	p.RecordScalerLatency("hc-ns", "hc-so", "hc-trigger", "prometheus", 1, "hc-metric", true, 500*time.Millisecond)
	p.RecordScalableObjectLatency("hc-ns", "hc-so", true, 300*time.Millisecond)

	// in high-cardinality mode the histogram carries the same label set as the gauge,
	// including the user-defined trigger name as scaler
	m := &dto.Metric{}
	hist, err := scalerMetricsDuration.GetMetricWith(getLabels("hc-ns", "hc-so", "hc-trigger", 1, "hc-metric", true))
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.EqualValues(t, 1, m.Histogram.GetSampleCount())
	assert.InDelta(t, 0.5, m.Histogram.GetSampleSum(), 0.001)

	hist, err = internalLoopDuration.GetMetricWith(prometheus.Labels{"namespace": "hc-ns", "type": "scaledobject", "resource": "hc-so"})
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.EqualValues(t, 1, m.Histogram.GetSampleCount())
	assert.InDelta(t, 0.3, m.Histogram.GetSampleSum(), 0.001)

	// deleting the scaled resource also deletes its histogram series in high-cardinality mode
	p.DeleteScalerMetrics("hc-ns", "hc-so", true)
	hist, err = scalerMetricsDuration.GetMetricWith(getLabels("hc-ns", "hc-so", "hc-trigger", 1, "hc-metric", true))
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.EqualValues(t, 0, m.Histogram.GetSampleCount())
}
