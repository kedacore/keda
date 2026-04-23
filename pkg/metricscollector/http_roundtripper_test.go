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

package metricscollector_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type collectorMockRoundTripper struct {
	resp *http.Response
	err  error
}

func (m *collectorMockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.resp != nil && m.resp.Request == nil {
		m.resp.Request = req
	}

	return m.resp, m.err
}

func fakeResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func newRequest(ctx context.Context) *http.Request {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	return req
}

func collectMetrics(t *testing.T, collector prometheus.Collector) []*dto.Metric {
	t.Helper()

	ch := make(chan prometheus.Metric, 32)
	collector.Collect(ch)
	close(ch)

	var metrics []*dto.Metric
	for metric := range ch {
		dtoMetric := &dto.Metric{}
		require.NoError(t, metric.Write(dtoMetric))
		metrics = append(metrics, dtoMetric)
	}

	return metrics
}

func hasLabels(metric *dto.Metric, labels map[string]string) bool {
	for key, value := range labels {
		found := false
		for _, label := range metric.GetLabel() {
			if label.GetName() == key && label.GetValue() == value {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func TestInstrumentedRoundTripper_ResponseReturnedUnmodified(t *testing.T) {
	metricscollector.ConfigureHTTPClientMetricsInstrumentation(false, false)

	expected := fakeResponse(http.StatusAccepted)
	rt := metricscollector.NewInstrumentedRoundTripper(&collectorMockRoundTripper{resp: expected})

	got, err := rt.RoundTrip(newRequest(context.Background()))

	require.NoError(t, err)
	defer got.Body.Close()
	assert.Same(t, expected, got)
}

func TestInstrumentedRoundTripper_NilNextUsesDefault(t *testing.T) {
	metricscollector.ConfigureHTTPClientMetricsInstrumentation(false, false)

	rt := metricscollector.NewInstrumentedRoundTripper(nil)
	assert.Equal(t, "*metricscollector.scalerMetricsRoundTripper", fmt.Sprintf("%T", rt))
}

func TestInstrumentedRoundTripper_WithScalerContextRecordsPrometheusMetrics(t *testing.T) {
	metricscollector.ConfigureHTTPClientMetricsInstrumentation(true, false)

	rt := metricscollector.NewInstrumentedRoundTripper(&collectorMockRoundTripper{resp: fakeResponse(http.StatusOK)})

	ctx := context.Background()
	ctx = context.WithValue(ctx, metricscollector.ScalerContextKey, "prometheus")
	ctx = context.WithValue(ctx, metricscollector.TriggerNameContextKey, "my-trigger")
	ctx = context.WithValue(ctx, metricscollector.MetricNameContextKey, "my-metric")
	ctx = context.WithValue(ctx, metricscollector.NamespaceContextKey, "default")
	ctx = context.WithValue(ctx, metricscollector.ScaledResourceContextKey, "my-so")

	resp, err := rt.RoundTrip(newRequest(ctx))
	require.NoError(t, err)
	defer resp.Body.Close()

	counter, err := metricscollector.HTTPClientRequestsCollector().
		GetMetricWithLabelValues("200", "default", "my-so", "prometheus", "my-trigger", "my-metric")
	require.NoError(t, err)

	m := &dto.Metric{}
	require.NoError(t, counter.Write(m))
	assert.GreaterOrEqual(t, m.Counter.GetValue(), float64(1))

	var found bool
	for _, metric := range collectMetrics(t, metricscollector.HTTPClientRequestDurationCollector()) {
		if hasLabels(metric, map[string]string{
			"code":   "200",
			"scaler": "prometheus",
		}) {
			assert.GreaterOrEqual(t, metric.GetHistogram().GetSampleCount(), uint64(1))
			found = true
			break
		}
	}

	assert.True(t, found, "expected histogram metric with scaler context labels")
}

func TestCreateHTTPClient_TransportIsInstrumented(t *testing.T) {
	metricscollector.ConfigureHTTPClientMetricsInstrumentation(false, false)

	client := kedautil.CreateHTTPClient(0, false)
	assert.Equal(t, "*metricscollector.scalerMetricsRoundTripper", fmt.Sprintf("%T", client.Transport))
}

func TestCreateRTWithTLSConfig_IsInstrumented(t *testing.T) {
	metricscollector.ConfigureHTTPClientMetricsInstrumentation(false, false)

	rt := kedautil.CreateRTWithTLSConfig(nil)
	assert.Equal(t, "*metricscollector.scalerMetricsRoundTripper", fmt.Sprintf("%T", rt))
}
