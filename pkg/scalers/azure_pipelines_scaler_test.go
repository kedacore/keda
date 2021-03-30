package scalers

import (
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
	{map[string]string{"organizationUrlFromEnv": "AZP_URL", "personalAccessTokenFromEnv": "AZP_TOKEN", "poolId": "1", "targetPipelinesQueueLength": "1"}, false, testAzurePipelinesResolvedEnv, map[string]string{}},
	// using triggerAuthentication
	{map[string]string{"poolId": "1", "targetPipelinesQueueLength": "1"}, false, testAzurePipelinesResolvedEnv, map[string]string{"organizationUrl": "https://dev.azure.com/sample", "personalAccessToken": "sample"}},
	// missing organizationUrl
	{map[string]string{"organizationUrlFromEnv": "", "personalAccessTokenFromEnv": "sample", "poolId": "1", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// missing personalAccessToken
	{map[string]string{"organizationUrlFromEnv": "AZP_URL", "poolId": "1", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
	// missing poolId
	{map[string]string{"organizationUrlFromEnv": "AZP_URL", "personalAccessTokenFromEnv": "AZP_TOKEN", "poolId": "", "targetPipelinesQueueLength": "1"}, true, testAzurePipelinesResolvedEnv, map[string]string{}},
}

var azurePipelinesMetricIdentifiers = []azurePipelinesMetricIdentifier{
	{&testAzurePipelinesMetadata[1], "azure-pipelines-queue-1"},
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
		meta, err := parseAzurePipelinesMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAzurePipelinesScaler := azurePipelinesScaler{
			metadata:   meta,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockAzurePipelinesScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
