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
	"slices"
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

const (
	azureMonitorMetricName    = "metricName"
	targetValueName           = "targetValue"
	activationTargetValueName = "activationTargetValue"
)

// monitorInfo to create metric request
type monitorInfo struct {
	ResourceURI         string
	TenantID            string
	SubscriptionID      string
	ResourceGroupName   string
	Name                *string
	Namespace           *string
	Filter              *string
	AggregationInterval string
	AggregationType     *azquery.AggregationType
	ClientID            string
	ClientPassword      string
	Cloud               azcloud.Configuration
}

func (m monitorInfo) MetricResourceURI() string {
	resourceInfo := strings.Split(m.ResourceURI, "/")
	resourceProviderNamespace := resourceInfo[0]
	resourceType := resourceInfo[1]
	resourceName := resourceInfo[2]
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s",
		m.SubscriptionID,
		m.ResourceGroupName,
		resourceProviderNamespace,
		resourceType,
		resourceName)
}

type azureMonitorScaler struct {
	metricType v2.MetricTargetType
	metadata   *azureMonitorMetadata
	logger     logr.Logger
	client     *azquery.MetricsClient
}

type azureMonitorMetadata struct {
	azureMonitorInfo      monitorInfo
	targetValue           float64
	activationTargetValue float64
	triggerIndex          int
}

// NewAzureMonitorScaler creates a new AzureMonitorScaler
func NewAzureMonitorScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_monitor_scaler")

	meta, err := parseAzureMonitorMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure monitor metadata: %w", err)
	}

	client, err := CreateAzureMetricsClient(config, meta, logger)
	if err != nil {
		return nil, err
	}
	return &azureMonitorScaler{
		metricType: metricType,
		metadata:   meta,
		logger:     logger,
		client:     client,
	}, nil
}

func CreateAzureMetricsClient(config *scalersconfig.ScalerConfig, meta *azureMonitorMetadata, logger logr.Logger) (*azquery.MetricsClient, error) {
	var creds azcore.TokenCredential
	var err error
	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		creds, err = azidentity.NewClientSecretCredential(meta.azureMonitorInfo.TenantID, meta.azureMonitorInfo.ClientID, meta.azureMonitorInfo.ClientPassword, nil)
	case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
		creds, err = azure.NewChainedCredential(logger, config.PodIdentity)
	default:
		return nil, fmt.Errorf("azure monitor does not support pod identity provider - %s", config.PodIdentity.Provider)
	}
	if err != nil {
		return nil, err
	}
	client, err := azquery.NewMetricsClient(creds, &azquery.MetricsClientOptions{
		ClientOptions: policy.ClientOptions{
			Transport: kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false),
			Cloud:     meta.azureMonitorInfo.Cloud,
		},
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func parseAzureMonitorMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*azureMonitorMetadata, error) {
	meta := azureMonitorMetadata{
		azureMonitorInfo: monitorInfo{},
	}

	if val, ok := config.TriggerMetadata[targetValueName]; ok && val != "" {
		targetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			logger.Error(err, "Error parsing azure monitor metadata", "targetValue", targetValueName)
			return nil, fmt.Errorf("error parsing azure monitor metadata %s: %w", targetValueName, err)
		}
		meta.targetValue = targetValue
	} else {
		if config.AsMetricSource {
			meta.targetValue = 0
		} else {
			return nil, fmt.Errorf("no targetValue given")
		}
	}

	if val, ok := config.TriggerMetadata[activationTargetValueName]; ok && val != "" {
		activationTargetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			logger.Error(err, "Error parsing azure monitor metadata", "targetValue", activationTargetValueName)
			return nil, fmt.Errorf("error parsing azure monitor metadata %s: %w", activationTargetValueName, err)
		}
		meta.activationTargetValue = activationTargetValue
	} else {
		meta.activationTargetValue = 0
	}

	if val, ok := config.TriggerMetadata["resourceURI"]; ok && val != "" {
		resourceURI := strings.Split(val, "/")
		if len(resourceURI) != 3 {
			return nil, fmt.Errorf("resourceURI not in the correct format. Should be namespace/resource_type/resource_name")
		}
		meta.azureMonitorInfo.ResourceURI = val
	} else {
		return nil, fmt.Errorf("no resourceURI given")
	}

	if val, ok := config.TriggerMetadata["resourceGroupName"]; ok && val != "" {
		meta.azureMonitorInfo.ResourceGroupName = val
	} else {
		return nil, fmt.Errorf("no resourceGroupName given")
	}

	if val, ok := config.TriggerMetadata[azureMonitorMetricName]; ok && val != "" {
		meta.azureMonitorInfo.Name = &val
	} else {
		return nil, fmt.Errorf("no metricName given")
	}

	if val, ok := config.TriggerMetadata["metricAggregationType"]; ok && val != "" {
		aggregationType := azquery.AggregationType(val)
		allowedTypes := azquery.PossibleAggregationTypeValues()
		if !slices.Contains(allowedTypes, aggregationType) {
			return nil, fmt.Errorf("invalid metricAggregationType given")
		}
		meta.azureMonitorInfo.AggregationType = &aggregationType
	} else {
		return nil, fmt.Errorf("no metricAggregationType given")
	}

	if val, ok := config.TriggerMetadata["metricFilter"]; ok && val != "" {
		meta.azureMonitorInfo.Filter = &val
	}

	if val, ok := config.TriggerMetadata["metricAggregationInterval"]; ok && val != "" {
		aggregationInterval := strings.Split(val, ":")
		if len(aggregationInterval) != 3 {
			return nil, fmt.Errorf("metricAggregationInterval not in the correct format. Should be hh:mm:ss")
		}
		meta.azureMonitorInfo.AggregationInterval = val
	}

	// Required authentication parameters below

	if val, ok := config.TriggerMetadata["subscriptionId"]; ok && val != "" {
		meta.azureMonitorInfo.SubscriptionID = val
	} else {
		return nil, fmt.Errorf("no subscriptionId given")
	}

	if val, ok := config.TriggerMetadata["tenantId"]; ok && val != "" {
		meta.azureMonitorInfo.TenantID = val
	} else {
		return nil, fmt.Errorf("no tenantId given")
	}

	if val, ok := config.TriggerMetadata["metricNamespace"]; ok {
		meta.azureMonitorInfo.Namespace = &val
	}

	clientID, clientPassword, err := parseAzurePodIdentityParams(config)
	if err != nil {
		return nil, err
	}
	meta.azureMonitorInfo.ClientID = clientID
	meta.azureMonitorInfo.ClientPassword = clientPassword

	cloud, err := parseCloud(config.TriggerMetadata)
	if err != nil {
		return nil, err
	}
	meta.azureMonitorInfo.Cloud = cloud

	meta.triggerIndex = config.TriggerIndex
	return &meta, nil
}

