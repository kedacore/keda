package azure

import (
	"context"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure/auth"

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

type testAppInsightsAuthConfigTestData struct {
	testName    string
	config      string
	info        AppInsightsInfo
	podIdentity kedav1alpha1.PodIdentityProvider
}

const (
	msiConfig               = "msiConfig"
	clientCredentialsConfig = "clientCredentialsConfig"
	workloadIdentityConfig  = "workloadIdentityConfig"
)

var testAppInsightsAuthConfigData = []testAppInsightsAuthConfigTestData{
	{"client credentials", clientCredentialsConfig, AppInsightsInfo{ClientID: "1234", ClientPassword: "pw", TenantID: "5678"}, ""},
	{"client credentials - pod id none", clientCredentialsConfig, AppInsightsInfo{ClientID: "1234", ClientPassword: "pw", TenantID: "5678"}, kedav1alpha1.PodIdentityProviderNone},
	{"azure workload identity", workloadIdentityConfig, AppInsightsInfo{}, kedav1alpha1.PodIdentityProviderAzureWorkload},
}

func TestAzAppInfoGetAuthConfig(t *testing.T) {
	for _, testData := range testAppInsightsAuthConfigData {
		authConfig := getAuthConfig(context.TODO(), testData.info, kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentity})
		switch testData.config {
		case msiConfig:
			if _, ok := authConfig.(auth.MSIConfig); !ok {
				t.Errorf("Test %v; incorrect auth config. expected MSI config", testData.testName)
			}
		case clientCredentialsConfig:
			if _, ok := authConfig.(auth.ClientCredentialsConfig); !ok {
				t.Errorf("Test: %v; incorrect auth config. expected client credentials config", testData.testName)
			}
		case workloadIdentityConfig:
			if _, ok := authConfig.(ADWorkloadIdentityConfig); !ok {
				t.Errorf("Test: %v; incorrect auth config. expected ad workload identity config", testData.testName)
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

type queryParameterTestData struct {
	testName         string
	isError          bool
	info             AppInsightsInfo
	expectedTimespan string
}

var queryParameterData = []queryParameterTestData{
	{testName: "invalid timespace", isError: true, info: AppInsightsInfo{AggregationType: "avg", AggregationTimespan: "00:00:00", Filter: "cloud/roleName eq 'role'"}},
	{testName: "empty filter", isError: false, expectedTimespan: "PT01H02M", info: AppInsightsInfo{AggregationType: "min", AggregationTimespan: "01:02", Filter: ""}},
	{testName: "filter specified", isError: false, expectedTimespan: "PT01H02M", info: AppInsightsInfo{AggregationType: "min", AggregationTimespan: "01:02", Filter: "cloud/roleName eq 'role'"}},
}

func TestQueryParamsForAppInsightsRequest(t *testing.T) {
	for _, testData := range queryParameterData {
		params, err := queryParamsForAppInsightsRequest(testData.info)
		if testData.isError {
			if err == nil {
				t.Errorf("Test: %v; Expected error but got success. testData: %v", testData.testName, testData)
			}
		} else {
			if err == nil {
				if testData.info.AggregationType != params["aggregation"] {
					t.Errorf("Test: %v; Expected aggregation %v actual %v", testData.testName, testData.info.AggregationType, params["aggregation"])
				}
				if testData.expectedTimespan != params["timespan"] {
					t.Errorf("Test: %v; Expected timespan %v actual %v", testData.testName, testData.expectedTimespan, params["timespan"])
				}
				if testData.info.Filter == "" {
					if params["filter"] != nil {
						t.Errorf("Test: %v; Filter should not be included in params", testData.testName)
					}
				} else {
					if params["filter"] != testData.info.Filter {
						t.Errorf("Test: %v; Expected filter %v actual %v", testData.testName, testData.info.Filter, params["filter"])
					}
				}
			} else {
				t.Errorf("Test: %v; Expected success but got error: %v", testData.testName, err)
			}
		}
	}
}
