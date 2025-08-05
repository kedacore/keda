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
	"github.com/kedacore/keda/v2/keda-scalers/azure"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// monitorInfo to create metric request
type azureMonitorMetadata struct {
	triggerIndex                 int
	TargetValue                  float64 `keda:"name=targetValue, order=triggerMetadata"`
	ActivationTargetValue        float64 `keda:"name=activationTargetValue, order=triggerMetadata, default=0"`
	ResourceURI                  string  `keda:"name=resourceURI, order=triggerMetadata"`
	TenantID                     string  `keda:"name=tenantId, order=triggerMetadata"`
	SubscriptionID               string  `keda:"name=subscriptionId, order=triggerMetadata"`
	ResourceGroupName            string  `keda:"name=resourceGroupName, order=triggerMetadata"`
	Name                         string  `keda:"name=metricName, order=triggerMetadata"`
	Namespace                    string  `keda:"name=metricNamespace, order=triggerMetadata, optional"`
	NamespaceRef                 *string
	Filter                       string `keda:"name=metricFilter, order=triggerMetadata, optional"`
	FilterRef                    *string
	AggregationInterval          string                  `keda:"name=metricAggregationInterval, order=triggerMetadata, optional"`
	AggregationType              azquery.AggregationType `keda:"name=metricAggregationType, order=triggerMetadata"`
	ClientID                     string                  `keda:"name=activeDirectoryClientId, order=triggerMetadata;resolvedEnv;authParams, optional"`
	ClientPassword               string                  `keda:"name=activeDirectoryClientPassword, order=triggerMetadata;resolvedEnv;authParams, optional"`
	CloudName                    string                  `keda:"name=cloud, order=triggerMetadata, optional"`
	AzureResourceManagerEndpoint string                  `keda:"name=azureResourceManagerEndpoint, order=triggerMetadata, optional"`
	Cloud                        azcloud.Configuration
}

func (m *azureMonitorMetadata) Validate() error {
	if m.Namespace != "" {
		m.NamespaceRef = &m.Namespace
	}

	if m.Filter != "" {
		m.FilterRef = &m.Filter
	}

	resourceURI := strings.Split(m.ResourceURI, "/")
	if len(resourceURI) != 3 {
		return fmt.Errorf("resourceURI not in the correct format. Should be namespace/resource_type/resource_name")
	}

	if m.AggregationType != "" {
		allowedTypes := azquery.PossibleAggregationTypeValues()
		if !slices.Contains(allowedTypes, m.AggregationType) {
			return fmt.Errorf("invalid metricAggregationType given")
		}
	} else {
		return fmt.Errorf("no metricAggregationType given")
	}

	if m.AggregationInterval != "" {
		aggregationInterval := strings.Split(m.AggregationInterval, ":")
		if len(aggregationInterval) != 3 {
			return fmt.Errorf("metricAggregationInterval not in the correct format. Should be hh:mm:ss")
		}
	}

	cloud, err := m.parseCloud(m.CloudName, m.AzureResourceManagerEndpoint)
	if err != nil {
		return err
	}
	m.Cloud = cloud

	return nil
}

func (m *azureMonitorMetadata) MetricResourceURI() string {
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

// checkAzurePodIdentityParams gets the activeDirectory clientID and password
func (m *azureMonitorMetadata) checkAzurePodIdentityParams(config *scalersconfig.ScalerConfig) error {
	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		if m.ClientID == "" {
			return fmt.Errorf("no activeDirectoryClientId given")
		}
		if len(m.ClientPassword) == 0 {
			return fmt.Errorf("no activeDirectoryClientPassword given")
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		// no params required to be parsed
	default:
		return fmt.Errorf("azure Monitor doesn't support pod identity %s", config.PodIdentity.Provider)
	}

	return nil
}

func (m azureMonitorMetadata) parseCloud(cloudName string, resourceManagerEndpoint string) (azcloud.Configuration, error) {
	foundCloud := azcloud.AzurePublic
	if cloudName != "" {
		if strings.EqualFold(cloudName, azure.PrivateCloud) {
			if resourceManagerEndpoint != "" {
				foundCloud.Services[azquery.ServiceNameLogs] = azcloud.ServiceConfiguration{
					Endpoint: resourceManagerEndpoint,
					Audience: resourceManagerEndpoint,
				}
			} else {
				return azcloud.Configuration{}, fmt.Errorf("logAnalyticsResourceURL must be provided for %s cloud type", azure.PrivateCloud)
			}
		} else if resource, ok := azure.AzureClouds[strings.ToUpper(cloudName)]; ok {
			foundCloud = resource
		} else {
			return azcloud.Configuration{}, fmt.Errorf("there is no cloud environment matching the name %s", cloudName)
		}
	}
	return foundCloud, nil
}

type azureMonitorScaler struct {
	metricType v2.MetricTargetType
	metadata   *azureMonitorMetadata
	logger     logr.Logger
	client     *azquery.MetricsClient
}

// NewAzureMonitorScaler creates a new AzureMonitorScaler
func NewAzureMonitorScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_monitor_scaler")

	meta, err := parseAzureMonitorMetadata(config)
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
		creds, err = azidentity.NewClientSecretCredential(meta.TenantID, meta.ClientID, meta.ClientPassword, nil)
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
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
			Cloud:     meta.Cloud,
		},
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func parseAzureMonitorMetadata(config *scalersconfig.ScalerConfig) (*azureMonitorMetadata, error) {
	meta := &azureMonitorMetadata{}
	meta.triggerIndex = config.TriggerIndex
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing azure monitor metadata: %w", err)
	}

	if !config.AsMetricSource && meta.TargetValue == 0 {
		return nil, fmt.Errorf("no targetValue given")
	}

	err := meta.checkAzurePodIdentityParams(config)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func (s *azureMonitorScaler) Close(context.Context) error {
	return nil
}

func (s *azureMonitorScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-monitor-%s", s.metadata.Name))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
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

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationTargetValue, nil
}
func (s *azureMonitorScaler) requestMetric(ctx context.Context) (float64, error) {
	timespan, err := formatTimeSpan(s.metadata.AggregationInterval)
	if err != nil {
		return -1, err
	}
	opts := &azquery.MetricsClientQueryResourceOptions{
		MetricNames:     &s.metadata.Name,
		MetricNamespace: s.metadata.NamespaceRef,
		Filter:          s.metadata.FilterRef,
		Interval:        nil,
		Top:             nil,
		ResultType:      nil,
		OrderBy:         nil,
	}

	opts.Timespan = timespan
	opts.Aggregation = append(opts.Aggregation, &s.metadata.AggregationType)
	response, err := s.client.QueryResource(ctx, s.metadata.MetricResourceURI(), opts)
	if err != nil || len(response.Value) != 1 {
		s.logger.Error(err, "error getting azure monitor metric")
		return -1, err
	}

	if len(response.Value) == 0 {
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

	val, err := verifyAggregationTypeIsSupported(s.metadata.AggregationType, dataPtr)
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
