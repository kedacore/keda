package azure

import (
	"context"
	"strings"
	"testing"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type testExtractAzAppInsightsTestData struct {
	testName      string
	isError       bool
	expectedValue float64
	info          AppInsightsInfo
	metricResult  ApplicationInsightsMetric
}

func mockAppInsightsInfo(aggregationType string) AppInsightsInfo {
	return AppInsightsInfo{
		MetricID:        "testns/test",
		AggregationType: aggregationType,
	}
}

func mockAppInsightsMetric(metricName, aggregationType string, value *float64) ApplicationInsightsMetric {
	metric := ApplicationInsightsMetric{
		Value: map[string]interface{}{
			metricName: map[string]interface{}{},
		},
	}

	if value == nil {
		metric.Value[metricName].(map[string]interface{})[aggregationType] = nil
	} else {
		metric.Value[metricName].(map[string]interface{})[aggregationType] = *value
	}

	return metric
}

func newMetricValue(f float64) *float64 {
	return &f
}

var testExtractAzAppInsightsData = []testExtractAzAppInsightsTestData{
	{"metric not found", true, -1, mockAppInsightsInfo("avg"), mockAppInsightsMetric("test/test", "avg", newMetricValue(0.0))},
	{"metric is nil", true, -1, mockAppInsightsInfo("avg"), mockAppInsightsMetric("testns/test", "avg", nil)},
	{"incorrect aggregation type", true, -1, mockAppInsightsInfo("avg"), mockAppInsightsMetric("testns/test", "max", newMetricValue(0.0))},
}

func TestAzGetAzureAppInsightsMetricValue(t *testing.T) {
	for _, testData := range testExtractAzAppInsightsData {
		value, err := extractAppInsightValue(testData.info, testData.metricResult)
		if testData.isError {
			if err == nil {
				t.Errorf("Test: %v; Expected error but got success. testData: %v", testData.testName, testData)
			}
		} else {
			if err == nil {
				if testData.expectedValue != value {
					t.Errorf("Test: %v; Expected value %v but got %v testData: %v", testData.testName, testData.expectedValue, value, testData)
				}
			} else {
				t.Errorf("Test: %v; Expected success but got error: %v", testData.testName, err)
			}
		}
	}
}

type testAppInsightsMSALClientTestData struct {
	testName       string
	expectError    bool
	info           AppInsightsInfo
	podIdentity    kedav1alpha1.PodIdentityProvider
	errorSubstring string
}

var testAppInsightsMSALClientData = []testAppInsightsMSALClientTestData{
	{"client credentials", false, AppInsightsInfo{ClientID: "1234", ClientPassword: "pw", TenantID: "5678"}, "", ""},
	{"client credentials - pod id none", false, AppInsightsInfo{ClientID: "1234", ClientPassword: "pw", TenantID: "5678"}, kedav1alpha1.PodIdentityProviderNone, ""},
	{"azure workload identity", true, AppInsightsInfo{}, kedav1alpha1.PodIdentityProviderAzureWorkload, "failed to create workload identity credential"},
	{"unsupported identity provider", true, AppInsightsInfo{}, "unsupported", "unsupported pod identity provider"},
}

func TestNewMSALAppInsightsClientCreation(t *testing.T) {
	for _, testData := range testAppInsightsMSALClientData {
		_, err := NewMSALAppInsightsClient(context.TODO(), testData.info, kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentity})
		if testData.expectError {
			if err == nil {
				t.Errorf("Test %v; expected error but got none", testData.testName)
			} else if testData.errorSubstring != "" && !strings.Contains(err.Error(), testData.errorSubstring) {
				t.Errorf("Test: %v; expected error containing '%s' but got '%s'", testData.testName, testData.errorSubstring, err.Error())
			}
		} else {
			if err != nil {
				t.Errorf("Test: %v; expected success but got error: %v", testData.testName, err)
			}
		}
	}
}

type toISO8601TestData struct {
	testName      string
	isError       bool
	time          string
	expectedValue string
}

var toISO8601Data = []toISO8601TestData{
	{testName: "time with no colons", isError: true, time: "00", expectedValue: "doesnotmatter"},
	{testName: "time with too many colons", isError: true, time: "00:00:00", expectedValue: "doesnotmatter"},
	{testName: "time is not a number", isError: true, time: "12a:55", expectedValue: "doesnotmatter"},
	{testName: "valid time", isError: false, time: "12:55", expectedValue: "PT12H55M"},
}

func TestToISO8601(t *testing.T) {
	for _, testData := range toISO8601Data {
		value, err := toISO8601(testData.time)
		if testData.isError {
			if err == nil {
				t.Errorf("Test: %v; Expected error but got success. testData: %v", testData.testName, testData)
			}
		} else {
			if err == nil {
				if testData.expectedValue != value {
					t.Errorf("Test: %v; Expected value %v but got %v testData: %v", testData.testName, testData.expectedValue, value, testData)
				}
			} else {
				t.Errorf("Test: %v; Expected success but got error: %v", testData.testName, err)
			}
		}
	}
}

