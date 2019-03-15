package provider

import (
	"github.com/Azure/Kore/pkg/handler"
	"github.com/golang/glog"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type KoreProvider struct {
	client          dynamic.Interface
	mapper          apimeta.RESTMapper
	values          map[provider.CustomMetricInfo]int64
	externalMetrics []externalMetric
	scaleHandler    *handler.ScaleHandler
}
type externalMetric struct {
	info   provider.ExternalMetricInfo
	labels map[string]string
	value  external_metrics.ExternalMetricValue
}

// NewProvider returns an instance of KoreProvider
func NewProvider(client dynamic.Interface, mapper apimeta.RESTMapper, handler *handler.ScaleHandler) provider.MetricsProvider {
	provider := &KoreProvider{
		client:          client,
		mapper:          mapper,
		values:          make(map[provider.CustomMetricInfo]int64),
		externalMetrics: make([]externalMetric, 2, 10),
		scaleHandler:    handler,
	}
	return provider
}

// GetExternalMetric retrieves metrics from the scalers
// Metric is normally identified by a name and a set of labels/tags. It is up to a specific
// implementation how to translate metricSelector to a filter for metric values.
// Namespace can be used by the implementation for metric identification, access control or ignored.
func (p *KoreProvider) GetExternalMetric(namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	// Note:
	//		metric name and namespace is used to lookup for the CRD which contains configuration to call azure
	// 		if not found then ignored and label selector is parsed for all the metrics
	glog.V(0).Infof("Received request for namespace: %s, metric name: %s, metric selectors: %s", namespace, info.Metric, metricSelector.String())

	externalmetrics, error := p.scaleHandler.GetScaledObjectMetrics(namespace, metricSelector, info.Metric)
	if error != nil {
		return nil, error
	}

	matchingMetrics := []external_metrics.ExternalMetricValue{}
	matchingMetrics = append(matchingMetrics, externalmetrics...)

	return &external_metrics.ExternalMetricValueList{
		Items: matchingMetrics,
	}, nil

}

func (p *KoreProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	externalMetricsInfo := []provider.ExternalMetricInfo{}

	// not implemented yet

	// TODO
	// iterate over all of the resources we have access
	// build metric info from https://docs.microsoft.com/en-us/azure/monitoring-and-diagnostics/monitoring-rest-api-walkthrough#retrieve-metric-definitions-multi-dimensional-api
	// important to remember to cache this and only get it at given interval

	return externalMetricsInfo
}

// GetMetricByName fetches a particular metric for a particular object.
// The namespace will be empty if the metric is root-scoped.
func (p *KoreProvider) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo) (*custom_metrics.MetricValue, error) {
	// not implemented yet
	return nil, apiErrors.NewServiceUnavailable("not implemented yet")
}

// GetMetricBySelector fetches a particular metric for a set of objects matching
// the given label selector.  The namespace will be empty if the metric is root-scoped.
func (p *KoreProvider) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo) (*custom_metrics.MetricValueList, error) {
	glog.V(0).Infof("Received request for custom metric: groupresource: %s, namespace: %s, metric name: %s, selectors: %s", info.GroupResource.String(), namespace, info.Metric, selector.String())
	return nil, apiErrors.NewServiceUnavailable("not implemented yet")
}

// ListAllMetrics provides a list of all available metrics at
// the current time.  Note that this is not allowed to return
// an error, so it is reccomended that implementors cache and
// periodically update this list, instead of querying every time.
func (p *KoreProvider) ListAllMetrics() []provider.CustomMetricInfo {
	// not implemented yet
	return []provider.CustomMetricInfo{}
}
