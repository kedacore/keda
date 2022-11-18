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

package apiserver

import (
	"context"
	"fmt"

	cm_rest "sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver/registry/rest"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/endpoints/request"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/metrics/pkg/apis/custom_metrics"
)

type REST struct {
	cmProvider provider.CustomMetricsProvider
}

var _ rest.Storage = &REST{}
var _ cm_rest.ListerWithOptions = &REST{}

func NewREST(cmProvider provider.CustomMetricsProvider) *REST {
	return &REST{
		cmProvider: cmProvider,
	}
}

// Implement Storage

func (r *REST) New() runtime.Object {
	return &custom_metrics.MetricValue{}
}

// Implement ListerWithOptions

func (r *REST) NewList() runtime.Object {
	return &custom_metrics.MetricValueList{}
}

func (r *REST) NewListOptions() (runtime.Object, bool, string) {
	return &custom_metrics.MetricListOptions{}, true, "metricName"
}

func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions, metricOpts runtime.Object) (runtime.Object, error) {
	metricOptions, ok := metricOpts.(*custom_metrics.MetricListOptions)
	if !ok {
		return nil, fmt.Errorf("invalid options object: %#v", options)
	}

	// populate the label selector, defaulting to all
	selector := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		selector = options.LabelSelector
	}

	metricLabelSelector := labels.Everything()
	if metricOptions != nil && len(metricOptions.MetricLabelSelector) > 0 {
		sel, err := labels.Parse(metricOptions.MetricLabelSelector)
		if err != nil {
			return nil, err
		}
		metricLabelSelector = sel
	}

	// grab the name, if present, from the field selector list options
	// (this is how the list handler logic injects it)
	// (otherwise we'd have to write a custom list handler)
	name := "*"
	if options != nil && options.FieldSelector != nil {
		if nameMatch, required := options.FieldSelector.RequiresExactMatch("metadata.name"); required {
			name = nameMatch
		}
	}

	namespace := genericapirequest.NamespaceValue(ctx)

	requestInfo, ok := request.RequestInfoFrom(ctx)
	if !ok {
		return nil, fmt.Errorf("unable to get resource and metric name from request")
	}

	resourceRaw := requestInfo.Resource
	metricName := requestInfo.Subresource

	groupResource := schema.ParseGroupResource(resourceRaw)

	// handle metrics describing namespaces
	if namespace != "" && resourceRaw == "metrics" {
		// namespace-describing metrics have a path of /namespaces/$NS/metrics/$metric,
		groupResource = schema.GroupResource{Resource: "namespaces"}
		metricName = name
		name = namespace
		namespace = ""
	}

	// handle namespaced and root metrics
	if name == "*" {
		return r.handleWildcardOp(ctx, namespace, groupResource, selector, metricName, metricLabelSelector)
	} else {
		return r.handleIndividualOp(ctx, namespace, groupResource, name, metricName, metricLabelSelector)
	}
}

func (r *REST) handleIndividualOp(ctx context.Context, namespace string, groupResource schema.GroupResource, name string, metricName string, metricLabelSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	singleRes, err := r.cmProvider.GetMetricByName(ctx, types.NamespacedName{Namespace: namespace, Name: name}, provider.CustomMetricInfo{
		GroupResource: groupResource,
		Metric:        metricName,
		Namespaced:    namespace != "",
	}, metricLabelSelector)
	if err != nil {
		return nil, err
	}

	return &custom_metrics.MetricValueList{
		Items: []custom_metrics.MetricValue{*singleRes},
	}, nil
}

func (r *REST) handleWildcardOp(ctx context.Context, namespace string, groupResource schema.GroupResource, selector labels.Selector, metricName string, metricLabelSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	return r.cmProvider.GetMetricBySelector(ctx, namespace, selector, provider.CustomMetricInfo{
		GroupResource: groupResource,
		Metric:        metricName,
		Namespaced:    namespace != "",
	}, metricLabelSelector)
}
