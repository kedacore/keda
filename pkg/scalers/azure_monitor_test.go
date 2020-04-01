package scalers

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
)

type parseAzMonitorMetadataTestData struct {
	metadata    map[string]string
	isError     bool
	resolvedEnv map[string]string
	authParams  map[string]string
}

var testAzMonitorResolvedEnv = map[string]string{
	"CLIENT_ID":       "xxx",
	"CLIENT_PASSWORD": "yyy",
}

var testParseAzMonitorMetadata = []parseAzMonitorMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}, map[string]string{}},
	// properly formed
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, false, testAzMonitorResolvedEnv, map[string]string{}},
	// no optional parameters
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, false, testAzMonitorResolvedEnv, map[string]string{}},
	// incorrectly formatted resourceURI
	{map[string]string{"resourceURI": "bad/format", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}},
	// improperly formatted aggregationInterval
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:1", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}},
	// missing resourceURI
	{map[string]string{"tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}},
	// missing tenantId
	{map[string]string{"resourceURI": "test/resource/uri", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}},
	// missing subscriptionId
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}},
	// missing resourceGroupName
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}},
	// missing metricName
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}},
	// missing metricAggregationType
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}},
	// filter included
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricFilter": "namespace eq 'default'", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, false, testAzMonitorResolvedEnv, map[string]string{}},
	// missing activeDirectoryClientId
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientPassword": "CLIENT_PASSWORD", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}},
	// missing activeDirectoryClientPassword
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "targetValue": "5"}, true, testAzMonitorResolvedEnv, map[string]string{}},
	// missing targetValue
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "activeDirectoryClientId": "CLIENT_ID", "activeDirectoryClientPassword": "CLIENT_PASSWORD"}, true, testAzMonitorResolvedEnv, map[string]string{}},
	// connection from authParams
	{map[string]string{"resourceURI": "test/resource/uri", "tenantId": "123", "subscriptionId": "456", "resourceGroupName": "test", "metricName": "metric", "metricAggregationInterval": "0:15:0", "metricAggregationType": "Average", "targetValue": "5"}, false, map[string]string{}, map[string]string{"activeDirectoryClientId": "zzz", "activeDirectoryClientPassword": "password"}},
}

func TestAzMonitorParseMetadata(t *testing.T) {
	for _, testData := range testParseAzMonitorMetadata {
		_, err := parseAzureMonitorMetadata(testData.metadata, testData.resolvedEnv, testData.authParams)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success. testData: %v", testData)
		}
	}
}

type testExtractAzMonitorTestData struct {
	testName      string
	isError       bool
	expectedValue float64
	metricRequest azureExternalMetricRequest
	metricResult  insights.Response
}

var testExtractAzMonitordata = []testExtractAzMonitorTestData{
	{"nothing returned", true, -1, azureExternalMetricRequest{}, insights.Response{Value: &[]insights.Metric{}}},
	{"timeseries null", true, -1, azureExternalMetricRequest{}, insights.Response{Value: &[]insights.Metric{insights.Metric{Timeseries: nil}}}},
	{"timeseries empty", true, -1, azureExternalMetricRequest{}, insights.Response{Value: &[]insights.Metric{insights.Metric{Timeseries: &[]insights.TimeSeriesElement{}}}}},
	{"data nil", true, -1, azureExternalMetricRequest{}, insights.Response{Value: &[]insights.Metric{insights.Metric{Timeseries: &[]insights.TimeSeriesElement{insights.TimeSeriesElement{Data: nil}}}}}},
	{"data empty", true, -1, azureExternalMetricRequest{}, insights.Response{Value: &[]insights.Metric{insights.Metric{Timeseries: &[]insights.TimeSeriesElement{insights.TimeSeriesElement{Data: &[]insights.MetricValue{}}}}}}},
	{"Total Aggregation requested", false, 40, azureExternalMetricRequest{Aggregation: "Total"}, insights.Response{Value: &[]insights.Metric{insights.Metric{Timeseries: &[]insights.TimeSeriesElement{insights.TimeSeriesElement{Data: &[]insights.MetricValue{insights.MetricValue{Total: returnFloat64Ptr(40)}}}}}}}},
	{"Average Aggregation requested", false, 41, azureExternalMetricRequest{Aggregation: "Average"}, insights.Response{Value: &[]insights.Metric{insights.Metric{Timeseries: &[]insights.TimeSeriesElement{insights.TimeSeriesElement{Data: &[]insights.MetricValue{insights.MetricValue{Average: returnFloat64Ptr(41)}}}}}}}},
	{"Maximum Aggregation requested", false, 42, azureExternalMetricRequest{Aggregation: "Maximum"}, insights.Response{Value: &[]insights.Metric{insights.Metric{Timeseries: &[]insights.TimeSeriesElement{insights.TimeSeriesElement{Data: &[]insights.MetricValue{insights.MetricValue{Maximum: returnFloat64Ptr(42)}}}}}}}},
	{"Minimum Aggregation requested", false, 43, azureExternalMetricRequest{Aggregation: "Minimum"}, insights.Response{Value: &[]insights.Metric{insights.Metric{Timeseries: &[]insights.TimeSeriesElement{insights.TimeSeriesElement{Data: &[]insights.MetricValue{insights.MetricValue{Minimum: returnFloat64Ptr(43)}}}}}}}},
	{"Count Aggregation requested", false, 44, azureExternalMetricRequest{Aggregation: "Count"}, insights.Response{Value: &[]insights.Metric{insights.Metric{Timeseries: &[]insights.TimeSeriesElement{insights.TimeSeriesElement{Data: &[]insights.MetricValue{insights.MetricValue{Count: returnFloat64Ptr(44)}}}}}}}},
}

func returnFloat64Ptr(x float64) *float64 {
	return &x
}

func returnint64Ptr(x int64) *int64 {
	return &x
}

func TestAzMonitorextractValue(t *testing.T) {
	for _, testData := range testExtractAzMonitordata {
		value, err := extractValue(testData.metricRequest, testData.metricResult)
		if err != nil && !testData.isError {
			t.Errorf("Test: %v; Expected success but got error: %v", testData.testName, err)
		}
		if testData.isError && err == nil {
			t.Errorf("Test: %v; Expected error but got success. testData: %v", testData.testName, testData)
		}
		if err != nil && value != testData.expectedValue {
			t.Errorf("Test: %v; Expected value %v but got %v testData: %v", testData.testName, testData.expectedValue, value, testData)
		}
	}
}
