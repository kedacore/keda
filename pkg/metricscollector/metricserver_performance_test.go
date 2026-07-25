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
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestRequestResult(t *testing.T) {
	assert.Equal(t, requestResultSuccess, requestResult(nil))
	assert.Equal(t, requestResultError, requestResult(errors.New("boom")))
}

func TestRecordAdapterExternalMetricRequest(t *testing.T) {
	reg := prometheus.NewRegistry()
	RegisterAdapterPerformancePromMetrics(reg)

	namespace := t.Name() + "-ns"
	scaledObject := t.Name() + "-so"
	metricName := t.Name() + "-metric"

	RecordAdapterExternalMetricRequest(0.05, nil, namespace, scaledObject, metricName)
	RecordAdapterExternalMetricRequest(0.1, errors.New("failed"), namespace, scaledObject, metricName)

	m := &dto.Metric{}

	counter, err := adapterExternalMetricRequestsTotal.GetMetricWithLabelValues(namespace, scaledObject, metricName, requestResultSuccess)
	require.NoError(t, err)
	require.NoError(t, counter.Write(m))
	assert.EqualValues(t, 1, m.Counter.GetValue())

	counter, err = adapterExternalMetricRequestsTotal.GetMetricWithLabelValues(namespace, scaledObject, metricName, requestResultError)
	require.NoError(t, err)
	require.NoError(t, counter.Write(m))
	assert.EqualValues(t, 1, m.Counter.GetValue())

	hist, err := adapterExternalMetricRequestDuration.GetMetricWithLabelValues(requestResultSuccess)
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.GreaterOrEqual(t, m.Histogram.GetSampleCount(), uint64(1))
}

func TestRecordAdapterExternalMetricRequest_Otel(t *testing.T) {
	previousAdapterOtelMetrics := adapterOtelMetrics
	t.Cleanup(func() {
		adapterOtelMetrics = previousAdapterOtelMetrics
	})

	reader := metric.NewManualReader()
	InitAdapterOtelPerformanceMetricsForTest(reader)

	namespace := t.Name() + "-ns"
	scaledObject := t.Name() + "-so"
	metricName := t.Name() + "-metric"

	RecordAdapterExternalMetricRequest(0.05, nil, namespace, scaledObject, metricName)
	RecordAdapterExternalMetricRequest(0.1, errors.New("failed"), namespace, scaledObject, metricName)

	got := metricdata.ResourceMetrics{}
	err := reader.Collect(context.Background(), &got)
	require.NoError(t, err)

	scopeMetrics := got.ScopeMetrics[0]
	requestCount := retrieveMetric(scopeMetrics.Metrics, "keda.external_metrics_provider.requests.count")
	require.NotNil(t, requestCount)

	var successCount, errorCount int64
	for _, dp := range requestCount.Data.(metricdata.Sum[int64]).DataPoints {
		result, _ := dp.Attributes.Value("result")
		switch result.AsString() {
		case requestResultSuccess:
			successCount = dp.Value
		case requestResultError:
			errorCount = dp.Value
		}
	}
	assert.Equal(t, int64(1), successCount)
	assert.Equal(t, int64(1), errorCount)

	requestDuration := retrieveMetric(scopeMetrics.Metrics, "keda.external_metrics_provider.request.duration.seconds")
	require.NotNil(t, requestDuration)
	assert.Equal(t, "s", requestDuration.Unit)
}

func TestRecordMetricsServiceGetMetricsRequest_Collectors(t *testing.T) {
	withPromCollector(t)

	namespace := t.Name() + "-ns"
	scaledObject := t.Name() + "-so"
	metricName := t.Name() + "-metric"

	RecordMetricsServiceGetMetricsRequest(0.05, nil, namespace, scaledObject, metricName)
	RecordMetricsServiceGetMetricsRequest(0.1, errors.New("failed"), namespace, scaledObject, metricName)

	m := &dto.Metric{}

	counter, err := metricsServiceGetMetricsRequestsTotal.GetMetricWithLabelValues(namespace, scaledObject, metricName, requestResultSuccess)
	require.NoError(t, err)
	require.NoError(t, counter.Write(m))
	assert.EqualValues(t, 1, m.Counter.GetValue())

	counter, err = metricsServiceGetMetricsRequestsTotal.GetMetricWithLabelValues(namespace, scaledObject, metricName, requestResultError)
	require.NoError(t, err)
	require.NoError(t, counter.Write(m))
	assert.EqualValues(t, 1, m.Counter.GetValue())

	hist, err := metricsServiceGetMetricsDuration.GetMetricWithLabelValues(requestResultSuccess)
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.GreaterOrEqual(t, m.Histogram.GetSampleCount(), uint64(1))
}

func TestPromMetrics_RecordMetricsServiceGetMetricsRequest(t *testing.T) {
	p := &PromMetrics{}

	namespace := t.Name() + "-ns"
	scaledObject := t.Name() + "-so"
	metricName := t.Name() + "-metric"

	p.RecordMetricsServiceGetMetricsRequest(0.05, nil, namespace, scaledObject, metricName)
	p.RecordMetricsServiceGetMetricsRequest(0.1, errors.New("failed"), namespace, scaledObject, metricName)

	m := &dto.Metric{}

	counter, err := metricsServiceGetMetricsRequestsTotal.GetMetricWithLabelValues(namespace, scaledObject, metricName, requestResultSuccess)
	require.NoError(t, err)
	require.NoError(t, counter.Write(m))
	assert.EqualValues(t, 1, m.Counter.GetValue())

	counter, err = metricsServiceGetMetricsRequestsTotal.GetMetricWithLabelValues(namespace, scaledObject, metricName, requestResultError)
	require.NoError(t, err)
	require.NoError(t, counter.Write(m))
	assert.EqualValues(t, 1, m.Counter.GetValue())

	hist, err := metricsServiceGetMetricsDuration.GetMetricWithLabelValues(requestResultSuccess)
	require.NoError(t, err)
	require.NoError(t, hist.(prometheus.Metric).Write(m))
	assert.GreaterOrEqual(t, m.Histogram.GetSampleCount(), uint64(1))
}
