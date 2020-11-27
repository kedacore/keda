package azure

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"k8s.io/klog"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
)

// Much of the code in this file is taken from the Azure Kubernetes Metrics Adapter
// https://github.com/Azure/azure-k8s-metrics-adapter/tree/master/pkg/azure/externalmetrics

type azureExternalMetricRequest struct {
	MetricName                string
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
	Filter              string
	AggregationInterval string
	AggregationType     string
	ClientID            string
	ClientPassword      string
}

// GetAzureMetricValue returns the value of an Azure Monitor metric, rounded to the nearest int
func GetAzureMetricValue(ctx context.Context, info MonitorInfo, podIdentity kedav1alpha1.PodIdentityProvider) (int32, error) {
	var podIdentityEnabled = true

	if podIdentity == "" || podIdentity == kedav1alpha1.PodIdentityProviderNone {
		podIdentityEnabled = false
	}

	client := createMetricsClient(info, podIdentityEnabled)
	requestPtr, err := createMetricsRequest(info)
	if err != nil {
		return -1, err
	}

	return executeRequest(ctx, client, requestPtr)
}

func createMetricsClient(info MonitorInfo, podIdentityEnabled bool) insights.MetricsClient {
	client := insights.NewMetricsClient(info.SubscriptionID)
	var config auth.AuthorizerConfig
	if podIdentityEnabled {
		config = auth.NewMSIConfig()
	} else {
		config = auth.NewClientCredentialsConfig(info.ClientID, info.ClientPassword, info.TenantID)
	}
	authorizer, _ := config.Authorizer()
	client.Authorizer = authorizer

	return client
}

func createMetricsRequest(info MonitorInfo) (*azureExternalMetricRequest, error) {
	metricRequest := azureExternalMetricRequest{
		MetricName:     info.Name,
		SubscriptionID: info.SubscriptionID,
		Aggregation:    info.AggregationType,
		Filter:         info.Filter,
		ResourceGroup:  info.ResourceGroupName,
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

func executeRequest(ctx context.Context, client insights.MetricsClient, request *azureExternalMetricRequest) (int32, error) {
	metricResponse, err := getAzureMetric(ctx, client, *request)
	if err != nil {
		return -1, fmt.Errorf("error getting azure monitor metric %s: %w", request.MetricName, err)
	}

	// casting drops everything after decimal, so round first
	metricValue := int32(math.Round(metricResponse))

	return metricValue, nil
}

func getAzureMetric(ctx context.Context, client insights.MetricsClient, azMetricRequest azureExternalMetricRequest) (float64, error) {
	err := azMetricRequest.validate()
	if err != nil {
		return -1, err
	}

	metricResourceURI := azMetricRequest.metricResourceURI()
	klog.V(2).Infof("resource uri: %s", metricResourceURI)

	metricResult, err := client.List(ctx, metricResourceURI,
		azMetricRequest.Timespan, nil,
		azMetricRequest.MetricName, azMetricRequest.Aggregation, nil,
		"", azMetricRequest.Filter, "", "")
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

	klog.V(2).Infof("metric type: %s %f", azMetricRequest.Aggregation, *valuePtr)

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
			return "", fmt.Errorf("errors parsing metricAggregationInterval: %v, %v, %v", herr, merr, serr)
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
