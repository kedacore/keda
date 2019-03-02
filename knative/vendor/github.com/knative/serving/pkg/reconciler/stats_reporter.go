/*
Copyright 2018 The Knative Authors

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

package reconciler

import (
	"context"
	"fmt"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

type Measurement int

const (
	// ServiceReadyCountN is the number of services that have become ready.
	ServiceReadyCountN = "service_ready_count"
	// ServiceReadyLatencyN is the time it takes for a service to become ready since the resource is created.
	ServiceReadyLatencyN = "service_ready_latency"
)

var (
	serviceReadyLatencyStat = stats.Int64(
		ServiceReadyLatencyN,
		"Time it takes for a service to become ready since created",
		stats.UnitMilliseconds)
	serviceReadyCountStat = stats.Int64(
		ServiceReadyCountN,
		"Number of services that became ready",
		stats.UnitDimensionless)

	reconcilerTagKey tag.Key
	keyTagKey        tag.Key
)

func init() {
	var err error
	// Create the tag keys that will be used to add tags to our measurements.
	// Tag keys must conform to the restrictions described in
	// go.opencensus.io/tag/validate.go. Currently those restrictions are:
	// - length between 1 and 255 inclusive
	// - characters are printable US-ASCII
	reconcilerTagKey = mustNewTagKey("reconciler")
	keyTagKey = mustNewTagKey("key")

	// Create views to see our measurements. This can return an error if
	// a previously-registered view has the same name with a different value.
	// View name defaults to the measure name if unspecified.
	err = view.Register(
		&view.View{
			Description: serviceReadyCountStat.Description(),
			Measure:     serviceReadyCountStat,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{reconcilerTagKey, keyTagKey},
		},
		&view.View{
			Description: serviceReadyLatencyStat.Description(),
			Measure:     serviceReadyLatencyStat,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{reconcilerTagKey, keyTagKey},
		},
	)
	if err != nil {
		panic(err)
	}
}

// StatsReporter reports reconcilers' metrics.
type StatsReporter interface {
	// ReportServiceReady reports the time it took a service to become Ready.
	ReportServiceReady(namespace, service string, d time.Duration) error
}

type reporter struct {
	ctx context.Context
}

// NewStatsReporter creates a reporter for reconcilers' metrics
func NewStatsReporter(reconciler string) (StatsReporter, error) {
	ctx, err := tag.New(
		context.Background(),
		tag.Insert(reconcilerTagKey, reconciler))
	if err != nil {
		return nil, err
	}
	return &reporter{ctx: ctx}, nil
}

// ReportServiceReady reports the time it took a service to become Ready
func (r *reporter) ReportServiceReady(namespace, service string, d time.Duration) error {
	key := fmt.Sprintf("%s/%s", namespace, service)
	v := int64(d / time.Millisecond)
	ctx, err := tag.New(
		r.ctx,
		tag.Insert(keyTagKey, key))
	if err != nil {
		return err
	}

	stats.Record(ctx, serviceReadyCountStat.M(1))
	stats.Record(ctx, serviceReadyLatencyStat.M(v))
	return nil
}

func mustNewTagKey(s string) tag.Key {
	tagKey, err := tag.NewKey(s)
	if err != nil {
		panic(err)
	}
	return tagKey
}
