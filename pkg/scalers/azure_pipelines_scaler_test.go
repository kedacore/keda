package scalers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type parseAzurePipelinesMetadataTestData struct {
	testName    string
	metadata    map[string]string
	isError     bool
	resolvedEnv map[string]string
	authParams  map[string]string
}

var testAzurePipelinesResolvedEnv = map[string]string{
	"AZP_URL":   "https://dev.azure.com/sample",
	"AZP_TOKEN": "sample",
}

var testAzurePipelinesMetadata = []parseAzurePipelinesMetadataTestData{
	// empty
	{"empty", map[string]string{}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// all properly formed
	{"all properly formed", map[string]string{"organizationURLFromEnv": "AZP_URL", "personalAccessTokenFromEnv": "AZP_TOKEN", "poolID": "1", "targetPipelinesQueueLength": "1"}, false, testAzurePipelinesResolvedEnv, map[string]string{}},
	// using triggerAuthentication
	{"using triggerAuthentication", map[string]string{"poolID": "1", "targetPipelinesQueueLength": "1"}, false, testAzurePipelinesResolvedEnv, map[string]string{"organizationURL": "https://dev.azure.com/sample", "personalAccessToken": "sample"}},
	// missing organizationURL
	{"missing organizationURL", map[string]string{"organizationURLFromEnv": "", "personalAccessTokenFromEnv": "sample", "poolID": "1", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// missing personalAccessToken
	{"missing personalAccessToken", map[string]string{"organizationURLFromEnv": "AZP_URL", "poolID": "1", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// missing poolID
	{"missing poolID", map[string]string{"organizationURLFromEnv": "AZP_URL", "personalAccessTokenFromEnv": "AZP_TOKEN", "poolID": "", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
}

func TestParseAzurePipelinesMetadata(t *testing.T) {
	for _, testData := range testAzurePipelinesMetadata {
		t.Run(testData.testName, func(t *testing.T) {
			var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"count":1,"value":[{"id":1}]}`))
			}))

			// set urls into local stub only if they are already defined
			if _, ok := testData.resolvedEnv["AZP_URL"]; ok {
				testData.resolvedEnv["AZP_URL"] = apiStub.URL
			}
			if _, ok := testData.authParams["organizationURL"]; ok {
				testData.authParams["organizationURL"] = apiStub.URL
			}

			_, err := parseAzurePipelinesMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testData.resolvedEnv, AuthParams: testData.authParams}, http.DefaultClient, context.TODO())
			if err != nil && !testData.isError {
				t.Error("Expected success but got error", err)
			}
			if testData.isError && err == nil {
				t.Error("Expected error but got success")
			}
		})
	}
}

type validateAzurePipelinesPoolTestData struct {
	testName   string
	metadata   map[string]string
	isError    bool
	queryParam string
	response   string
}

var testValidateAzurePipelinesPoolData = []validateAzurePipelinesPoolTestData{
	// poolID exists and only one is returned
	{"poolID exists and only one is returned", map[string]string{"poolID": "1"}, false, "poolID", `{"count":1,"value":[{"id":1}]}`},
	// poolID exists and more than one are returned
	{"poolID exists and more than one are returned", map[string]string{"poolID": "1"}, true, "poolID", `{"count":2,"value":[{"id":1},{"id":2}]}`},
	// poolID doesn't exist
	{"poolID doesn't exist", map[string]string{"poolID": "1"}, true, "poolID", `{"count":0,"value":[]}`},
	// poolName exists and only one is returned
	{"poolName exists and only one is returned", map[string]string{"poolName": "sample"}, false, "poolName", `{"count":1,"value":[{"id":1}]}`},
	// poolName exists and more than one are returned
	{"poolName exists and more than one are returned", map[string]string{"poolName": "sample"}, true, "poolName", `{"count":2,"value":[{"id":1},{"id":2}]}`},
	// poolName doesn't exist
	{"poolName doesn't exist", map[string]string{"poolName": "sample"}, true, "poolName", `{"count":0,"value":[]}`},
	// poolName is used if poolName and poolID are defined
	{"poolName is used if poolName and poolID are defined", map[string]string{"poolName": "sample", "poolID": "1"}, false, "poolName", `{"count":1,"value":[{"id":1}]}`},
}

func TestValidateAzurePipelinesPool(t *testing.T) {
	for _, testData := range testValidateAzurePipelinesPoolData {
		t.Run(testData.testName, func(t *testing.T) {
			var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, ok := r.URL.Query()[testData.queryParam]
				if !ok {
					t.Error("Worng QueryParam")
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(testData.response))
			}))

			authParams := map[string]string{
				"organizationURL":     apiStub.URL,
				"personalAccessToken": "PAT",
			}

			_, err := parseAzurePipelinesMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: nil, AuthParams: authParams}, http.DefaultClient, context.TODO())
			if err != nil && !testData.isError {
				t.Error("Expected success but got error", err)
			}
			if testData.isError && err == nil {
				t.Error("Expected error but got success")
			}
		})
	}
}

type azurePipelinesMetricIdentifier struct {
	scalerIndex int
	name        string
}

var azurePipelinesMetricIdentifiers = []azurePipelinesMetricIdentifier{
	{0, "s0-azure-pipelines-1"},
	{1, "s1-azure-pipelines-1"},
}

func TestAzurePipelinesGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range azurePipelinesMetricIdentifiers {
		var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"count":1,"value":[{ "id":1}]}`))
		}))

		authParams := map[string]string{
			"organizationURL":     apiStub.URL,
			"personalAccessToken": "PAT",
		}

		metadata := map[string]string{
			"poolID":                     "1",
			"targetPipelinesQueueLength": "1",
		}

		meta, err := parseAzurePipelinesMetadata(&ScalerConfig{TriggerMetadata: metadata, ResolvedEnv: nil, AuthParams: authParams, ScalerIndex: testData.scalerIndex}, http.DefaultClient, context.TODO())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		mockAzurePipelinesScaler := azurePipelinesScaler{
			metadata:   meta,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockAzurePipelinesScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
