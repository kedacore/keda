package scalers

import (
	"context"
	"fmt"
	"strings"

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
	Namespace                 string
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

    metricResponse, err := metricsClient.getAzureMetric(metricRequest)
    if err != nil  {
        azureMonitorLog.Error(err, "error getting azure monitor metric")
    }

    return 2, nil
}

func newMonitorClient(metadata *azureMonitorMetadata) azureExternalMetricClient {
    client := insights.NewMetricsClient(metadata.subscriptionID)
    config := auth.NewClientCredentialsConfig(metadata.servicePrincipalID, metadata.servicePrincipalPass, metadata.tentantID)
    
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
		err := fmt.Errorf("Got an empty response for metric %s/%s and aggregate type %s", azMetricRequest.Namespace, azMetricRequest.MetricName, insights.AggregationType(strings.ToTitle(azMetricRequest.Aggregation)))
		return 0, err
	}

	timeseries := *metricVals[0].Timeseries
	if timeseries == nil {
		err := fmt.Errorf("Got metric result for %s/%s and aggregate type %s without timeseries", azMetricRequest.Namespace, azMetricRequest.MetricName, insights.AggregationType(strings.ToTitle(azMetricRequest.Aggregation)))
		return 0, err
	}

	data := *timeseries[0].Data
	if data == nil {
		err := fmt.Errorf("Got metric result for %s/%s and aggregate type %s without any metric values", azMetricRequest.Namespace, azMetricRequest.MetricName, insights.AggregationType(strings.ToTitle(azMetricRequest.Aggregation)))
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
		err := fmt.Errorf("Unsupported aggregation type %s specified in metric %s/%s", insights.AggregationType(strings.ToTitle(azMetricRequest.Aggregation)), azMetricRequest.Namespace, azMetricRequest.MetricName)
		return 0, err
	}

	if valuePtr == nil {
		err := fmt.Errorf("Unable to get value for metric %s/%s with aggregation %s. No value returned by the Azure Monitor", azMetricRequest.Namespace, azMetricRequest.MetricName, azMetricRequest.Aggregation)
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
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s",
		amr.SubscriptionID,
		amr.ResourceGroup,
		amr.ResourceProviderNamespace,
		amr.ResourceType,
		amr.ResourceName)
}