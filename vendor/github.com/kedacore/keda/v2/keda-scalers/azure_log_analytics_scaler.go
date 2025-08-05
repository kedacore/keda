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
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azcloud "github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/keda-scalers/azure"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type azureLogAnalyticsScaler struct {
	metricType v2.MetricTargetType
	metadata   *azureLogAnalyticsMetadata
	name       string
	namespace  string
	client     *azquery.LogsClient
	logger     logr.Logger
}

type azureLogAnalyticsMetadata struct {
	TenantID                string `keda:"name=tenantId, order=authParams;triggerMetadata;resolvedEnv, optional"`
	ClientID                string `keda:"name=clientId, order=authParams;triggerMetadata;resolvedEnv, optional"`
	ClientSecret            string `keda:"name=clientSecret, order=authParams;triggerMetadata;resolvedEnv, optional"`
	WorkspaceID             string `keda:"name=workspaceId, order=authParams;triggerMetadata;resolvedEnv"`
	PodIdentity             kedav1alpha1.AuthPodIdentity
	Query                   string  `keda:"name=query, order=triggerMetadata"`
	Threshold               float64 `keda:"name=threshold, order=triggerMetadata"`
	ActivationThreshold     float64 `keda:"name=activationThreshold, order=triggerMetadata, default=0"`
	LogAnalyticsResourceURL string  `keda:"name=logAnalyticsResourceURL, order=triggerMetadata, optional"`
	TriggerIndex            int
	CloudName               string `keda:"name=cloud, order=triggerMetadata, default=azurePublicCloud"`
	Cloud                   azcloud.Configuration
	UnsafeSsl               bool          `keda:"name=unsafeSsl, order=triggerMetadata, default=false"`
	Timeout                 time.Duration `keda:"name=timeout, order=triggerMetadata, optional"`
}

func (m *azureLogAnalyticsMetadata) Validate() error {
	missingParameter := ""

	switch m.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		if m.TenantID == "" {
			missingParameter = "tenantId"
		}
		if m.ClientID == "" {
			missingParameter = "clientId"
		}
		if m.ClientSecret == "" {
			missingParameter = "clientSecret"
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		break
	default:
		return fmt.Errorf("error parsing metadata. Details: Log Analytics Scaler doesn't support pod identity %s", m.PodIdentity.Provider)
	}

	m.Cloud = azcloud.AzurePublic
	if strings.EqualFold(m.CloudName, azure.PrivateCloud) {
		if m.LogAnalyticsResourceURL != "" {
			m.Cloud.Services[azquery.ServiceNameLogs] = azcloud.ServiceConfiguration{
				Endpoint: fmt.Sprintf("%s/v1", m.LogAnalyticsResourceURL),
				Audience: m.LogAnalyticsResourceURL,
			}
		} else {
			return fmt.Errorf("logAnalyticsResourceURL must be provided for %s cloud type", azure.PrivateCloud)
		}
	} else if resource, ok := azure.AzureClouds[strings.ToUpper(m.CloudName)]; ok {
		m.Cloud = resource
	} else {
		return fmt.Errorf("there is no cloud environment matching the name %s", m.CloudName)
	}

	if m.Timeout > 0 {
		m.Timeout *= time.Millisecond
	}

	if missingParameter != "" {
		return fmt.Errorf("error parsing metadata. Details: %s was not found in metadata. Check your ScaledObject configuration", missingParameter)
	}

	return nil
}

// NewAzureLogAnalyticsScaler creates a new Azure Log Analytics Scaler
func NewAzureLogAnalyticsScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	azureLogAnalyticsMetadata, err := parseAzureLogAnalyticsMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Log Analytics scaler. Scaled object: %s. Namespace: %s. Inner Error: %w", config.ScalableObjectName, config.ScalableObjectNamespace, err)
	}

	logger := InitializeLogger(config, "azure_log_analytics_scaler")

	client, err := CreateAzureLogsClient(config, azureLogAnalyticsMetadata, logger)
	if err != nil {
		return nil, err
	}

	return &azureLogAnalyticsScaler{
		metricType: metricType,
		metadata:   azureLogAnalyticsMetadata,
		name:       config.ScalableObjectName,
		namespace:  config.ScalableObjectNamespace,
		client:     client,
		logger:     logger,
	}, nil
}

