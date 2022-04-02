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

package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Azure/azure-kusto-go/kusto"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type azureDataExplorerScaler struct {
	metricType v2beta2.MetricTargetType
	metadata   *azure.DataExplorerMetadata
	client     *kusto.Client
	name       string
	namespace  string
}

const adxName = "azure-data-explorer"

var dataExplorerLogger = logf.Log.WithName("azure_data_explorer_scaler")

func NewAzureDataExplorerScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	metadata, err := parseAzureDataExplorerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse azure data explorer metadata: %s", err)
	}

	client, err := azure.CreateAzureDataExplorerClient(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure data explorer client: %s", err)
	}

	return &azureDataExplorerScaler{
		metricType: metricType,
		metadata:   metadata,
		client:     client,
		name:       config.Name,
		namespace:  config.Namespace,
	}, nil
}

func parseAzureDataExplorerMetadata(config *ScalerConfig) (*azure.DataExplorerMetadata, error) {
	metadata, err := parseAzureDataExplorerAuthParams(config)
	if err != nil {
		return nil, err
	}

	// Get database name.
	databaseName, err := getParameterFromConfig(config, "databaseName", false)
	if err != nil {
		return nil, err
	}
	metadata.DatabaseName = databaseName

	// Get endpoint.
	endpoint, err := getParameterFromConfig(config, "endpoint", false)
	if err != nil {
		return nil, err
	}
	metadata.Endpoint = endpoint

	// Get query.
	query, err := getParameterFromConfig(config, "query", false)
	if err != nil {
		return nil, err
	}
	metadata.Query = query

	// Get threshold.
	if val, ok := config.TriggerMetadata["threshold"]; ok {
		threshold, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing metadata. Details: can't parse threshold. Inner Error: %v", err)
		}
		metadata.Threshold = threshold
	}

	// Generate metricName.
	metadata.MetricName = GenerateMetricNameWithIndex(config.ScalerIndex, kedautil.NormalizeString(fmt.Sprintf("%s-%s", adxName, metadata.DatabaseName)))

	activeDirectoryEndpoint, err := azure.ParseActiveDirectoryEndpoint(config.TriggerMetadata)
	if err != nil {
		return nil, err
	}
	metadata.ActiveDirectoryEndpoint = activeDirectoryEndpoint

	dataExplorerLogger.V(1).Info("Parsed azureDataExplorerMetadata",
		"database", metadata.DatabaseName,
		"endpoint", metadata.Endpoint,
		"metricName", metadata.MetricName,
		"query", metadata.Query,
		"threshold", metadata.Threshold,
		"activeDirectoryEndpoint", metadata.ActiveDirectoryEndpoint,
	)

	return metadata, nil
}

func parseAzureDataExplorerAuthParams(config *ScalerConfig) (*azure.DataExplorerMetadata, error) {
	metadata := azure.DataExplorerMetadata{}

	switch config.PodIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzure:
		metadata.PodIdentity = config.PodIdentity
	case "", kedav1alpha1.PodIdentityProviderNone:
		dataExplorerLogger.V(1).Info("Pod Identity is not provided. Trying to resolve clientId, clientSecret and tenantId.")

		tenantID, err := getParameterFromConfig(config, "tenantId", true)
		if err != nil {
			return nil, err
		}
		metadata.TenantID = tenantID

		clientID, err := getParameterFromConfig(config, "clientId", true)
		if err != nil {
			return nil, err
		}
		metadata.ClientID = clientID

		clientSecret, err := getParameterFromConfig(config, "clientSecret", true)
		if err != nil {
			return nil, err
		}
		metadata.ClientSecret = clientSecret
	default:
		return nil, fmt.Errorf("error parsing auth params")
	}
	return &metadata, nil
}

func (s azureDataExplorerScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	metricValue, err := azure.GetAzureDataExplorerMetricValue(ctx, s.client, s.metadata.DatabaseName, s.metadata.Query)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("failed to get metrics for scaled object %s in namespace %s: %v", s.name, s.namespace, err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(metricValue, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}
	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s azureDataExplorerScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: s.metadata.MetricName,
		},
		Target: GetMetricTarget(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

func (s azureDataExplorerScaler) IsActive(ctx context.Context) (bool, error) {
	metricValue, err := azure.GetAzureDataExplorerMetricValue(ctx, s.client, s.metadata.DatabaseName, s.metadata.Query)
	if err != nil {
		return false, fmt.Errorf("failed to get azure data explorer metric value: %s", err)
	}

	return metricValue > 0, nil
}

func (s azureDataExplorerScaler) Close(context.Context) error {
	return nil
}
