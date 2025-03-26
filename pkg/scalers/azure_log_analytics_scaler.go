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
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	tenantID            string
	clientID            string
	clientSecret        string
	workspaceID         string
	podIdentity         kedav1alpha1.AuthPodIdentity
	query               string
	threshold           float64
	activationThreshold float64
	triggerIndex        int
	cloud               azcloud.Configuration
	unsafeSsl           bool
	timeout             time.Duration // custom HTTP client timeout
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
		creds, err = azidentity.NewClientSecretCredential(meta.tenantID, meta.clientID, meta.clientSecret, nil)
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
			Transport: kedautil.CreateHTTPClient(meta.timeout, meta.unsafeSsl),
			Cloud:     meta.cloud,
		},
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func parseAzureLogAnalyticsMetadata(config *scalersconfig.ScalerConfig) (*azureLogAnalyticsMetadata, error) {
	meta := azureLogAnalyticsMetadata{}
	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// Getting tenantId
		tenantID, err := getParameterFromConfig(config, "tenantId", true)
		if err != nil {
			return nil, err
		}
		meta.tenantID = tenantID

		// Getting clientId
		clientID, err := getParameterFromConfig(config, "clientId", true)
		if err != nil {
			return nil, err
		}
		meta.clientID = clientID

		// Getting clientSecret
		clientSecret, err := getParameterFromConfig(config, "clientSecret", true)
		if err != nil {
			return nil, err
		}
		meta.clientSecret = clientSecret

		meta.podIdentity = config.PodIdentity
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		meta.podIdentity = config.PodIdentity
	default:
		return nil, fmt.Errorf("error parsing metadata. Details: Log Analytics Scaler doesn't support pod identity %s", config.PodIdentity.Provider)
	}

	// Getting workspaceId
	workspaceID, err := getParameterFromConfig(config, "workspaceId", true)
	if err != nil {
		return nil, err
	}
	meta.workspaceID = workspaceID

	// Getting query, observe that we dont check AuthParams for query
	query, err := getParameterFromConfig(config, "query", false)
	if err != nil {
		return nil, err
	}
	meta.query = query

	// Getting threshold, observe that we don't check AuthParams for threshold
	val, err := getParameterFromConfig(config, "threshold", false)
	if err != nil {
		if config.AsMetricSource {
			val = "0"
		} else {
			return nil, err
		}
	}
	threshold, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing metadata. Details: can't parse threshold. Inner Error: %w", err)
	}
	meta.threshold = threshold

	// Getting activationThreshold
	meta.activationThreshold = 0
	val, err = getParameterFromConfig(config, "activationThreshold", false)
	if err == nil {
		activationThreshold, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing metadata. Details: can't parse threshold. Inner Error: %w", err)
		}
		meta.activationThreshold = activationThreshold
	}
	meta.triggerIndex = config.TriggerIndex

	meta.cloud = azcloud.AzurePublic
	if cloud, ok := config.TriggerMetadata["cloud"]; ok {
		if strings.EqualFold(cloud, azure.PrivateCloud) {
			if resource, ok := config.TriggerMetadata["logAnalyticsResourceURL"]; ok && resource != "" {
				meta.cloud.Services[azquery.ServiceNameLogs] = azcloud.ServiceConfiguration{
					Endpoint: fmt.Sprintf("%s/v1", resource),
					Audience: resource,
				}
			} else {
				return nil, fmt.Errorf("logAnalyticsResourceURL must be provided for %s cloud type", azure.PrivateCloud)
			}
		} else if resource, ok := azure.AzureClouds[strings.ToUpper(cloud)]; ok {
			meta.cloud = resource
		} else {
			return nil, fmt.Errorf("there is no cloud environment matching the name %s", cloud)
		}
	}

	// Getting unsafeSsl, observe that we don't check AuthParams for unsafeSsl
	meta.unsafeSsl = false
	unsafeSslVal, err := getParameterFromConfig(config, "unsafeSsl", false)
	if err == nil {
		unsafeSsl, err := strconv.ParseBool(unsafeSslVal)
		if err != nil {
			return nil, fmt.Errorf("error parsing metadata. Details: can't parse unsafeSsl. Inner Error: %w", err)
		}
		meta.unsafeSsl = unsafeSsl
	}

	// Resolve HTTP client timeout
	meta.timeout = config.GlobalHTTPTimeout
	timeoutVal, err := getParameterFromConfig(config, "timeout", false)
	if err == nil {
		timeout, err := strconv.Atoi(timeoutVal)
		if err != nil {
			return nil, fmt.Errorf("unable to parse timeout: %w", err)
		}

		if timeout <= 0 {
			return nil, fmt.Errorf("timeout must be greater than 0: %w", err)
		}

		meta.timeout = time.Duration(timeout) * time.Millisecond
	}

	return &meta, nil
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
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("%s-%s", "azure-log-analytics", s.metadata.workspaceID))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.threshold),
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

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.activationThreshold, nil
}

func (s *azureLogAnalyticsScaler) Close(context.Context) error {
	return nil
}

func (s *azureLogAnalyticsScaler) getMetricData(ctx context.Context) (float64, error) {
	response, err := s.client.QueryWorkspace(ctx, s.metadata.workspaceID, azquery.Body{
		Query: &s.metadata.query,
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
