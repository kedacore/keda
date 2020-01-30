package scalers

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"k8s.io/klog"
)

type azureExternalMetricRequest struct {
	MetricName                string
	SubscriptionID            string
	Type                      string
	ResourceName              string
	ResourceProviderNamespace string
	ResourceType              string
	Aggregation               string
	Timespan                  string
	Filter                    string
	ResourceGroup             string
	Topic                     string
	Subscription              string
}

type azureExternalMetricResponse struct {
	Value float64
}

type azureExternalMetricClient interface {
	getAzureMetric(azMetricRequest azureExternalMetricRequest) (azureExternalMetricResponse, error)
}

type insightsmonitorClient interface {
	List(ctx context.Context, resourceURI string, timespan string, interval *string, metricnames string, aggregation string, top *int32, orderby string, filter string, resultType insights.ResultType, metricnamespace string) (result insights.Response, err error)
}

type monitorClient struct {
	client insightsmonitorClient
}

// GetAzureMetricValue is a func
func GetAzureMetricValue(ctx context.Context, metricMetadata *azureMonitorMetadata) (int32, error) {
	metricsClient := newMonitorClient(metricMetadata)

	metricRequest := azureExternalMetricRequest{
		Timespan:       timeSpan(),
		SubscriptionID: metricMetadata.subscriptionID,
	}

	metricRequest.MetricName = metricMetadata.name
	metricRequest.ResourceGroup = metricMetadata.resourceGroupName
	resourceInfo := strings.Split(metricMetadata.resourceURI, "/")

	if len(resourceInfo) != 3 {
		return -1, fmt.Errorf("resourceURI is missing resource namespace, resource type, or resource name")
	}

	metricRequest.ResourceProviderNamespace = resourceInfo[0]
	metricRequest.ResourceType = resourceInfo[1]
	metricRequest.ResourceName = resourceInfo[2]

	metricRequest.Aggregation = metricMetadata.aggregationType

	filter := metricMetadata.filter
	if filter != "" {
		metricRequest.Filter = filter
	}

	aggregationInterval := metricMetadata.aggregationInterval
	if aggregationInterval != "" {
		metricRequest.Timespan = aggregationInterval
	}

	metricResponse, err := metricsClient.getAzureMetric(metricRequest)
	if err != nil {
		azureMonitorLog.Error(err, "error getting azure monitor metric")
		//return -1, fmt.Errorf("MetricName %s: , ResourceGroup: %s, Namespace: %s, ResourceType: %s, ResourceName: %s, Aggregation: %s, Timespan: %s", metricRequest.MetricName, metricRequest.ResourceGroup, metricRequest.ResourceProviderNamespace, metricRequest.ResourceType, metricRequest.ResourceName, metricRequest.Aggregation, metricRequest.Timespan)
		return -1, fmt.Errorf("Error getting azure monitor metric %s: %s", metricRequest.MetricName, err.Error())
	}

	// casting drops everything after decimal, so round first
	metricValue := int32(math.Round(metricResponse.Value))

	return metricValue, nil
}

func newMonitorClient(metadata *azureMonitorMetadata) azureExternalMetricClient {
	client := insights.NewMetricsClient(metadata.subscriptionID)
	config := auth.NewClientCredentialsConfig(metadata.clientID, metadata.clientPassword, metadata.tentantID)

	authorizer, err := config.Authorizer()
	if err == nil {
		client.Authorizer = authorizer
	}

	return &monitorClient{
		client: client,
	}
}

func (c *monitorClient) getAzureMetric(azMetricRequest azureExternalMetricRequest) (azureExternalMetricResponse, error) {
	err := azMetricRequest.validate()
	if err != nil {
		return azureExternalMetricResponse{}, err
	}

	metricResourceURI := azMetricRequest.metricResourceURI()
	klog.V(2).Infof("resource uri: %s", metricResourceURI)

	metricResult, err := c.client.List(context.Background(), metricResourceURI,
		azMetricRequest.Timespan, nil,
		azMetricRequest.MetricName, azMetricRequest.Aggregation, nil,
		"", azMetricRequest.Filter, "", "")
	if err != nil {
		return azureExternalMetricResponse{}, err
	}

	value, err := extractValue(azMetricRequest, metricResult)

	return azureExternalMetricResponse{
		Value: value,
	}, err
}

func extractValue(azMetricRequest azureExternalMetricRequest, metricResult insights.Response) (float64, error) {
	metricVals := *metricResult.Value

	if len(metricVals) == 0 {
		err := fmt.Errorf("Got an empty response for metric %s/%s and aggregate type %s", azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName, insights.AggregationType(strings.ToTitle(azMetricRequest.Aggregation)))
		return 0, err
	}

	timeseries := *metricVals[0].Timeseries
	if timeseries == nil {
		err := fmt.Errorf("Got metric result for %s/%s and aggregate type %s without timeseries", azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName, insights.AggregationType(strings.ToTitle(azMetricRequest.Aggregation)))
		return 0, err
	}

	data := *timeseries[0].Data
	if data == nil {
		err := fmt.Errorf("Got metric result for %s/%s and aggregate type %s without any metric values", azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName, insights.AggregationType(strings.ToTitle(azMetricRequest.Aggregation)))
		return 0, err
	}

	var valuePtr *float64
	if strings.EqualFold(string(insights.Average), azMetricRequest.Aggregation) && data[len(data)-1].Average != nil {
		valuePtr = data[len(data)-1].Average
	} else if strings.EqualFold(string(insights.Total), azMetricRequest.Aggregation) && data[len(data)-1].Total != nil {
		valuePtr = data[len(data)-1].Total
	} else if strings.EqualFold(string(insights.Maximum), azMetricRequest.Aggregation) && data[len(data)-1].Maximum != nil {
		valuePtr = data[len(data)-1].Maximum
	} else if strings.EqualFold(string(insights.Minimum), azMetricRequest.Aggregation) && data[len(data)-1].Minimum != nil {
		valuePtr = data[len(data)-1].Minimum
	} else if strings.EqualFold(string(insights.Count), azMetricRequest.Aggregation) && data[len(data)-1].Count != nil {
		fValue := float64(*data[len(data)-1].Count)
		valuePtr = &fValue
	} else {
		err := fmt.Errorf("Unsupported aggregation type %s specified in metric %s/%s", insights.AggregationType(strings.ToTitle(azMetricRequest.Aggregation)), azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName)
		return 0, err
	}

	if valuePtr == nil {
		err := fmt.Errorf("Unable to get value for metric %s/%s with aggregation %s. No value returned by the Azure Monitor", azMetricRequest.ResourceProviderNamespace, azMetricRequest.MetricName, azMetricRequest.Aggregation)
		return 0, err
	}

	klog.V(2).Infof("metric type: %s %f", azMetricRequest.Aggregation, *valuePtr)

	return *valuePtr, nil
}

func (amr azureExternalMetricRequest) validate() error {
	// Shared
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

func timeSpan() string {
	// defaults to last five minutes.
	// TODO support configuration via config
	endtime := time.Now().UTC().Format(time.RFC3339)
	starttime := time.Now().Add(-(5 * time.Minute)).UTC().Format(time.RFC3339)
	return fmt.Sprintf("%s/%s", starttime, endtime)
}
