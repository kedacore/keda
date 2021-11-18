package scalers

import (
	"context"
	"net/http"
	"testing"
)

type parseAzurePipelinesMetadataTestData struct {
	metadata    map[string]string
	isError     bool
	resolvedEnv map[string]string
	authParams  map[string]string
}

type azurePipelinesMetricIdentifier struct {
	metadataTestData *parseAzurePipelinesMetadataTestData
	scalerIndex      int
	name             string
}

var testAzurePipelinesResolvedEnv = map[string]string{
	"AZP_URL":   "https://dev.azure.com/sample",
	"AZP_TOKEN": "sample",
}

var testAzurePipelinesMetadata = []parseAzurePipelinesMetadataTestData{
	// empty
	{map[string]string{}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// all properly formed
	{map[string]string{"organizationURLFromEnv": "AZP_URL", "personalAccessTokenFromEnv": "AZP_TOKEN", "poolID": "1", "targetPipelinesQueueLength": "1"}, false, testAzurePipelinesResolvedEnv, map[string]string{}},
	// using triggerAuthentication
	{map[string]string{"poolID": "1", "targetPipelinesQueueLength": "1"}, false, testAzurePipelinesResolvedEnv, map[string]string{"organizationURL": "https://dev.azure.com/sample", "personalAccessToken": "sample"}},
	// missing organizationURL
	{map[string]string{"organizationURLFromEnv": "", "personalAccessTokenFromEnv": "sample", "poolID": "1", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// missing personalAccessToken
	{map[string]string{"organizationURLFromEnv": "AZP_URL", "poolID": "1", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// missing poolID
	{map[string]string{"organizationURLFromEnv": "AZP_URL", "personalAccessTokenFromEnv": "AZP_TOKEN", "poolID": "", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
}

var azurePipelinesMetricIdentifiers = []azurePipelinesMetricIdentifier{
	{&testAzurePipelinesMetadata[1], 0, "s0-azure-pipelines-1"},
	{&testAzurePipelinesMetadata[1], 1, "s1-azure-pipelines-1"},
}

func TestParseAzurePipelinesMetadata(t *testing.T) {
	for _, testData := range testAzurePipelinesMetadata {
		_, err := parseAzurePipelinesMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testData.resolvedEnv, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestAzurePipelinesGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range azurePipelinesMetricIdentifiers {
		meta, err := parseAzurePipelinesMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testData.metadataTestData.resolvedEnv, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex})
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
