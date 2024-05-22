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

package azure

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// Much of the code in this file is taken from the Azure Kubernetes Metrics Adapter
// https://github.com/Azure/azure-k8s-metrics-adapter/tree/master/pkg/azure/externalmetrics

type azureExternalMetricRequest struct {
	MetricName                string
	MetricNamespace           string
	SubscriptionID            string
	ResourceName              string
	ResourceProviderNamespace string
	ResourceType              string
	Aggregation               string
	Timespan                  string
	Filter                    string
	ResourceGroup             string
}

// MonitorInfo to create metric request
type MonitorInfo struct {
	ResourceURI                  string
	TenantID                     string
	SubscriptionID               string
	ResourceGroupName            string
	Name                         string
	Namespace                    string
	Filter                       string
	AggregationInterval          string
	AggregationType              string
	ClientID                     string
	ClientPassword               string
	AzureResourceManagerEndpoint string
	ActiveDirectoryEndpoint      string
}

var azureMonitorLog = logf.Log.WithName("azure_monitor_scaler")

// GetAzureMetricValue returns the value of an Azure Monitor metric, rounded to the nearest int
func GetAzureMetricValue(ctx context.Context, info MonitorInfo, podIdentity kedav1alpha1.AuthPodIdentity) (float64, error) {
	client := createMetricsClient(ctx, info, podIdentity)
	requestPtr, err := createMetricsRequest(info)
	if err != nil {
		return -1, err
	}

	return executeRequest(ctx, client, requestPtr)
}

func createMetricsClient(ctx context.Context, info MonitorInfo, podIdentity kedav1alpha1.AuthPodIdentity) insights.MetricsClient {
	client := insights.NewMetricsClientWithBaseURI(info.AzureResourceManagerEndpoint, info.SubscriptionID)
	var authConfig auth.AuthorizerConfig
	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		config := auth.NewClientCredentialsConfig(info.ClientID, info.ClientPassword, info.TenantID)
		config.Resource = info.AzureResourceManagerEndpoint
		config.AADEndpoint = info.ActiveDirectoryEndpoint

		authConfig = config
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		authConfig = NewAzureADWorkloadIdentityConfig(ctx, podIdentity.GetIdentityID(), podIdentity.GetIdentityTenantID(), podIdentity.GetIdentityAuthorityHost(), info.AzureResourceManagerEndpoint)
	}

	authorizer, _ := authConfig.Authorizer()
	client.Authorizer = authorizer

	return client
}

func createMetricsRequest(info MonitorInfo) (*azureExternalMetricRequest, error) {
	metricRequest := azureExternalMetricRequest{
		MetricName:      info.Name,
		MetricNamespace: info.Namespace,
		SubscriptionID:  info.SubscriptionID,
		Aggregation:     info.AggregationType,
		Filter:          info.Filter,
		ResourceGroup:   info.ResourceGroupName,
	}

	resourceInfo := strings.Split(info.ResourceURI, "/")
	metricRequest.ResourceProviderNamespace = resourceInfo[0]
	metricRequest.ResourceType = resourceInfo[1]
	metricRequest.ResourceName = resourceInfo[2]

	// if no timespan is provided, defaults to 5 minutes
	timespan, err := formatTimeSpan(info.AggregationInterval)
	if err != nil {
		return nil, err
	}

	metricRequest.Timespan = timespan

	return &metricRequest, nil
}

func executeRequest(ctx context.Context, client insights.MetricsClient, request *azureExternalMetricRequest) (float64, error) {
	metricResponse, err := getAzureMetric(ctx, client, *request)
	if err != nil {
		return -1, fmt.Errorf("error getting azure monitor metric %s: %w", request.MetricName, err)
	}

	return metricResponse, nil
}

func getAzureMetric(ctx context.Context, client insights.MetricsClient, azMetricRequest azureExternalMetricRequest) (float64, error) {
	err := azMetricRequest.validate()
	if err != nil {
		return -1, err
	}

	metricResourceURI := azMetricRequest.metricResourceURI()
	azureMonitorLog.V(2).Info("metric request", "resource uri", metricResourceURI)

	metricResult, err := client.List(ctx, metricResourceURI,
		azMetricRequest.Timespan, nil,
		azMetricRequest.MetricName, azMetricRequest.Aggregation, nil,
		"", azMetricRequest.Filter, "", azMetricRequest.MetricNamespace)
	if err != nil {
		return -1, err
	}

	value, err := extractValue(azMetricRequest, metricResult)

	return value, err
}

