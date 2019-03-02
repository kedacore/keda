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

package queue

import (
	"context"
	"errors"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Measurement type for reporting.
type Measurement int

const (
	// ViewReportingPeriod is the interval of time between reporting aggregated views.
	ViewReportingPeriod = time.Second
	// ReporterReportingPeriod is the interval of time between reporting stats by queue proxy.
	// It should be equal to or larger than ViewReportingPeriod so that no stat
	// will be dropped if LastValue aggregation is used for a view.
	ReporterReportingPeriod = time.Second

	operationsPerSecondN       = "operations_per_second"
	averageConcurrentRequestsN = "average_concurrent_requests"

	// OperationsPerSecondM number of operations per second.
	OperationsPerSecondM Measurement = iota
	// AverageConcurrentRequestsM average number of requests currently being handled by this pod.
	AverageConcurrentRequestsM
)

var (
	measurements = []*stats.Float64Measure{
		// TODO(#2524): make reporting period accurate.
		OperationsPerSecondM: stats.Float64(
			operationsPerSecondN,
			"Number of operations per second",
			stats.UnitNone),
		AverageConcurrentRequestsM: stats.Float64(
			averageConcurrentRequestsN,
			"Number of requests currently being handled by this pod",
			stats.UnitNone),
	}
)

// Reporter structure represents a prometheus exporter.
type Reporter struct {
	Initialized     bool
	ctx             context.Context
	configTagKey    tag.Key
	namespaceTagKey tag.Key
	revisionTagKey  tag.Key
	podTagKey       tag.Key
}

// NewStatsReporter creates a reporter that collects and reports queue metrics.
func NewStatsReporter(namespace string, config string, revision string, pod string) (*Reporter, error) {
	if len(namespace) < 1 {
		return nil, errors.New("Namespace must not be empty")
	}
	if len(config) < 1 {
		return nil, errors.New("Config must not be empty")
	}
	if len(revision) < 1 {
		return nil, errors.New("Revision must not be empty")
	}
	if len(pod) < 1 {
		return nil, errors.New("Pod must not be empty")
	}

	// Create the tag keys that will be used to add tags to our measurements.
	nsTag, err := tag.NewKey("destination_namespace")
	if err != nil {
		return nil, err
	}
	configTag, err := tag.NewKey("destination_configuration")
	if err != nil {
		return nil, err
	}
	revTag, err := tag.NewKey("destination_revision")
	if err != nil {
		return nil, err
	}
	podTag, err := tag.NewKey("destination_pod")
	if err != nil {
		return nil, err
	}

	// Create views to see our measurements. This can return an error if
	// a previously-registered view has the same name with a different value.
	// View name defaults to the measure name if unspecified.
	err = view.Register(
		&view.View{
			Description: "Number of requests received since last Stat",
			Measure:     measurements[OperationsPerSecondM],
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{nsTag, configTag, revTag, podTag},
		},
		&view.View{
			Description: "Number of requests currently being handled by this pod",
			Measure:     measurements[AverageConcurrentRequestsM],
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{nsTag, configTag, revTag, podTag},
		},
	)
	if err != nil {
		return nil, err
	}

	ctx, err := tag.New(
		context.Background(),
		tag.Insert(nsTag, namespace),
		tag.Insert(configTag, config),
		tag.Insert(revTag, revision),
		tag.Insert(podTag, pod),
	)
	if err != nil {
		return nil, err
	}
	return &Reporter{
		Initialized: true,

		ctx:             ctx,
		namespaceTagKey: nsTag,
		configTagKey:    configTag,
		revisionTagKey:  revTag,
		podTagKey:       podTag,
	}, nil
}

// Report captures request metrics.
func (r *Reporter) Report(operationsPerSecond float64, averageConcurrentRequests float64) error {
	if !r.Initialized {
		return errors.New("StatsReporter is not Initialized yet")
	}
	stats.Record(r.ctx, measurements[OperationsPerSecondM].M(operationsPerSecond))
	stats.Record(r.ctx, measurements[AverageConcurrentRequestsM].M(averageConcurrentRequests))
	return nil
}

// UnregisterViews Unregister views.
func (r *Reporter) UnregisterViews() error {
	if r.Initialized != true {
		return errors.New("Reporter is not initialized")
	}
	var views []*view.View
	if v := view.Find(operationsPerSecondN); v != nil {
		views = append(views, v)
	}
	if v := view.Find(averageConcurrentRequestsN); v != nil {
		views = append(views, v)
	}
	view.Unregister(views...)
	r.Initialized = false
	return nil
}
