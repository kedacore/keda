/*
Copyright 2021 The KEDA Authors

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
	"strings"
	"sync"

	"github.com/go-logr/logr"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/prommetrics"
	"github.com/kedacore/keda/v2/pkg/scaling"
)

// KedaProvider implements External Metrics Provider
type KedaProvider struct {
	client                  client.Client
	scaleHandler            scaling.ScaleHandler
	watchedNamespace        string
	ctx                     context.Context
	externalMetricsInfo     *[]provider.ExternalMetricInfo
	externalMetricsInfoLock *sync.RWMutex
}

var (
	logger        logr.Logger
	metricsServer prommetrics.PrometheusMetricServer
)

// NewProvider returns an instance of KedaProvider
func NewProvider(ctx context.Context, adapterLogger logr.Logger, scaleHandler scaling.ScaleHandler, client client.Client, watchedNamespace string, externalMetricsInfo *[]provider.ExternalMetricInfo, externalMetricsInfoLock *sync.RWMutex) provider.MetricsProvider {
	provider := &KedaProvider{
		client:                  client,
		scaleHandler:            scaleHandler,
		watchedNamespace:        watchedNamespace,
		ctx:                     ctx,
		externalMetricsInfo:     externalMetricsInfo,
		externalMetricsInfoLock: externalMetricsInfoLock,
	}
	logger = adapterLogger.WithName("provider")
	logger.Info("starting")
	return provider
}

// GetExternalMetric retrieves metrics from the scalers
// Metric is normally identified by a name and a set of labels/tags. It is up to a specific
// implementation how to translate metricSelector to a filter for metric values.
// Namespace can be used by the implementation for metric identification, access control or ignored.
func (p *KedaProvider) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	// Note:
	//		metric name and namespace is used to lookup for the CRD which contains configuration
	// 		if not found then ignored and label selector is parsed for all the metrics
	logger.V(1).Info("KEDA Metrics Server received request for external metrics", "namespace", namespace, "metric name", info.Metric, "metricSelector", metricSelector.String())
	selector, err := labels.ConvertSelectorToLabelsMap(metricSelector.String())
	if err != nil {
		logger.Error(err, "error converting Selector to Labels Map")
		return nil, err
	}

	// get the scaled objects matching namespace and labels
	scaledObjects := &kedav1alpha1.ScaledObjectList{}
	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(selector),
	}
	err = p.client.List(ctx, scaledObjects, opts...)
	if err != nil {
		return nil, err
	} else if len(scaledObjects.Items) != 1 {
		return nil, fmt.Errorf("exactly one ScaledObject should match label %s", metricSelector.String())
	}

	scaledObject := &scaledObjects.Items[0]
	var matchingMetrics []external_metrics.ExternalMetricValue

	cache, err := p.scaleHandler.GetScalersCache(ctx, scaledObject)
	metricsServer.RecordScalerObjectError(scaledObject.Namespace, scaledObject.Name, err)
	if err != nil {
		return nil, fmt.Errorf("error when getting scalers %s", err)
	}

	scalerError := false

	for scalerIndex, scalerPair := range cache.GetScalers() {
		metricSpecs := scalerPair.Scaler.GetMetricSpecForScaling(ctx)
		scalerName := strings.Replace(fmt.Sprintf("%T", scalerPair.Scaler), "*scalers.", "", 1)
		triggerName := scalerPair.TriggerName

		for _, metricSpec := range metricSpecs {
			// skip cpu/memory resource scaler
			if metricSpec.External == nil {
				continue
			}
			// Filter only the desired metric
			if strings.EqualFold(metricSpec.External.Metric.Name, info.Metric) {
				metrics, err := cache.GetMetricsForScaler(ctx, scalerIndex, info.Metric, metricSelector)
				metrics, err = p.getMetricsWithFallback(ctx, metrics, err, info.Metric, scaledObject, metricSpec)
				if err != nil {
					scalerError = true
					logger.Error(err, "error getting metric for scaler", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name, "scaler", scalerPair.Scaler)
				} else {
					for _, metric := range metrics {
						metricValue := metric.Value.AsApproximateFloat64()
						metricsServer.RecordHPAScalerMetric(namespace, scaledObject.Name, scalerName, scalerIndex, triggerName, metric.MetricName, metricValue)
					}
					matchingMetrics = append(matchingMetrics, metrics...)
				}
				metricsServer.RecordHPAScalerError(namespace, scaledObject.Name, scalerName, scalerIndex, triggerName, info.Metric, err)
			}
		}
	}

	// invalidate the cache for the ScaledObject, if we hit an error in any scaler
	// in this case we try to build all scalers (and resolve all secrets/creds) again in the next call
	if scalerError {
		err := p.scaleHandler.ClearScalersCache(ctx, scaledObject)
		if err != nil {
			logger.Error(err, "error clearing scalers cache")
		}
		logger.V(1).Info("scaler error encountered, clearing scaler cache")
	}

	if len(matchingMetrics) == 0 {
		return nil, fmt.Errorf("no matching metrics found for " + info.Metric)
	}

	return &external_metrics.ExternalMetricValueList{
		Items: matchingMetrics,
	}, nil
}

// ListAllExternalMetrics returns the supported external metrics for this provider
func (p *KedaProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	logger.V(1).Info("KEDA Metrics Server received request for list of all provided external metrics names")

	p.externalMetricsInfoLock.RLock()
	defer p.externalMetricsInfoLock.RUnlock()
	externalMetricsInfo := *p.externalMetricsInfo

	return externalMetricsInfo
}

// GetMetricByName fetches a particular metric for a particular object.
// The namespace will be empty if the metric is root-scoped.
func (p *KedaProvider) GetMetricByName(ctx context.Context, name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	// not implemented yet
	return nil, apiErrors.NewServiceUnavailable("not implemented yet")
}

// GetMetricBySelector fetches a particular metric for a set of objects matching
// the given label selector.  The namespace will be empty if the metric is root-scoped.
func (p *KedaProvider) GetMetricBySelector(ctx context.Context, namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	logger.V(0).Info("Received request for custom metric, which is not supported by this adapter", "groupresource", info.GroupResource.String(), "namespace", namespace, "metric name", info.Metric, "selector", selector.String())
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