func CreateAzureLogsClient(config *scalersconfig.ScalerConfig, meta *azureLogAnalyticsMetadata, logger logr.Logger) (*azquery.LogsClient, error) {
	var creds azcore.TokenCredential
	var err error
	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		creds, err = azidentity.NewClientSecretCredential(meta.TenantID, meta.ClientID, meta.ClientSecret, nil)
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		creds, err = azure.NewChainedCredential(logger, config.PodIdentity)
	default:
		return nil, fmt.Errorf("azure monitor does not support pod identity provider - %s", config.PodIdentity.Provider)
	}
	if err != nil {
		return nil, err
	}
	client, err := azquery.NewLogsClient(creds, &azquery.LogsClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: kedautil.CreateHTTPClient(meta.Timeout, meta.UnsafeSsl),
			Cloud:     meta.Cloud,
		},
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func parseAzureLogAnalyticsMetadata(config *scalersconfig.ScalerConfig) (*azureLogAnalyticsMetadata, error) {
	meta := &azureLogAnalyticsMetadata{}
	meta.TriggerIndex = config.TriggerIndex
	meta.Timeout = config.GlobalHTTPTimeout
	meta.PodIdentity = config.PodIdentity
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing azure loganalytics metadata: %w", err)
	}

	return meta, nil
}

// getParameterFromConfig gets the parameter from the configs, if checkAuthParams is true
// then AuthParams is also check for the parameter
func getParameterFromConfig(config *scalersconfig.ScalerConfig, parameter string, checkAuthParams bool) (string, error) {
	if val, ok := config.AuthParams[parameter]; checkAuthParams && ok && val != "" {
		return val, nil
	} else if val, ok := config.TriggerMetadata[parameter]; ok && val != "" {
		return val, nil
	} else if val, ok := config.TriggerMetadata[fmt.Sprintf("%sFromEnv", parameter)]; ok && val != "" {
		return config.ResolvedEnv[config.TriggerMetadata[fmt.Sprintf("%sFromEnv", parameter)]], nil
	}
	return "", fmt.Errorf("error parsing metadata. Details: %s was not found in metadata. Check your ScaledObject configuration", parameter)
}

func (s *azureLogAnalyticsScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("%s-%s", "azure-log-analytics", s.metadata.WorkspaceID))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureLogAnalyticsScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.getMetricData(ctx)

	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("failed to get metrics. Scaled object: %s. Namespace: %s. Inner Error: %w", s.name, s.namespace, err)
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationThreshold, nil
}

func (s *azureLogAnalyticsScaler) Close(context.Context) error {
	return nil
}

func (s *azureLogAnalyticsScaler) getMetricData(ctx context.Context) (float64, error) {
	response, err := s.client.QueryWorkspace(ctx, s.metadata.WorkspaceID, azquery.Body{
		Query: &s.metadata.Query,
	}, nil)
	if err != nil {
		return -1, err
	}

	// Pre-validation of query result:
	switch {
	case len(response.Tables) == 0 || len(response.Tables[0].Columns) == 0 || len(response.Tables[0].Rows) == 0:
		return -1, fmt.Errorf("error validating Log Analytics request. Details: there is no results after running your query")
	case len(response.Tables) > 1:
		return -1, fmt.Errorf("error validating Log Analytics request. Details: too many tables in query result: %d, expected: 1", len(response.Tables))
	case len(response.Tables[0].Rows) > 1:
		return -1, fmt.Errorf("error validating Log Analytics request. Details: too many rows in query result: %d, expected: 1", len(response.Tables[0].Rows))
	}

	if len(response.Tables[0].Rows[0]) > 0 {
		metricDataType := response.Tables[0].Columns[0].Type
		metricVal := response.Tables[0].Rows[0][0]
		if metricDataType == nil || metricVal == nil {
			return -1, fmt.Errorf("error parsing the response %w", err)
		}
		parsedMetricVal, err := parseTableValueToFloat64(metricVal, *metricDataType)
		if err != nil {
			return -1, fmt.Errorf("error parsing the response %w", err)
		}
		return parsedMetricVal, nil
	}

	return -1, fmt.Errorf("error parsing the response %w", err)
}

func parseTableValueToFloat64(value interface{}, dataType azquery.LogsColumnType) (float64, error) {
	if value != nil {
		// type can be: real, int, long
		if dataType == azquery.LogsColumnTypeReal || dataType == azquery.LogsColumnTypeInt || dataType == azquery.LogsColumnTypeLong {
			convertedValue, isConverted := value.(float64)
			if !isConverted {
				return 0, fmt.Errorf("error validating Log Analytics request. Details: cannot convert result to type float64")
			}
			if convertedValue < 0 {
				return 0, fmt.Errorf("error validating Log Analytics request. Details: value should be >=0, but received %f", value)
			}
			return convertedValue, nil
		}
		return 0, fmt.Errorf("error validating Log Analytics request. Details: value data type should be real, int or long, but received %s", dataType)
	}
	return 0, fmt.Errorf("error validating Log Analytics request. Details: value is empty, check your query")
}
