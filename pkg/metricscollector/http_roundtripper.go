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
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// contextKey is an unexported type for context keys defined in this package,
// preventing collisions with keys from other packages.
type contextKey string

const (
	// ScalerContextKey is the context key used to attach the scaler type name
	// (e.g. "prometheus", "redis") to an outbound HTTP request so that metrics
	// observers can include it as a dimension.
	ScalerContextKey contextKey = "scaler"

	// TriggerNameContextKey is the context key used to attach the user-defined
	// trigger name to an outbound HTTP request.
	TriggerNameContextKey contextKey = "trigger_name"

	// MetricNameContextKey is the context key used to attach the metric name
	// being queried to an outbound HTTP request.
	MetricNameContextKey contextKey = "metric_name"

	// NamespaceContextKey is the context key used to attach the namespace of the
	// ScaledObject/ScaledJob that owns the scaler making the request.
	NamespaceContextKey contextKey = "namespace"

	// ScaledResourceContextKey is the context key used to attach the name of the
	// ScaledObject/ScaledJob that owns the scaler making the request.
	ScaledResourceContextKey contextKey = "scaled_resource"
)

type scalerMetricsRoundTripper struct {
	next         http.RoundTripper
	instrumented http.RoundTripper
}

func NewInstrumentedRoundTripper(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	enablePrometheusMetrics := HTTPClientPrometheusMetricsEnabled()
	enableOpenTelemetryMetrics := HTTPClientOpenTelemetryMetricsEnabled()

	var instrumented http.RoundTripper = next

	if enablePrometheusMetrics {
		counterOpts := []promhttp.Option{
			promhttp.WithLabelFromCtx("namespace", labelFromCtx(NamespaceContextKey)),
			promhttp.WithLabelFromCtx("scaled_resource", labelFromCtx(ScaledResourceContextKey)),
			promhttp.WithLabelFromCtx("scaler", labelFromCtx(ScalerContextKey)),
			promhttp.WithLabelFromCtx("trigger_name", labelFromCtx(TriggerNameContextKey)),
			promhttp.WithLabelFromCtx("metric_name", labelFromCtx(MetricNameContextKey)),
		}

		durationOpts := []promhttp.Option{
			promhttp.WithLabelFromCtx("scaler", labelFromCtx(ScalerContextKey)),
		}

		instrumented = promhttp.InstrumentRoundTripperCounter(
			HTTPClientRequestsCollector(),
			instrumented,
			counterOpts...,
		)
		instrumented = promhttp.InstrumentRoundTripperDuration(
			HTTPClientRequestDurationCollector(),
			instrumented,
			durationOpts...,
		)
	}

	if enableOpenTelemetryMetrics {
		instrumented = otelhttp.NewTransport(instrumented)
	}

	return &scalerMetricsRoundTripper{
		next:         next,
		instrumented: instrumented,
	}
}

func (r *scalerMetricsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if !hasScalerMetricsContext(req.Context()) {
		return r.next.RoundTrip(req)
	}

	return r.instrumented.RoundTrip(req)
}

func hasScalerMetricsContext(ctx context.Context) bool {
	_, scalerOK := ctx.Value(ScalerContextKey).(string)
	_, triggerOK := ctx.Value(TriggerNameContextKey).(string)
	_, metricOK := ctx.Value(MetricNameContextKey).(string)
	_, nsOK := ctx.Value(NamespaceContextKey).(string)
	_, resourceOK := ctx.Value(ScaledResourceContextKey).(string)

	return scalerOK && triggerOK && metricOK && nsOK && resourceOK
}

func labelFromCtx(key contextKey) promhttp.LabelValueFromCtx {
	return func(ctx context.Context) string {
		if value, ok := ctx.Value(key).(string); ok {
			return value
		}

		return ""
	}
}
