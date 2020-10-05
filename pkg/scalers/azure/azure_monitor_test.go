package azure

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
)

type testExtractAzMonitorTestData struct {
	testName      string
	isError       bool
	expectedValue float64
	metricRequest azureExternalMetricRequest
	metricResult  insights.Response
}

var testExtractAzMonitordata = []testExtractAzMonitorTestData{
	{"nothing returned", true, -1, azureExternalMetricRequest{}, insights.Response{Value: &[]insights.Metric{}}},
	{"timeseries null", true, -1, azureExternalMetricRequest{}, insights.Response{Value: &[]insights.Metric{{Timeseries: nil}}}},
	{"timeseries empty", true, -1, azureExternalMetricRequest{}, insights.Response{Value: &[]insights.Metric{{Timeseries: &[]insights.TimeSeriesElement{}}}}},
	{"data nil", true, -1, azureExternalMetricRequest{}, insights.Response{Value: &[]insights.Metric{{Timeseries: &[]insights.TimeSeriesElement{{Data: nil}}}}}},
	{"data empty", true, -1, azureExternalMetricRequest{}, insights.Response{Value: &[]insights.Metric{{Timeseries: &[]insights.TimeSeriesElement{{Data: &[]insights.MetricValue{}}}}}}},
	{"Total Aggregation requested", false, 40, azureExternalMetricRequest{Aggregation: "Total"}, insights.Response{Value: &[]insights.Metric{{Timeseries: &[]insights.TimeSeriesElement{{Data: &[]insights.MetricValue{{Total: returnFloat64Ptr(40)}}}}}}}},
	{"Average Aggregation requested", false, 41, azureExternalMetricRequest{Aggregation: "Average"}, insights.Response{Value: &[]insights.Metric{{Timeseries: &[]insights.TimeSeriesElement{{Data: &[]insights.MetricValue{{Average: returnFloat64Ptr(41)}}}}}}}},
	{"Maximum Aggregation requested", false, 42, azureExternalMetricRequest{Aggregation: "Maximum"}, insights.Response{Value: &[]insights.Metric{{Timeseries: &[]insights.TimeSeriesElement{{Data: &[]insights.MetricValue{{Maximum: returnFloat64Ptr(42)}}}}}}}},
	{"Minimum Aggregation requested", false, 43, azureExternalMetricRequest{Aggregation: "Minimum"}, insights.Response{Value: &[]insights.Metric{{Timeseries: &[]insights.TimeSeriesElement{{Data: &[]insights.MetricValue{{Minimum: returnFloat64Ptr(43)}}}}}}}},
	{"Count Aggregation requested", false, 44, azureExternalMetricRequest{Aggregation: "Count"}, insights.Response{Value: &[]insights.Metric{{Timeseries: &[]insights.TimeSeriesElement{{Data: &[]insights.MetricValue{{Count: returnFloat64Ptr(44)}}}}}}}},
}

func returnFloat64Ptr(x float64) *float64 {
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
