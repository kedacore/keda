package provider

import (
	"context"
	"fmt"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	"github.com/kedacore/keda/pkg/handler"

	"github.com/go-logr/logr"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KedaProvider implements External Metrics Provider
type KedaProvider struct {
	client           client.Client
	values           map[provider.CustomMetricInfo]int64
	externalMetrics  []externalMetric
	scaleHandler     *handler.ScaleHandler
	watchedNamespace string
}
type externalMetric struct {
	info   provider.ExternalMetricInfo
	labels map[string]string
	value  external_metrics.ExternalMetricValue
}

var logger logr.Logger

// NewProvider returns an instance of KedaProvider
func NewProvider(adapterLogger logr.Logger, scaleHandler *handler.ScaleHandler, client client.Client, watchedNamespace string) provider.MetricsProvider {
	provider := &KedaProvider{
		values:           make(map[provider.CustomMetricInfo]int64),
		externalMetrics:  make([]externalMetric, 2, 10),
		client:           client,
		scaleHandler:     scaleHandler,
		watchedNamespace: watchedNamespace,
	}
	logger = adapterLogger.WithName("provider")
	logger.Info("starting")
	return provider
}

// GetExternalMetric retrieves metrics from the scalers
// Metric is normally identified by a name and a set of labels/tags. It is up to a specific
// implementation how to translate metricSelector to a filter for metric values.
// Namespace can be used by the implementation for metric identification, access control or ignored.
func (p *KedaProvider) GetExternalMetric(namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	// Note:
	//		metric name and namespace is used to lookup for the CRD which contains configuration to call azure
	// 		if not found then ignored and label selector is parsed for all the metrics
	logger.V(1).Info("Keda provider received request for external metrics", "namespace", namespace, "metric name", info.Metric, "metricSelector", metricSelector.String())
	selector, err := labels.ConvertSelectorToLabelsMap(metricSelector.String())
	if err != nil {
		logger.Error(err, "Error converting Selector to Labels Map")
		return nil, err
	}

	//get the scaled objects matching namespace and labels
	scaledObjects := &kedav1alpha1.ScaledObjectList{}
	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(selector),
	}
	err = p.client.List(context.TODO(), scaledObjects, opts...)
	if err != nil {
		return nil, err
	} else if len(scaledObjects.Items) != 1 {
		return nil, fmt.Errorf("Exactly one scaled object should match label %s", metricSelector.String())
	}

	scaledObject := &scaledObjects.Items[0]
	matchingMetrics := []external_metrics.ExternalMetricValue{}
	scalers, _, err := p.scaleHandler.GetDeploymentScalers(scaledObject)
	if err != nil {
		return nil, fmt.Errorf("Error when getting scalers %s", err)
	}

	for _, scaler := range scalers {
		metrics, err := scaler.GetMetrics(context.TODO(), info.Metric, metricSelector)
		if err != nil {
			logger.Error(err, "error getting metric for scaler", "ScaledObject.Namespace", scaledObject.Namespace, "ScaledObject.Name", scaledObject.Name, "Scaler", scaler)
		} else {
			matchingMetrics = append(matchingMetrics, metrics...)
		}

		scaler.Close()
	}

	return &external_metrics.ExternalMetricValueList{
		Items: matchingMetrics,
	}, nil

}

// ListAllExternalMetrics returns the supported external metrics for this provider
func (p *KedaProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {

	externalMetricsInfo := []provider.ExternalMetricInfo{}

	//get all ScaledObjects in namespace(s) watched by the operator
	scaledObjects := &kedav1alpha1.ScaledObjectList{}
	opts := []client.ListOption{
		client.InNamespace(p.watchedNamespace),
	}
	err := p.client.List(context.TODO(), scaledObjects, opts...)
	if err != nil {
		logger.Error(err, "Cannot get list of ScaledObjects", "WatchedNamespace", p.watchedNamespace)
		return nil
	}

	// get metrics from all watched ScaledObjects
	for _, scaledObject := range scaledObjects.Items {
		for _, metric := range scaledObject.Status.ExternalMetricNames {
			externalMetricsInfo = append(externalMetricsInfo, provider.ExternalMetricInfo{Metric: metric})
		}
	}
	return externalMetricsInfo
}

// GetMetricByName fetches a particular metric for a particular object.
// The namespace will be empty if the metric is root-scoped.
func (p *KedaProvider) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	// not implemented yet
	return nil, apiErrors.NewServiceUnavailable("not implemented yet")
}

// GetMetricBySelector fetches a particular metric for a set of objects matching
// the given label selector.  The namespace will be empty if the metric is root-scoped.
func (p *KedaProvider) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	logger.V(0).Info("Received request for custom metric", "groupresource", info.GroupResource.String(), "namespace", namespace, "metric name", info.Metric, "selector", selector.String())
	return nil, apiErrors.NewServiceUnavailable("not implemented yet")
}

// ListAllMetrics provides a list of all available metrics at
// the current time.  Note that this is not allowed to return
// an error, so it is recommended that implementors cache and
// periodically update this list, instead of querying every time.
func (p *KedaProvider) ListAllMetrics() []provider.CustomMetricInfo {
	// not implemented yet
	return []provider.CustomMetricInfo{}
}
