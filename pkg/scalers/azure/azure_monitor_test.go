package azure

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
)

type testExtractAzMonitorTestData struct {
	testName      string
	isError       bool
	expectedValue float64
	metricRequest azureExternalMetricRequest
	metricResult  armmonitor.MetricsClientListResponse
}

var testExtractAzMonitordata = []testExtractAzMonitorTestData{
	{"nothing returned", true, -1, azureExternalMetricRequest{}, armmonitor.MetricsClientListResponse{Response: armmonitor.Response{Value: []*armmonitor.Metric{{}}}}},
	{"timeseries null", true, -1, azureExternalMetricRequest{}, armmonitor.MetricsClientListResponse{Response: armmonitor.Response{Value: []*armmonitor.Metric{{Timeseries: nil}}}}},
	{"timeseries empty", true, -1, azureExternalMetricRequest{}, armmonitor.MetricsClientListResponse{Response: armmonitor.Response{Value: []*armmonitor.Metric{{Timeseries: []*armmonitor.TimeSeriesElement{}}}}}},
	{"data nil", true, -1, azureExternalMetricRequest{}, armmonitor.MetricsClientListResponse{Response: armmonitor.Response{Value: []*armmonitor.Metric{{Timeseries: []*armmonitor.TimeSeriesElement{{Data: nil}}}}}}},
	{"data empty", true, -1, azureExternalMetricRequest{}, armmonitor.MetricsClientListResponse{Response: armmonitor.Response{Value: []*armmonitor.Metric{{Timeseries: []*armmonitor.TimeSeriesElement{{Data: []*armmonitor.MetricValue{}}}}}}}},
	{"Total Aggregation requested", false, 40, azureExternalMetricRequest{Aggregation: "Total"}, armmonitor.MetricsClientListResponse{Response: armmonitor.Response{Value: []*armmonitor.Metric{{Timeseries: []*armmonitor.TimeSeriesElement{{Data: []*armmonitor.MetricValue{{Total: newMetricValue(40)}}}}}}}}},
	{"Average Aggregation requested", false, 41, azureExternalMetricRequest{Aggregation: "Average"}, armmonitor.MetricsClientListResponse{Response: armmonitor.Response{Value: []*armmonitor.Metric{{Timeseries: []*armmonitor.TimeSeriesElement{{Data: []*armmonitor.MetricValue{{Average: newMetricValue(41)}}}}}}}}},
	{"Maximum Aggregation requested", false, 42, azureExternalMetricRequest{Aggregation: "Maximum"}, armmonitor.MetricsClientListResponse{Response: armmonitor.Response{Value: []*armmonitor.Metric{{Timeseries: []*armmonitor.TimeSeriesElement{{Data: []*armmonitor.MetricValue{{Maximum: newMetricValue(42)}}}}}}}}},
	{"Minimum Aggregation requested", false, 43, azureExternalMetricRequest{Aggregation: "Minimum"}, armmonitor.MetricsClientListResponse{Response: armmonitor.Response{Value: []*armmonitor.Metric{{Timeseries: []*armmonitor.TimeSeriesElement{{Data: []*armmonitor.MetricValue{{Minimum: newMetricValue(43)}}}}}}}}},
	{"Count Aggregation requested", false, 44, azureExternalMetricRequest{Aggregation: "Count"}, armmonitor.MetricsClientListResponse{Response: armmonitor.Response{Value: []*armmonitor.Metric{{Timeseries: []*armmonitor.TimeSeriesElement{{Data: []*armmonitor.MetricValue{{Count: newMetricValue(44)}}}}}}}}}}

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
