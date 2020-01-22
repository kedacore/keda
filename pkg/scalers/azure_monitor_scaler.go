package scalers

import (
	"context"
	"fmt"

	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultTargetMetricValue = 5
	azureMonitorMetricName   = "metricName"
)

type azureMonitorScaler struct {
	metadata *azureMonitorMetadata
}

type azureMonitorMetadata struct {
	resourceURI          string
	tentantID            string
	subscriptionID       string
	resourceGroupName    string
	name                 string
	filter               string
	aggregationInterval  string
	aggregationType      string
	servicePrincipalID   string
	servicePrincipalPass string
	targetMetricValue    int
}

var azureMonitorLog = logf.Log.WithName("azure_monitor_scaler")

// NewAzureMonitorScaler stuff
func NewAzureMonitorScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parseAzureMonitorMetadata(metadata, resolvedEnv, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure monitor metadata: %s", err)
	}

	return &azureMonitorScaler{
		metadata: meta,
	}, nil
}

func parseAzureMonitorMetadata(metadata, resolvedEnv, authParams map[string]string) (*azureMonitorMetadata, error) {
	meta := azureMonitorMetadata{}
	meta.targetMetricValue = defaultTargetMetricValue

	/*if val, ok := metadata[metricResourceURI]; ok {
		Length, err := strconv.Atoi(val)
		if err != nil {
			azureMonitorLog.Error(err, "Error parsing azure queue metadata", "queueLengthMet ricName", LengthMetricName)
			return nil, "", fmt.Errorf("Error parsing azure queue metadata %s: %s", LengthMetricName, err.Error())
		}

		meta.targetLength = Length
	}*/

	if val, ok := metadata["resourceURI"]; ok && val != "" {
		meta.resourceURI = val
	} else {
		return nil, fmt.Errorf("no resourceURI given")
	}

	if val, ok := metadata["resourceTenantId"]; ok && val != "" {
		meta.tentantID = val
	} else {
		return nil, fmt.Errorf("no resourceTenantId given")
	}

	if val, ok := metadata["resourceSubscriptionId"]; ok && val != "" {
		meta.subscriptionID = val
	} else {
		return nil, fmt.Errorf("no resourceSubscriptionId given")
	}

	if val, ok := metadata["resourceGroupName"]; ok && val != "" {
		meta.resourceGroupName = val
	} else {
		return nil, fmt.Errorf("no resourceGroupName given")
	}

	if val, ok := metadata[azureMonitorMetricName]; ok && val != "" {
		meta.name = val
	} else {
		return nil, fmt.Errorf("no metricName given")
	}

	if val, ok := metadata["metricFilter"]; ok {
		if val != "" {
			meta.filter = val
		}
	}

	if val, ok := metadata["metricAggregationInterval"]; ok {
		if val != "" {
			meta.aggregationInterval = val
		}
	}

	if val, ok := metadata["metricAggregationType"]; ok && val != "" {
		meta.subscriptionID = val
	} else {
		return nil, fmt.Errorf("no metricAggregationType given")
	}

	if val, ok := metadata["adServicePrincipleId"]; ok && val != "" {
		meta.servicePrincipalID = val
	} else {
		return nil, fmt.Errorf("no adServicePrincipleId given")
	}

	if val, ok := metadata["adServicePrinciplePassword"]; ok {
		meta.servicePrincipalPass = val
	} else {
		return nil, fmt.Errorf("no adServicePrinciplePassword given")
	}

	return &meta, nil
}

// needs to interact with azure monitor
func (s *azureMonitorScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := s.GetAzureMetric()
	if err != nil {
		azureMonitorLog.Error(err, "error getting azure monitor metric")
		return false, err
	}

	return val > 0, nil
}

func (s *azureMonitorScaler) Close() error {
	return nil
}

func (s *azureMonitorScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetMetricVal := resource.NewQuantity(int64(s.metadata.targetMetricValue), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: azureMonitorMetricName, TargetAverageValue: targetMetricVal}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureMonitorScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := s.GetAzureMetric()
	if err != nil {
		azureMonitorLog.Error(err, "error getting azure monitor metric")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(val), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// GetAzureMetric does things
func (s *azureMonitorScaler) GetAzureMetric() (int, error) {
	return defaultTargetMetricValue, nil
}

/*
package externalmetrics

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"k8s.io/klog"
)

type insightsmonitorClient interface {
	List(ctx context.Context, resourceURI string, timespan string, interval *string, metricnames string, aggregation string, top *int32, orderby string, filter string, resultType insights.ResultType, metricnamespace string) (result insights.Response, err error)
}

type monitorClient struct {
	client                insightsmonitorClient
	DefaultSubscriptionID string
}

func NewMonitorClient(defaultsubscriptionID string) AzureExternalMetricClient {
	client := insights.NewMetricsClient(defaultsubscriptionID)
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err == nil {
		client.Authorizer = authorizer
	}

	return &monitorClient{
		client:                client,
		DefaultSubscriptionID: defaultsubscriptionID,
	}
}

func newMonitorClient(defaultsubscriptionID string, client insightsmonitorClient) monitorClient {
	return monitorClient{
		client:                client,
		DefaultSubscriptionID: defaultsubscriptionID,
	}
}

// GetAzureMetric calls Azure Monitor endpoint and returns a metric
func (c *monitorClient) GetAzureMetric(azMetricRequest AzureExternalMetricRequest) (AzureExternalMetricResponse, error) {
	err := azMetricRequest.Validate()
	if err != nil {
		return AzureExternalMetricResponse{}, err
	}

	metricResourceURI := azMetricRequest.MetricResourceURI()
	klog.V(2).Infof("resource uri: %s", metricResourceURI)

	metricResult, err := c.client.List(context.Background(), metricResourceURI,
		azMetricRequest.Timespan, nil,
		azMetricRequest.MetricName, azMetricRequest.Aggregation, nil,
		"", azMetricRequest.Filter, "", "")
	if err != nil {
		return AzureExternalMetricResponse{}, err
	}

	value, err := extractValue(azMetricRequest, metricResult)

	return AzureExternalMetricResponse{
		Value: value,
	}, err
}

func extractValue(azMetricRequest AzureExternalMetricRequest, metricResult insights.Response) (float64, error) {
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
*/
