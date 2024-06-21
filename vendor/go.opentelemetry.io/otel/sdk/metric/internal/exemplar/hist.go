// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exemplar // import "go.opentelemetry.io/otel/sdk/metric/internal/exemplar"

import (
	"context"
	"slices"
	"sort"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

// Histogram returns a [Reservoir] that samples the last measurement that falls
// within a histogram bucket. The histogram bucket upper-boundaries are define
// by bounds.
//
// The passed bounds will be sorted by this function.
func Histogram[N int64 | float64](bounds []float64) Reservoir[N] {
	slices.Sort(bounds)
	return &histRes[N]{
		bounds:  bounds,
		storage: newStorage[N](len(bounds) + 1),
	}
}

type histRes[N int64 | float64] struct {
	*storage[N]

	// bounds are bucket bounds in ascending order.
	bounds []float64
}

func (r *histRes[N]) Offer(ctx context.Context, t time.Time, n N, a []attribute.KeyValue) {
	r.store[sort.SearchFloat64s(r.bounds, float64(n))] = newMeasurement(ctx, t, n, a)
}
