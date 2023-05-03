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
	"github.com/kedacore/keda/v2/pkg/fallback"
	"github.com/kedacore/keda/v2/pkg/metricsservice"
	prommetrics "github.com/kedacore/keda/v2/pkg/prommetrics/adapter"
	"github.com/kedacore/keda/v2/pkg/scaling"
)

//	prommetrics "github.com/kedacore/keda/v2/pkg/prommetrics/adapter"

// KedaProvider implements External Metrics Provider
type KedaProvider struct {
	client                  client.Client
	scaleHandler            scaling.ScaleHandler
	watchedNamespace        string
	ctx                     context.Context
	externalMetricsInfo     *[]provider.ExternalMetricInfo
	externalMetricsInfoLock *sync.RWMutex

	grpcClient            metricsservice.GrpcClient
	useMetricsServiceGrpc bool
}

var (
	logger            logr.Logger
	promMetricsServer prommetrics.PrometheusMetricServer

	grpcClientConnected bool
)

// NewProvider returns an instance of KedaProvider
func NewProvider(ctx context.Context, adapterLogger logr.Logger, scaleHandler scaling.ScaleHandler, client client.Client, grpcClient metricsservice.GrpcClient, useMetricsServiceGrpc bool, watchedNamespace string, externalMetricsInfo *[]provider.ExternalMetricInfo, externalMetricsInfoLock *sync.RWMutex) provider.MetricsProvider {
	provider := &KedaProvider{
		client:                  client,
		scaleHandler:            scaleHandler,
		watchedNamespace:        watchedNamespace,
		ctx:                     ctx,
		externalMetricsInfo:     externalMetricsInfo,
		externalMetricsInfoLock: externalMetricsInfoLock,
		grpcClient:              grpcClient,
		useMetricsServiceGrpc:   useMetricsServiceGrpc,
	}
	logger = adapterLogger.WithName("provider")
	logger.Info("starting")

	go func() {
		if !grpcClient.WaitForConnectionReady(ctx, logger) {
			grpcClientConnected = false
			logger.Error(fmt.Errorf("timeout while waiting to establish gRPC connection to KEDA Metrics Service server"), "timeout", "server", grpcClient.GetServerURL())
		} else if !grpcClientConnected {
			grpcClientConnected = true
			logger.Info("Connection to KEDA Metrics Service gRPC server has been successfully established", "server", grpcClient.GetServerURL())
		}
	}()

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

	// Get Metrics from Metrics Service gRPC Server
	if p.useMetricsServiceGrpc {
		if !p.grpcClient.WaitForConnectionReady(ctx, logger) {
			grpcClientConnected = false
			logger.Error(fmt.Errorf("timeout while waiting to establish gRPC connection to KEDA Metrics Service server"), "timeout", "server", p.grpcClient.GetServerURL())
			return nil, err
		}
		if !grpcClientConnected {
			grpcClientConnected = true
			logger.Info("Connection to KEDA Metrics Service gRPC server has been successfully established", "server", p.grpcClient.GetServerURL())
		}

		// selector is in form: `scaledobject.keda.sh/name: scaledobject-name`
		scaledObjectName := selector.Get(kedav1alpha1.ScaledObjectOwnerAnnotation)
		if scaledObjectName == "" {
			err := fmt.Errorf("scaledObject name is not specified")
			logger.Error(err, fmt.Sprintf("please specify scaledObject name, it needs to be set as value of label selector %q on the query", kedav1alpha1.ScaledObjectOwnerAnnotation))

			return &external_metrics.ExternalMetricValueList{}, err
		}

		metrics, promMetrics, err := p.grpcClient.GetMetrics(ctx, scaledObjectName, namespace, info.Metric)
		logger.V(1).WithValues("scaledObjectName", scaledObjectName, "scaledObjectNamespace", namespace, "metrics", metrics).Info("Receiving metrics")

		// [DEPRECATED] handle exporting Prometheus metrics from Operator to Metrics Server
		if promMetrics != nil {
			var scaledObjectErr error
			if promMetrics.ScaledObjectErr {
				scaledObjectErr = fmt.Errorf("scaledObject error")
			}
			promMetricsServer.RecordScaledObjectError(namespace, scaledObjectName, scaledObjectErr)
			for _, scalerMetric := range promMetrics.ScalerMetric {
				promMetricsServer.RecordHPAScalerMetric(namespace, scaledObjectName, scalerMetric.ScalerName, int(scalerMetric.ScalerIndex), scalerMetric.MetricName, float64(scalerMetric.MetricValue))
			}
			for _, scalerError := range promMetrics.ScalerError {
				var scalerErr error
				if scalerError.Error {
					scalerErr = fmt.Errorf("scaler error")
				}
				promMetricsServer.RecordHPAScalerError(namespace, scaledObjectName, scalerError.ScalerName, int(scalerError.ScalerIndex), scalerError.MetricName, scalerErr)
			}
		}

		return metrics, err
	}

	// ------ Deprecated way of getting metric directly from MS ------ //
	// --------------------------------------------------------------- //
	// Get Metrics by querying directly the external service
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
	promMetricsServer.RecordScaledObjectError(scaledObject.Namespace, scaledObject.Name, err)
	if err != nil {
		return nil, fmt.Errorf("error when getting scalers %w", err)
	}

	// let's check metrics for all scalers in a ScaledObject
	scalerError := false
	scalers, scalerConfigs := cache.GetScalers()
	for scalerIndex := 0; scalerIndex < len(scalers); scalerIndex++ {
		metricSpecs := scalers[scalerIndex].GetMetricSpecForScaling(ctx)
		scalerName := strings.Replace(fmt.Sprintf("%T", scalers[scalerIndex]), "*scalers.", "", 1)
		if scalerConfigs[scalerIndex].TriggerName != "" {
			scalerName = scalerConfigs[scalerIndex].TriggerName
		}

		for _, metricSpec := range metricSpecs {
			// skip cpu/memory resource scaler
			if metricSpec.External == nil {
				continue
			}
			// Filter only the desired metric
			if strings.EqualFold(metricSpec.External.Metric.Name, info.Metric) {
				metrics, _, _, err := cache.GetMetricsAndActivityForScaler(ctx, scalerIndex, info.Metric)
				metrics, err = fallback.GetMetricsWithFallback(ctx, p.client, metrics, err, info.Metric, scaledObject, metricSpec)
				if err != nil {
					scalerError = true
					logger.Error(err, "error getting metric for scaler", "scaledObject.Namespace", scaledObject.Namespace, "scaledObject.Name", scaledObject.Name, "scaler", scalerName)
				} else {
					for _, metric := range metrics {
						metricValue := metric.Value.AsApproximateFloat64()
						promMetricsServer.RecordHPAScalerMetric(namespace, scaledObject.Name, scalerName, scalerIndex, metric.MetricName, metricValue)
					}
					matchingMetrics = append(matchingMetrics, metrics...)
				}
				promMetricsServer.RecordHPAScalerError(namespace, scaledObject.Name, scalerName, scalerIndex, info.Metric, err)
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
func (p *KedaProvider) GetMetricByName(_ context.Context, _ types.NamespacedName, _ provider.CustomMetricInfo, _ labels.Selector) (*custom_metrics.MetricValue, error) {
	// not implemented yet
	return nil, apiErrors.NewServiceUnavailable("not implemented yet")
}

// GetMetricBySelector fetches a particular metric for a set of objects matching
// the given label selector.  The namespace will be empty if the metric is root-scoped.
func (p *KedaProvider) GetMetricBySelector(_ context.Context, namespace string, selector labels.Selector, info provider.CustomMetricInfo, _ labels.Selector) (*custom_metrics.MetricValueList, error) {
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
