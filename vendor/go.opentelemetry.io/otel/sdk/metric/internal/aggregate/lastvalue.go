// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package aggregate // import "go.opentelemetry.io/otel/sdk/metric/internal/aggregate"

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/internal/exemplar"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// datapoint is timestamped measurement data.
type datapoint[N int64 | float64] struct {
	attrs     attribute.Set
	timestamp time.Time
	value     N
	res       exemplar.Reservoir[N]
}

func newLastValue[N int64 | float64](limit int, r func() exemplar.Reservoir[N]) *lastValue[N] {
	return &lastValue[N]{
		newRes: r,
		limit:  newLimiter[datapoint[N]](limit),
		values: make(map[attribute.Distinct]datapoint[N]),
	}
}

// lastValue summarizes a set of measurements as the last one made.
type lastValue[N int64 | float64] struct {
	sync.Mutex

	newRes func() exemplar.Reservoir[N]
	limit  limiter[datapoint[N]]
	values map[attribute.Distinct]datapoint[N]
}

func (s *lastValue[N]) measure(ctx context.Context, value N, fltrAttr attribute.Set, droppedAttr []attribute.KeyValue) {
	t := now()

	s.Lock()
	defer s.Unlock()

	attr := s.limit.Attributes(fltrAttr, s.values)
	d, ok := s.values[attr.Equivalent()]
	if !ok {
		d.res = s.newRes()
	}

	d.attrs = attr
	d.timestamp = t
	d.value = value
	d.res.Offer(ctx, t, value, droppedAttr)

	s.values[attr.Equivalent()] = d
}

func (s *lastValue[N]) computeAggregation(dest *[]metricdata.DataPoint[N]) {
	s.Lock()
	defer s.Unlock()

	n := len(s.values)
	*dest = reset(*dest, n, n)

	var i int
	for _, v := range s.values {
		(*dest)[i].Attributes = v.attrs
		// The event time is the only meaningful timestamp, StartTime is
		// ignored.
		(*dest)[i].Time = v.timestamp
		(*dest)[i].Value = v.value
		v.res.Collect(&(*dest)[i].Exemplars)
		i++
	}
	// Do not report stale values.
	clear(s.values)
}
