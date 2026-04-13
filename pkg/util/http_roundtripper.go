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

package util

import (
	"net/http"
	"time"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
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

// InstrumentedRoundTripper wraps an http.RoundTripper and records outbound
// HTTP request metrics after every completed round-trip. It reads known
// context keys from the request context to populate metric dimensions. It
// does not buffer or inspect the response body.
type InstrumentedRoundTripper struct {
	next http.RoundTripper
}

// NewInstrumentedRoundTripper wraps next with a RoundTripper that records
// HTTP request metrics after every request. If next is nil,
// http.DefaultTransport is used.
func NewInstrumentedRoundTripper(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &InstrumentedRoundTripper{next: next}
}

func (r *InstrumentedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := r.next.RoundTrip(req)
	duration := time.Since(start).Seconds()

	ctx := req.Context()
	scaler, scalerOK := ctx.Value(ScalerContextKey).(string)
	triggerName, triggerOK := ctx.Value(TriggerNameContextKey).(string)
	metricName, metricOK := ctx.Value(MetricNameContextKey).(string)
	namespace, nsOK := ctx.Value(NamespaceContextKey).(string)
	scaledResource, srOK := ctx.Value(ScaledResourceContextKey).(string)

	// Only record metrics for scaler metric-fetch requests, identified by the
	// presence of all five context keys injected by buildScalerRequestCtx.
	// Other HTTP calls (e.g. during scaler initialization) are not recorded.
	if !scalerOK || !triggerOK || !metricOK || !nsOK || !srOK {
		return resp, err
	}

	if err != nil {
		metricscollector.RecordHTTPClientRequest(duration, 0, true, scaler, triggerName, metricName, namespace, scaledResource)
		return nil, err
	}
	metricscollector.RecordHTTPClientRequest(duration, resp.StatusCode, false, scaler, triggerName, metricName, namespace, scaledResource)
	return resp, nil
}