func parseCloud(metadata map[string]string) (azcloud.Configuration, error) {
	foundCloud := azcloud.AzurePublic
	if cloud, ok := metadata["cloud"]; ok {
		if strings.EqualFold(cloud, azure.PrivateCloud) {
			if resource, ok := metadata["azureResourceManagerEndpoint"]; ok && resource != "" {
				foundCloud.Services[azquery.ServiceNameLogs] = azcloud.ServiceConfiguration{
					Endpoint: resource,
					Audience: resource,
				}
			} else {
				return azcloud.Configuration{}, fmt.Errorf("logAnalyticsResourceURL must be provided for %s cloud type", azure.PrivateCloud)
			}
		} else if resource, ok := azure.AzureClouds[strings.ToUpper(cloud)]; ok {
			foundCloud = resource
		} else {
			return azcloud.Configuration{}, fmt.Errorf("there is no cloud environment matching the name %s", cloud)
		}
	}
	return foundCloud, nil
}

// parseAzurePodIdentityParams gets the activeDirectory clientID and password
func parseAzurePodIdentityParams(config *scalersconfig.ScalerConfig) (clientID string, clientPassword string, err error) {
	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		clientID, err = getParameterFromConfig(config, "activeDirectoryClientId", true)
		if err != nil || clientID == "" {
			return "", "", fmt.Errorf("no activeDirectoryClientId given")
		}

		if config.AuthParams["activeDirectoryClientPassword"] != "" {
			clientPassword = config.AuthParams["activeDirectoryClientPassword"]
		} else if config.TriggerMetadata["activeDirectoryClientPasswordFromEnv"] != "" {
			clientPassword = config.ResolvedEnv[config.TriggerMetadata["activeDirectoryClientPasswordFromEnv"]]
		}

		if len(clientPassword) == 0 {
			return "", "", fmt.Errorf("no activeDirectoryClientPassword given")
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		// no params required to be parsed
	default:
		return "", "", fmt.Errorf("azure Monitor doesn't support pod identity %s", config.PodIdentity.Provider)
	}

	return clientID, clientPassword, nil
}

func (s *azureMonitorScaler) Close(context.Context) error {
	return nil
}

