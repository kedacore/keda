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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/go-logr/logr"
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
	ResourceURI         string
	TenantID            string
	SubscriptionID      string
	ResourceGroupName   string
	Name                string
	Namespace           string
	Filter              string
	AggregationInterval string
	AggregationType     string
	ClientID            string
	ClientPassword      string
}

var azureMonitorLog = logf.Log.WithName("azure_monitor_scaler")

// GetAzureMetricValue returns the value of an Azure Monitor metric, rounded to the nearest int
func GetAzureMetricValue(ctx context.Context, logger logr.Logger, info MonitorInfo, podIdentity kedav1alpha1.AuthPodIdentity) (float64, error) {
	client, err := createMetricsClient(ctx, logger, info, podIdentity)
	if err != nil {
		return -1, err
	}
	requestPtr, err := createMetricsRequest(info)
	if err != nil {
		return -1, err
	}
	return executeRequest(ctx, client, requestPtr)
}

func createMetricsClient(ctx context.Context, logger logr.Logger, info MonitorInfo, podIdentity kedav1alpha1.AuthPodIdentity) (*armmonitor.MetricsClient, error) {
	var creds azcore.TokenCredential
	var err error
	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// TODO (jorturfer) podIdentity.GetIdentityTenantID(), podIdentity.GetIdentityAuthorityHost()
		creds, err = azidentity.NewClientSecretCredential(info.TenantID, info.ClientID, info.ClientPassword, nil)
	case kedav1alpha1.PodIdentityProviderAzure, kedav1alpha1.PodIdentityProviderAzureWorkload:
		creds, err = NewChainedCredential(logger, podIdentity.GetIdentityID(), podIdentity.Provider)
	default:
		return nil, fmt.Errorf("azure monitor does not support pod identity provider - %s", podIdentity.Provider)
	}
	if err != nil {
		return nil, err
	}
	clientFactory, err := armmonitor.NewClientFactory(info.SubscriptionID, creds, nil)
	if err != nil {
		return nil, err
	}

	return clientFactory.NewMetricsClient(), nil
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

func executeRequest(ctx context.Context, client *armmonitor.MetricsClient, request *azureExternalMetricRequest) (float64, error) {
	metricResponse, err := getAzureMetric(ctx, client, *request)
	if err != nil {
		return -1, fmt.Errorf("error getting azure monitor metric %s: %w", request.MetricName, err)
	}

	return metricResponse, nil
}

func getAzureMetric(ctx context.Context, client *armmonitor.MetricsClient, azMetricRequest azureExternalMetricRequest) (float64, error) {
	err := azMetricRequest.validate()
	if err != nil {
		return -1, err
	}

	metricResourceURI := azMetricRequest.metricResourceURI()
	azureMonitorLog.V(2).Info("metric request", "resource uri", metricResourceURI)

	opts := &armmonitor.MetricsClientListOptions{
		Interval:   nil,
		Top:        nil,
		Orderby:    nil,
		ResultType: nil,
	}
	if azMetricRequest.Timespan != "" {
		opts.Timespan = &azMetricRequest.Timespan
	}
	if azMetricRequest.MetricName != "" {
		opts.Metricnames = &azMetricRequest.MetricName
	}
	if azMetricRequest.MetricNamespace != "" {
		opts.Metricnamespace = &azMetricRequest.MetricNamespace
	}
	if azMetricRequest.Aggregation != "" {
		opts.Aggregation = &azMetricRequest.Aggregation
	}
	if azMetricRequest.Filter != "" {
		opts.Filter = &azMetricRequest.Filter
	}

	metricResult, err := client.List(ctx, metricResourceURI, opts)
	if err != nil {
		return -1, err
	}

	value, err := extractValue(azMetricRequest, metricResult)

	return value, err
}

func extractValue(azMetricRequest azureExternalMetricRequest, metricResult armmonitor.MetricsClientListResponse) (float64, error) {
	metricVals := metricResult.Value
	if len(metricVals) == 0 {
		err := fmt.Errorf("got an empty response for metric %s/%s and aggregate type %s", azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName, azMetricRequest.Aggregation)
		return -1, err
	}

	timeseriesPtr := metricVals[0].Timeseries
	if timeseriesPtr == nil || len(timeseriesPtr) == 0 {
		err := fmt.Errorf("got metric result for %s/%s and aggregate type %s without timeseries", azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName, azMetricRequest.Aggregation)
		return -1, err
	}

	dataPtr := (timeseriesPtr)[0].Data
	if dataPtr == nil || len(dataPtr) == 0 {
		err := fmt.Errorf("got metric result for %s/%s and aggregate type %s without any metric values", azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName, azMetricRequest.Aggregation)
		return -1, err
	}

	valuePtr, err := verifyAggregationTypeIsSupported(azMetricRequest.Aggregation, dataPtr)
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

func verifyAggregationTypeIsSupported(aggregationType string, data []*armmonitor.MetricValue) (*float64, error) {
	if data == nil {
		err := fmt.Errorf("invalid response")
		return nil, err
	}
	var valuePtr *float64
	switch {
	case strings.EqualFold(string(armmonitor.AggregationTypeAverage), aggregationType) && data[len(data)-1].Average != nil:
		valuePtr = data[len(data)-1].Average
	case strings.EqualFold(string(armmonitor.AggregationTypeTotal), aggregationType) && data[len(data)-1].Total != nil:
		valuePtr = data[len(data)-1].Total
	case strings.EqualFold(string(armmonitor.AggregationTypeMaximum), aggregationType) && data[len(data)-1].Maximum != nil:
		valuePtr = data[len(data)-1].Maximum
	case strings.EqualFold(string(armmonitor.AggregationTypeMinimum), aggregationType) && data[len(data)-1].Minimum != nil:
		valuePtr = data[len(data)-1].Minimum
	case strings.EqualFold(string(armmonitor.AggregationTypeCount), aggregationType) && data[len(data)-1].Count != nil:
		valuePtr = data[len(data)-1].Count
	default:
		err := fmt.Errorf("unsupported aggregation type %s", aggregationType)
		return nil, err
	}
	return valuePtr, nil
}
