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
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/keda-scalers/azure"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type azureDataExplorerScaler struct {
	metricType v2.MetricTargetType
	metadata   *azure.DataExplorerMetadata
	client     *kusto.Client
	name       string
	namespace  string
	logger     logr.Logger
}

const adxName = "azure-data-explorer"

func NewAzureDataExplorerScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_data_explorer_scaler")

	metadata, err := parseAzureDataExplorerMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to parse azure data explorer metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)
	client, err := azure.CreateAzureDataExplorerClient(metadata, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure data explorer client: %w", err)
	}

	return &azureDataExplorerScaler{
		metricType: metricType,
		metadata:   metadata,
		client:     client,
		name:       config.ScalableObjectName,
		namespace:  config.ScalableObjectNamespace,
		logger:     logger,
	}, nil
}

func parseAzureDataExplorerMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*azure.DataExplorerMetadata, error) {
	metadata, err := parseAzureDataExplorerAuthParams(config, logger)
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
		threshold, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing metadata. Details: can't parse threshold. Inner Error: %w", err)
		}
		metadata.Threshold = threshold
	}

	// Get activationThreshold.
	metadata.ActivationThreshold = 0
	if val, ok := config.TriggerMetadata["activationThreshold"]; ok {
		activationThreshold, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing metadata. Details: can't parse activationThreshold. Inner Error: %w", err)
		}
		metadata.ActivationThreshold = activationThreshold
	}

	// Generate metricName.
	metadata.MetricName = GenerateMetricNameWithIndex(config.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("%s-%s", adxName, metadata.DatabaseName)))

	activeDirectoryEndpoint, err := azure.ParseActiveDirectoryEndpoint(config.TriggerMetadata)
	if err != nil {
		return nil, err
	}
	metadata.ActiveDirectoryEndpoint = activeDirectoryEndpoint

	logger.V(1).Info("Parsed azureDataExplorerMetadata",
		"database", metadata.DatabaseName,
		"endpoint", metadata.Endpoint,
		"metricName", metadata.MetricName,
		"query", metadata.Query,
		"threshold", metadata.Threshold,
		"activeDirectoryEndpoint", metadata.ActiveDirectoryEndpoint,
	)

	return metadata, nil
}

func parseAzureDataExplorerAuthParams(config *scalersconfig.ScalerConfig, logger logr.Logger) (*azure.DataExplorerMetadata, error) {
	metadata := azure.DataExplorerMetadata{}

	switch config.PodIdentity.Provider {
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		metadata.PodIdentity = config.PodIdentity
	case "", kedav1alpha1.PodIdentityProviderNone:
		logger.V(1).Info("Pod Identity is not provided. Trying to resolve clientId, clientSecret and tenantId.")

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

		var clientSecret string
		if val, ok := config.AuthParams["clientSecret"]; ok && val != "" {
			clientSecret = val
		} else if val, ok = config.TriggerMetadata["clientSecretFromEnv"]; ok && val != "" {
			clientSecret = val
		} else {
			return nil, fmt.Errorf("error parsing metadata. Details: clientSecret was not found in metadata. Check your ScaledObject configuration")
		}
		metadata.ClientSecret = clientSecret

	default:
		return nil, fmt.Errorf("error parsing auth params")
	}
	return &metadata, nil
}

func (s azureDataExplorerScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	metricValue, err := azure.GetAzureDataExplorerMetricValue(ctx, s.client, s.metadata.DatabaseName, s.metadata.Query)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to get metrics for scaled object %s in namespace %s: %w", s.name, s.namespace, err)
	}

	metric := GenerateMetricInMili(metricName, metricValue)

	return []external_metrics.ExternalMetricValue{metric}, metricValue > s.metadata.ActivationThreshold, nil
}

func (s azureDataExplorerScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.MetricName,
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s azureDataExplorerScaler) Close(context.Context) error {
	if s.client != nil && s.client.HttpClient() != nil {
		s.client.HttpClient().CloseIdleConnections()
	}
	return nil
}