func extractValue(azMetricRequest azureExternalMetricRequest, metricResult insights.Response) (float64, error) {
	metricVals := *metricResult.Value

	if len(metricVals) == 0 {
		err := fmt.Errorf("got an empty response for metric %s/%s and aggregate type %s", azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName, insights.AggregationType(strings.ToTitle(azMetricRequest.Aggregation)))
		return -1, err
	}

	timeseriesPtr := metricVals[0].Timeseries
	if timeseriesPtr == nil || len(*timeseriesPtr) == 0 {
		err := fmt.Errorf("got metric result for %s/%s and aggregate type %s without timeseries", azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName, insights.AggregationType(strings.ToTitle(azMetricRequest.Aggregation)))
		return -1, err
	}

	dataPtr := (*timeseriesPtr)[0].Data
	if dataPtr == nil || len(*dataPtr) == 0 {
		err := fmt.Errorf("got metric result for %s/%s and aggregate type %s without any metric values", azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName, insights.AggregationType(strings.ToTitle(azMetricRequest.Aggregation)))
		return -1, err
	}

	valuePtr, err := verifyAggregationTypeIsSupported(azMetricRequest.Aggregation, *dataPtr)
	if err != nil {
		return -1, fmt.Errorf("unable to get value for metric %s/%s with aggregation %s. No value returned by Azure Monitor", azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName, azMetricRequest.Aggregation)
	}

	azureMonitorLog.V(2).Info("value extracted from metric request", "metric type", azMetricRequest.Aggregation, "metric value", *valuePtr)

	return *valuePtr, nil
}

func (amr azureExternalMetricRequest) validate() error {
	if amr.MetricName == "" {
		return fmt.Errorf("metricName is required")
	}
	if amr.ResourceGroup == "" {
		return fmt.Errorf("resourceGroup is required")
	}
	if amr.SubscriptionID == "" {
		return fmt.Errorf("subscriptionID is required. set a default or pass via label selectors")
	}
	return nil
}

func (amr azureExternalMetricRequest) metricResourceURI() string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s",
		amr.SubscriptionID,
		amr.ResourceGroup,
		amr.ResourceProviderNamespace,
		amr.ResourceType,
		amr.ResourceName)
}

// formatTimeSpan defaults to a 5 minute timespan if the user does not provide one
func formatTimeSpan(timeSpan string) (string, error) {
	endtime := time.Now().UTC().Format(time.RFC3339)
	starttime := time.Now().Add(-(5 * time.Minute)).UTC().Format(time.RFC3339)
	if timeSpan != "" {
		aggregationInterval := strings.Split(timeSpan, ":")
		hours, herr := strconv.Atoi(aggregationInterval[0])
		minutes, merr := strconv.Atoi(aggregationInterval[1])
		seconds, serr := strconv.Atoi(aggregationInterval[2])

		if herr != nil || merr != nil || serr != nil {
			return "", fmt.Errorf("errors parsing metricAggregationInterval: %v, %v, %w", herr, merr, serr)
		}

		starttime = time.Now().Add(-(time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second)).UTC().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s/%s", starttime, endtime), nil
}

func verifyAggregationTypeIsSupported(aggregationType string, data []insights.MetricValue) (*float64, error) {
	var valuePtr *float64
	switch {
	case strings.EqualFold(string(insights.Average), aggregationType) && data[len(data)-1].Average != nil:
		valuePtr = data[len(data)-1].Average
	case strings.EqualFold(string(insights.Total), aggregationType) && data[len(data)-1].Total != nil:
		valuePtr = data[len(data)-1].Total
	case strings.EqualFold(string(insights.Maximum), aggregationType) && data[len(data)-1].Maximum != nil:
		valuePtr = data[len(data)-1].Maximum
	case strings.EqualFold(string(insights.Minimum), aggregationType) && data[len(data)-1].Minimum != nil:
		valuePtr = data[len(data)-1].Minimum
	case strings.EqualFold(string(insights.Count), aggregationType) && data[len(data)-1].Count != nil:
		valuePtr = data[len(data)-1].Count
	default:
		err := fmt.Errorf("unsupported aggregation type %s", insights.AggregationType(strings.ToTitle(aggregationType)))
		return nil, err
	}
	return valuePtr, nil
}
