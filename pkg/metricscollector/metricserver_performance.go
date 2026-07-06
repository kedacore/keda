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
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	requestResultSuccess = "success"
	requestResultError   = "error"
)

var (
	adapterExternalMetricRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "external_metrics_provider",
			Name:      "requests_total",
			Help:      "Total number of external metric requests served to the Kubernetes HPA, labeled by outcome.",
		},
		[]string{"namespace", "scaled_object", "metric", "result"},
	)

	adapterExternalMetricRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "external_metrics_provider",
			Name:      "request_duration_seconds",
			Help:      "Duration in seconds of external metric requests served to the Kubernetes HPA.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"result"},
	)

	adapterPerformancePromOnce sync.Once

	adapterOtelMetrics               *adapterOtelPerformanceMetrics
	adapterOtelPerformanceMetricsLog = logf.Log.WithName("adapter_otel_performance_metrics")
)

type adapterOtelPerformanceMetrics struct {
	externalMetricRequests api.Int64Counter
	externalMetricDuration api.Float64Histogram
}

// RegisterAdapterPerformancePromMetrics registers HPA-facing performance metrics.
// Registration runs at most once; the first call wins and later calls are no-ops even if a different registerer is passed.
// Intended for use by keda-metrics-apiserver which exposes metrics through legacyregistry.
func RegisterAdapterPerformancePromMetrics(registerer prometheus.Registerer) {
	adapterPerformancePromOnce.Do(func() {
		registerer.MustRegister(adapterExternalMetricRequestsTotal)
		registerer.MustRegister(adapterExternalMetricRequestDuration)
	})
}

// InitAdapterOtelPerformanceMetrics initializes OpenTelemetry export of HPA-facing performance metrics.
func InitAdapterOtelPerformanceMetrics() {
	if adapterOtelMetrics != nil {
		return
	}

	protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")

	var exporter metric.Exporter
	var err error
	switch protocol {
	case "grpc":
		adapterOtelPerformanceMetricsLog.V(1).Info("start OTEL grpc client for adapter performance metrics")
		exporter, err = otlpmetricgrpc.New(context.Background())
	default:
		adapterOtelPerformanceMetricsLog.V(1).Info("start OTEL http client for adapter performance metrics")
		exporter, err = otlpmetrichttp.New(context.Background())
	}

	if err != nil {
		adapterOtelPerformanceMetricsLog.Error(err, "failed to initialize adapter OTEL performance metrics")
		return
	}

	initAdapterOtelPerformanceMetricsWithReader(metric.NewPeriodicReader(exporter))
}

func initAdapterOtelPerformanceMetricsWithReader(reader metric.Reader) {
	if adapterOtelMetrics != nil {
		return
	}

	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(meterProvider)

	meter := meterProvider.Meter("keda-adapter-performance-metrics")
	msg := "failed to create OpenTelemetry instrument for adapter performance metrics"

	externalMetricRequests, err := meter.Int64Counter(
		"keda.external_metrics_provider.requests.count",
		api.WithDescription("Total number of external metric requests served to the Kubernetes HPA"),
	)
	if err != nil {
		adapterOtelPerformanceMetricsLog.Error(err, msg)
		return
	}

	externalMetricDuration, err := meter.Float64Histogram(
		"keda.external_metrics_provider.request.duration.seconds",
		api.WithDescription("Duration in seconds of external metric requests served to the Kubernetes HPA"),
		api.WithUnit("s"),
	)
	if err != nil {
		adapterOtelPerformanceMetricsLog.Error(err, msg)
		return
	}

	adapterOtelMetrics = &adapterOtelPerformanceMetrics{
		externalMetricRequests: externalMetricRequests,
		externalMetricDuration: externalMetricDuration,
	}
}

// RecordAdapterExternalMetricRequest records an HPA external metric request handled by keda-metrics-apiserver.
func RecordAdapterExternalMetricRequest(durationSeconds float64, err error, namespace, scaledObject, metricName string) {
	result := requestResult(err)

	adapterExternalMetricRequestsTotal.WithLabelValues(namespace, scaledObject, metricName, result).Inc()
	adapterExternalMetricRequestDuration.WithLabelValues(result).Observe(durationSeconds)

	if adapterOtelMetrics != nil {
		counterOpt := api.WithAttributes(
			attribute.Key("namespace").String(namespace),
			attribute.Key("scaled_object").String(scaledObject),
			attribute.Key("metric").String(metricName),
			attribute.Key("result").String(result),
		)
		histOpt := api.WithAttributes(attribute.Key("result").String(result))

		adapterOtelMetrics.externalMetricRequests.Add(context.Background(), 1, counterOpt)
		adapterOtelMetrics.externalMetricDuration.Record(context.Background(), durationSeconds, histOpt)
	}
}

func requestResult(err error) string {
	if err != nil {
		return requestResultError
	}
	return requestResultSuccess
}
