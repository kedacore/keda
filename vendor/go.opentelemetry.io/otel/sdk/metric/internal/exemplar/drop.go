// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exemplar // import "go.opentelemetry.io/otel/sdk/metric/internal/exemplar"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Drop returns a [Reservoir] that drops all measurements it is offered.
func Drop[N int64 | float64]() Reservoir[N] { return &dropRes[N]{} }

type dropRes[N int64 | float64] struct{}

// Offer does nothing, all measurements offered will be dropped.
func (r *dropRes[N]) Offer(context.Context, time.Time, N, []attribute.KeyValue) {}

// Collect resets dest. No exemplars will ever be returned.
func (r *dropRes[N]) Collect(dest *[]metricdata.Exemplar[N]) {
	*dest = (*dest)[:0]
}
