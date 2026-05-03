/*
Copyright 2026 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package metricscollector

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestRecordScalerLatencyEmitsHistogram verifies that calling RecordScalerLatency
// produces observations in keda_scaler_metrics_duration_seconds in addition to
// the legacy gauge keda_scaler_metrics_latency_seconds. See issue #7675.
func TestRecordScalerLatencyEmitsHistogram(t *testing.T) {
	scalerMetricsDuration.Reset()
	scalerMetricsLatency.Reset()

	p := &PromMetrics{}
	p.RecordScalerLatency("ns", "so1", "kafka-scaler", 0, "lag", true, 250*time.Millisecond)
	p.RecordScalerLatency("ns", "so1", "kafka-scaler", 0, "lag", true, 1500*time.Millisecond)

	if got, want := testutil.CollectAndCount(scalerMetricsDuration, "keda_scaler_metrics_duration_seconds"), 1; got != want {
		t.Fatalf("histogram series count = %d, want %d", got, want)
	}
	// CollectAndCount on a histogram counts time series, not observations.
	// Use ToFloat64 on the gauge to confirm it was also written (it stores last-set value).
	if got := testutil.ToFloat64(scalerMetricsLatency); got != 1.5 {
		t.Fatalf("gauge keda_scaler_metrics_latency_seconds = %v, want 1.5 (last value)", got)
	}
}

// TestRecordScalableObjectLatencyEmitsHistogram covers the internal-loop variant.
func TestRecordScalableObjectLatencyEmitsHistogram(t *testing.T) {
	internalLoopDuration.Reset()
	internalLoopLatency.Reset()

	p := &PromMetrics{}
	p.RecordScalableObjectLatency("ns", "so2", true, 30*time.Millisecond)
	p.RecordScalableObjectLatency("ns", "so2", true, 70*time.Millisecond)

	if got, want := testutil.CollectAndCount(internalLoopDuration, "keda_internal_scale_loop_duration_seconds"), 1; got != want {
		t.Fatalf("histogram series count = %d, want %d", got, want)
	}
	if got := testutil.ToFloat64(internalLoopLatency); got != 0.07 {
		t.Fatalf("gauge keda_internal_scale_loop_latency_seconds = %v, want 0.07 (last value)", got)
	}
}

// TestDeleteScalerMetricsDropsHistogram ensures DeleteScalerMetrics also cleans
// the new histogram series so we don't keep stale labels around forever.
func TestDeleteScalerMetricsDropsHistogram(t *testing.T) {
	scalerMetricsDuration.Reset()

	p := &PromMetrics{}
	p.RecordScalerLatency("ns", "so3", "redis-scaler", 0, "len", true, 10*time.Millisecond)
	if got := testutil.CollectAndCount(scalerMetricsDuration, "keda_scaler_metrics_duration_seconds"); got != 1 {
		t.Fatalf("expected 1 series before delete, got %d", got)
	}
	p.DeleteScalerMetrics("ns", "so3", true)
	if got := testutil.CollectAndCount(scalerMetricsDuration, "keda_scaler_metrics_duration_seconds"); got != 0 {
		t.Fatalf("expected 0 series after delete, got %d", got)
	}
}
