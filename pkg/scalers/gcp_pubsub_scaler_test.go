package scalers

import (
	"testing"
)

var testPubSubResolvedEnv = map[string]string{
	"SAMPLE_CREDS": "{}",
}

type parsePubSubMetadataTestData struct {
	authParams map[string]string
	metadata   map[string]string
	isError    bool
}

type gcpPubSubMetricIdentifier struct {
	metadataTestData *parsePubSubMetadataTestData
	name             string
}

var testPubSubMetadata = []parsePubSubMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed
	{nil, map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// // missing subscriptionName
	{nil, map[string]string{"subscriptionName": "", "subscriptionSize": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// // missing credentials
	{nil, map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7", "credentialsFromEnv": ""}, true},
	// incorrect credentials
	{nil, map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7", "credentialsFromEnv": "WRONG_CREDS"}, false},
	// malformed subscriptionSize
	{nil, map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "AA", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// // Credentials from AuthParams
	{map[string]string{"GoogleApplicationCredentials": "Creds", "podIdentityOwner": ""}, map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7"}, false},
	// // Credentials from AuthParams with empty creds
	{map[string]string{"GoogleApplicationCredentials": "", "podIdentityOwner": ""}, map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7"}, true},
}

var gcpPubSubMetricIdentifiers = []gcpPubSubMetricIdentifier{
	{&testPubSubMetadata[1], "gcp-mysubscription"},
}

func TestPubSubParseMetadata(t *testing.T) {
	for _, testData := range testPubSubMetadata {
		_, err := parsePubSubMetadata(&ScalerConfig{AuthParams: testData.authParams, TriggerMetadata: testData.metadata, ResolvedEnv: testPubSubResolvedEnv})
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
