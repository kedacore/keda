/*
Copyright 2017 The Kubernetes Authors.

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

package provider

import (
	"context"
	"fmt"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

// CustomMetricInfo describes a metric for a particular
// fully-qualified group resource.
type CustomMetricInfo struct {
	GroupResource schema.GroupResource
	Namespaced    bool
	Metric        string
}

// ExternalMetricInfo describes a metric.
type ExternalMetricInfo struct {
	Metric string
}

func (i CustomMetricInfo) String() string {
	if i.Namespaced {
		return fmt.Sprintf("%s/%s(namespaced)", i.GroupResource.String(), i.Metric)
	}
	return fmt.Sprintf("%s/%s", i.GroupResource.String(), i.Metric)
}

// Normalized returns a copy of the current MetricInfo with the GroupResource resolved using the
// provided REST mapper, to ensure consistent pluralization, etc, for use when looking up or comparing
// the MetricInfo.  It also returns the singular form of the GroupResource associated with the given
// MetricInfo.
func (i CustomMetricInfo) Normalized(mapper apimeta.RESTMapper) (normalizedInfo CustomMetricInfo, singluarResource string, err error) {
	normalizedGroupRes, err := mapper.ResourceFor(i.GroupResource.WithVersion(""))
	if err != nil {
		return i, "", err
	}
	i.GroupResource = normalizedGroupRes.GroupResource()

	singularResource, err := mapper.ResourceSingularizer(i.GroupResource.Resource)
	if err != nil {
		return i, "", err
	}

	return i, singularResource, nil
}

// CustomMetricsProvider is a source of custom metrics
// which is able to supply a list of available metrics,
// as well as metric values themselves on demand.
//
// Note that group-resources are provided  as GroupResources,
// not GroupKinds.  This is to allow flexibility on the part
// of the implementor: implementors do not necessarily need
// to be aware of all existing kinds and their corresponding
// REST mappings in order to perform queries.
//
// For queries that use label selectors, it is up to the
// implementor to decide how to make use of the label selector --
// they may wish to query the main Kubernetes API server, or may
// wish to simply make use of stored information in their TSDB.
type CustomMetricsProvider interface {
	// GetMetricByName fetches a particular metric for a particular object.
	// The namespace will be empty if the metric is root-scoped.
	GetMetricByName(ctx context.Context, name types.NamespacedName, info CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error)

	// GetMetricBySelector fetches a particular metric for a set of objects matching
	// the given label selector.  The namespace will be empty if the metric is root-scoped.
	GetMetricBySelector(ctx context.Context, namespace string, selector labels.Selector, info CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error)

	// ListAllMetrics provides a list of all available metrics at
	// the current time.  Note that this is not allowed to return
	// an error, so it is recommended that implementors cache and
	// periodically update this list, instead of querying every time.
	ListAllMetrics() []CustomMetricInfo
}

// ExternalMetricsProvider is a source of external metrics.
// Metric is normally identified by a name and a set of labels/tags. It is up to a specific
// implementation how to translate metricSelector to a filter for metric values.
// Namespace can be used by the implemetation for metric identification, access control or ignored.
type ExternalMetricsProvider interface {
	GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error)

	ListAllExternalMetrics() []ExternalMetricInfo
}

type MetricsProvider interface {
	CustomMetricsProvider
	ExternalMetricsProvider
}
