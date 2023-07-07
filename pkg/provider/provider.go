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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider/defaults"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/metricsservice"
)

// KedaProvider implements External Metrics Provider
type KedaProvider struct {
	defaults.DefaultExternalMetricsProvider

	client           client.Client
	watchedNamespace string
	ctx              context.Context

	grpcClient            metricsservice.GrpcClient
	useMetricsServiceGrpc bool
}

var (
	logger logr.Logger

	grpcClientConnected bool
)

// NewProvider returns an instance of KedaProvider
func NewProvider(ctx context.Context, adapterLogger logr.Logger, client client.Client, grpcClient metricsservice.GrpcClient, useMetricsServiceGrpc bool, watchedNamespace string) provider.ExternalMetricsProvider {
	provider := &KedaProvider{
		client:                client,
		watchedNamespace:      watchedNamespace,
		ctx:                   ctx,
		grpcClient:            grpcClient,
		useMetricsServiceGrpc: useMetricsServiceGrpc,
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

	metrics, err := p.grpcClient.GetMetrics(ctx, scaledObjectName, namespace, info.Metric)
	logger.V(1).WithValues("scaledObjectName", scaledObjectName, "scaledObjectNamespace", namespace, "metrics", metrics).Info("Receiving metrics")

	return metrics, err
}
