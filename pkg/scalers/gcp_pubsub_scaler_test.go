package scalers

import (
	"testing"
)

var testPubSubResolvedEnv = map[string]string{
	"SAMPLE_CREDS": "{}",
}

type parsePubSubMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type gcpPubSubMetricIdentifier struct {
	metadataTestData *parsePubSubMetadataTestData
	name             string
}

var testPubSubMetadata = []parsePubSubMetadataTestData{
	{map[string]string{}, true},
	// all properly formed
	{map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// missing subscriptionName
	{map[string]string{"subscriptionName": "", "subscriptionSize": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// missing credentials
	{map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7", "credentialsFromEnv": ""}, true},
	// incorrect credentials
	{map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7", "credentialsFromEnv": "WRONG_CREDS"}, true},
	// malformed subscriptionSize
	{map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "AA", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
}

var gcpPubSubMetricIdentifiers = []gcpPubSubMetricIdentifier{
	{&testPubSubMetadata[1], "gcp-mysubscription"},
}

func TestPubSubParseMetadata(t *testing.T) {
	for _, testData := range testPubSubMetadata {
		_, err := parsePubSubMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testPubSubResolvedEnv})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestGcpPubSubGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range gcpPubSubMetricIdentifiers {
		meta, err := parsePubSubMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testPubSubResolvedEnv})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGcpPubSubScaler := pubsubScaler{nil, meta}

		metricSpec := mockGcpPubSubScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
