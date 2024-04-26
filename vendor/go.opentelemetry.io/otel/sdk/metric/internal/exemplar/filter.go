// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exemplar // import "go.opentelemetry.io/otel/sdk/metric/internal/exemplar"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SampledFilter returns a [Reservoir] wrapping r that will only offer measurements
// to r if the passed context associated with the measurement contains a sampled
// [go.opentelemetry.io/otel/trace.SpanContext].
func SampledFilter[N int64 | float64](r Reservoir[N]) Reservoir[N] {
	return filtered[N]{Reservoir: r}
}

type filtered[N int64 | float64] struct {
	Reservoir[N]
}

func (f filtered[N]) Offer(ctx context.Context, t time.Time, n N, a []attribute.KeyValue) {
	if trace.SpanContextFromContext(ctx).IsSampled() {
		f.Reservoir.Offer(ctx, t, n, a)
	}
}
