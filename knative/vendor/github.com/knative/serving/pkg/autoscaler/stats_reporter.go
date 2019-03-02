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

package autoscaler

import (
	"context"
	"errors"

	"github.com/knative/pkg/metrics/metricskey"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	desiredPodCountM = stats.Int64(
		"desired_pods",
		"Number of pods autoscaler wants to allocate",
		stats.UnitDimensionless)
	requestedPodCountM = stats.Int64(
		"requested_pods",
		"Number of pods autoscaler requested from Kubernetes",
		stats.UnitDimensionless)
	actualPodCountM = stats.Int64(
		"actual_pods",
		"Number of pods that are allocated currently",
		stats.UnitDimensionless)
	observedPodCountM = stats.Float64(
		"observed_pods",
		"Number of pods that are observed currently",
		stats.UnitDimensionless)
	stableRequestConcurrencyM = stats.Float64(
		"stable_request_concurrency",
		"Average of requests count per observed pod in each stable window (default 60 seconds)",
		stats.UnitDimensionless)
	panicRequestConcurrencyM = stats.Float64(
		"panic_request_concurrency",
		"Average of requests count per observed pod in each panic window (default 6 seconds)",
		stats.UnitDimensionless)
	targetRequestConcurrencyM = stats.Float64(
		"target_concurrency_per_pod",
		"The desired number of concurrent requests for each pod",
		stats.UnitDimensionless)
	panicM = stats.Int64(
		"panic_mode",
		"1 if autoscaler is in panic mode, 0 otherwise",
		stats.UnitDimensionless)
	namespaceTagKey tag.Key
	configTagKey    tag.Key
	revisionTagKey  tag.Key
	serviceTagKey   tag.Key
)

func init() {
	var err error
	// Create the tag keys that will be used to add tags to our measurements.
	// Tag keys must conform to the restrictions described in
	// go.opencensus.io/tag/validate.go. Currently those restrictions are:
	// - length between 1 and 255 inclusive
	// - characters are printable US-ASCII
	namespaceTagKey, err = tag.NewKey(metricskey.LabelNamespaceName)
	if err != nil {
		panic(err)
	}
	serviceTagKey, err = tag.NewKey(metricskey.LabelServiceName)
	if err != nil {
		panic(err)
	}
	configTagKey, err = tag.NewKey(metricskey.LabelConfigurationName)
	if err != nil {
		panic(err)
	}
	revisionTagKey, err = tag.NewKey(metricskey.LabelRevisionName)
	if err != nil {
		panic(err)
	}

	// Create views to see our measurements. This can return an error if
	// a previously-registered view has the same name with a different value.
	// View name defaults to the measure name if unspecified.
	err = view.Register(
		&view.View{
			Description: "Number of pods autoscaler wants to allocate",
			Measure:     desiredPodCountM,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{namespaceTagKey, serviceTagKey, configTagKey, revisionTagKey},
		},
		&view.View{
			Description: "Number of pods autoscaler requested from Kubernetes",
			Measure:     requestedPodCountM,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{namespaceTagKey, serviceTagKey, configTagKey, revisionTagKey},
		},
		&view.View{
			Description: "Number of pods that are allocated currently",
			Measure:     actualPodCountM,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{namespaceTagKey, serviceTagKey, configTagKey, revisionTagKey},
		},
		&view.View{
			Description: "Number of pods that are observed currently",
			Measure:     observedPodCountM,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{namespaceTagKey, serviceTagKey, configTagKey, revisionTagKey},
		},
		&view.View{
			Description: "Average of requests count in each 60 second stable window",
			Measure:     stableRequestConcurrencyM,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{namespaceTagKey, serviceTagKey, configTagKey, revisionTagKey},
		},
		&view.View{
			Description: "Average of requests count in each 6 second panic window",
			Measure:     panicRequestConcurrencyM,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{namespaceTagKey, serviceTagKey, configTagKey, revisionTagKey},
		},
		&view.View{
			Description: "The desired number of concurrent requests for each pod",
			Measure:     targetRequestConcurrencyM,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{namespaceTagKey, serviceTagKey, configTagKey, revisionTagKey},
		},
		&view.View{
			Description: "1 if autoscaler is in panic mode, 0 otherwise",
			Measure:     panicM,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{namespaceTagKey, serviceTagKey, configTagKey, revisionTagKey},
		},
	)
	if err != nil {
		panic(err)
	}

}

// StatsReporter defines the interface for sending autoscaler metrics
type StatsReporter interface {
	ReportDesiredPodCount(v int64) error
	ReportRequestedPodCount(v int64) error
	ReportActualPodCount(v int64) error
	ReportObservedPodCount(v float64) error
	ReportStableRequestConcurrency(v float64) error
	ReportPanicRequestConcurrency(v float64) error
	ReportTargetRequestConcurrency(v float64) error
	ReportPanic(v int64) error
}

// Reporter holds cached metric objects to report autoscaler metrics
type Reporter struct {
	ctx         context.Context
	initialized bool
}

// NewStatsReporter creates a reporter that collects and reports autoscaler metrics
func NewStatsReporter(podNamespace string, service string, config string, revision string) (*Reporter, error) {

	r := &Reporter{}

	// Our tags are static. So, we can get away with creating a single context
	// and reuse it for reporting all of our metrics.
	ctx, err := tag.New(
		context.Background(),
		tag.Insert(namespaceTagKey, podNamespace),
		tag.Insert(serviceTagKey, service),
		tag.Insert(configTagKey, config),
		tag.Insert(revisionTagKey, revision))
	if err != nil {
		return nil, err
	}

	r.ctx = ctx
	r.initialized = true
	return r, nil
}

// ReportDesiredPodCount captures value v for desired pod count measure.
func (r *Reporter) ReportDesiredPodCount(v int64) error {
	return r.report(desiredPodCountM.M(v))
}

// ReportRequestedPodCount captures value v for requested pod count measure.
func (r *Reporter) ReportRequestedPodCount(v int64) error {
	return r.report(requestedPodCountM.M(v))
}

// ReportActualPodCount captures value v for actual pod count measure.
func (r *Reporter) ReportActualPodCount(v int64) error {
	return r.report(actualPodCountM.M(v))
}

// ReportObservedPodCount captures value v for observed pod count measure.
func (r *Reporter) ReportObservedPodCount(v float64) error {
	return r.report(observedPodCountM.M(v))
}

// ReportStableRequestConcurrency captures value v for stable request concurrency measure.
func (r *Reporter) ReportStableRequestConcurrency(v float64) error {
	return r.report(stableRequestConcurrencyM.M(v))
}

// ReportPanicRequestConcurrency captures value v for panic request concurrency measure.
func (r *Reporter) ReportPanicRequestConcurrency(v float64) error {
	return r.report(panicRequestConcurrencyM.M(v))
}

// ReportTargetRequestConcurrency captures value v for target request concurrency measure.
func (r *Reporter) ReportTargetRequestConcurrency(v float64) error {
	return r.report(targetRequestConcurrencyM.M(v))
}

// ReportPanic captures value v for panic mode measure.
func (r *Reporter) ReportPanic(v int64) error {
	return r.report(panicM.M(v))
}

func (r *Reporter) report(m stats.Measurement) error {
	if !r.initialized {
		return errors.New("StatsReporter is not initialized yet")
	}

	stats.Record(r.ctx, m)
	return nil
}