func (s *azureMonitorScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-monitor-%s", *s.metadata.azureMonitorInfo.Name))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureMonitorScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.requestMetric(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.activationTargetValue, nil
}
func (s *azureMonitorScaler) requestMetric(ctx context.Context) (float64, error) {
	timespan, err := formatTimeSpan(s.metadata.azureMonitorInfo.AggregationInterval)
	if err != nil {
		return -1, err
	}
	opts := &azquery.MetricsClientQueryResourceOptions{
		MetricNames:     s.metadata.azureMonitorInfo.Name,
		MetricNamespace: s.metadata.azureMonitorInfo.Namespace,
		Filter:          s.metadata.azureMonitorInfo.Filter,
		Interval:        nil,
		Top:             nil,
		ResultType:      nil,
		OrderBy:         nil,
	}

	opts.Timespan = timespan
	opts.Aggregation = append(opts.Aggregation, s.metadata.azureMonitorInfo.AggregationType)
	response, err := s.client.QueryResource(ctx, s.metadata.azureMonitorInfo.MetricResourceURI(), opts)
	if err != nil || len(response.Value) != 1 {
		s.logger.Error(err, "error getting azure monitor metric")
		return -1, err
	}

	if response.Value == nil || len(response.Value) == 0 {
		err := fmt.Errorf("got an empty response for metric %s/%s and aggregate type %s", "azMetricRequest.ResourceProviderNamespace", "azMetricRequest.MetricName", "azMetricRequest.Aggregation")
		return -1, err
	}

	timeseriesPtr := response.Value[0].TimeSeries
	if len(timeseriesPtr) == 0 {
		err := fmt.Errorf("got metric result for %s/%s and aggregate type %s without timeseries", "azMetricRequest.ResourceProviderNamespace", "azMetricRequest.MetricName", "azMetricRequest.Aggregation")
		return -1, err
	}

	dataPtr := response.Value[0].TimeSeries[0].Data
	if len(dataPtr) == 0 {
		err := fmt.Errorf("got metric result for %s/%s and aggregate type %s without any metric values", "azMetricRequest.ResourceProviderNamespace", "azMetricRequest.MetricName", "azMetricRequest.Aggregation")
		return -1, err
	}

	val, err := verifyAggregationTypeIsSupported(*s.metadata.azureMonitorInfo.AggregationType, dataPtr)
	if err != nil {
		return -1, err
	}
	return val, nil
}

// formatTimeSpan defaults to a 5 minute timespan if the user does not provide one
func formatTimeSpan(timeSpan string) (*azquery.TimeInterval, error) {
	endtime := time.Now().UTC()
	starttime := time.Now().Add(-(5 * time.Minute)).UTC()
	if timeSpan != "" {
		aggregationInterval := strings.Split(timeSpan, ":")
		hours, herr := strconv.Atoi(aggregationInterval[0])
		minutes, merr := strconv.Atoi(aggregationInterval[1])
		seconds, serr := strconv.Atoi(aggregationInterval[2])

		if herr != nil || merr != nil || serr != nil {
			return nil, fmt.Errorf("errors parsing metricAggregationInterval: %v, %v, %w", herr, merr, serr)
		}

		starttime = time.Now().Add(-(time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second)).UTC()
	}
	interval := azquery.NewTimeInterval(starttime, endtime)
	return &interval, nil
}

func verifyAggregationTypeIsSupported(aggregationType azquery.AggregationType, data []*azquery.MetricValue) (float64, error) {
	if data == nil {
		err := fmt.Errorf("invalid response")
		return -1, err
	}
	var valuePtr *float64
	switch {
	case strings.EqualFold(string(azquery.AggregationTypeAverage), string(aggregationType)) && data[len(data)-1].Average != nil:
		valuePtr = data[len(data)-1].Average
	case strings.EqualFold(string(azquery.AggregationTypeTotal), string(aggregationType)) && data[len(data)-1].Total != nil:
		valuePtr = data[len(data)-1].Total
	case strings.EqualFold(string(azquery.AggregationTypeMaximum), string(aggregationType)) && data[len(data)-1].Maximum != nil:
		valuePtr = data[len(data)-1].Maximum
	case strings.EqualFold(string(azquery.AggregationTypeMinimum), string(aggregationType)) && data[len(data)-1].Minimum != nil:
		valuePtr = data[len(data)-1].Minimum
	case strings.EqualFold(string(azquery.AggregationTypeCount), string(aggregationType)) && data[len(data)-1].Count != nil:
		valuePtr = data[len(data)-1].Count
	default:
		err := fmt.Errorf("unsupported aggregation type %s", aggregationType)
		return -1, err
	}
	return *valuePtr, nil
}
