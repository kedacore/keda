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

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
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
var _ rest.Lister = &REST{}

func NewREST(cmProvider provider.CustomMetricsProvider) *REST {
	return &REST{
		cmProvider: cmProvider,
	}
}

// Implement Storage

func (r *REST) New() runtime.Object {
	return &custom_metrics.MetricValue{}
}

// Implement Lister

func (r *REST) NewList() runtime.Object {
	return &custom_metrics.MetricValueList{}
}

func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	// populate the label selector, defaulting to all
	selector := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		selector = options.LabelSelector
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
		return r.handleWildcardOp(namespace, groupResource, selector, metricName)
	} else {
		return r.handleIndividualOp(namespace, groupResource, name, metricName)
	}
}

func (r *REST) handleIndividualOp(namespace string, groupResource schema.GroupResource, name string, metricName string) (*custom_metrics.MetricValueList, error) {
	singleRes, err := r.cmProvider.GetMetricByName(types.NamespacedName{Namespace: namespace, Name: name}, provider.CustomMetricInfo{
		GroupResource: groupResource,
		Metric:        metricName,
		Namespaced:    namespace != "",
	})
	if err != nil {
		return nil, err
	}

	return &custom_metrics.MetricValueList{
		Items: []custom_metrics.MetricValue{*singleRes},
	}, nil
}

func (r *REST) handleWildcardOp(namespace string, groupResource schema.GroupResource, selector labels.Selector, metricName string) (*custom_metrics.MetricValueList, error) {
	return r.cmProvider.GetMetricBySelector(namespace, selector, provider.CustomMetricInfo{
		GroupResource: groupResource,
		Metric:        metricName,
		Namespaced:    namespace != "",
	})
}
