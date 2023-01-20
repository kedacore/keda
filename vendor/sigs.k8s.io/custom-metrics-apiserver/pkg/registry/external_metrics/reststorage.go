/*
Copyright 2018 The Kubernetes Authors.

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

package apiserver

import (
	"context"
	"fmt"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver/metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
)

// REST is a wrapper for CustomMetricsProvider that provides implementation for Storage and Lister
// interfaces.
type REST struct {
	emProvider        provider.ExternalMetricsProvider
	freshnessObserver metrics.FreshnessObserver
	rest.TableConvertor
}

var _ rest.Storage = &REST{}
var _ rest.Lister = &REST{}

// NewREST returns new REST object for provided CustomMetricsProvider.
func NewREST(emProvider provider.ExternalMetricsProvider) *REST {
	freshnessObserver := metrics.NewFreshnessObserver(external_metrics.GroupName)
	return &REST{
		emProvider:        emProvider,
		freshnessObserver: freshnessObserver,
	}
}

// Implement Storage

// New returns empty MetricValue.
func (r *REST) New() runtime.Object {
	return &external_metrics.ExternalMetricValue{}
}

func (r *REST) Destroy() {
}

// Implement Lister

// NewList returns empty MetricValueList.
func (r *REST) NewList() runtime.Object {
	return &external_metrics.ExternalMetricValueList{}
}

// List selects resources in the storage which match to the selector.
func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	// populate the label selector, defaulting to all
	metricSelector := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		metricSelector = options.LabelSelector
	}

	namespace := request.NamespaceValue(ctx)

	requestInfo, ok := request.RequestInfoFrom(ctx)
	if !ok {
		return nil, fmt.Errorf("unable to get resource and metric name from request")
	}
	metricName := requestInfo.Resource

	res, err := r.emProvider.GetExternalMetric(ctx, namespace, metricSelector, provider.ExternalMetricInfo{Metric: metricName})
	if err != nil {
		return nil, err
	}

	for _, m := range res.Items {
		r.freshnessObserver.Observe(m.Timestamp)
	}

	return res, nil
}
