package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
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
	scalerIndex      int
	name             string
}

type gcpPubSubSubscription struct {
	metadataTestData *parsePubSubMetadataTestData
	scalerIndex      int
	name             string
	projectID        string
}

var testPubSubMetadata = []parsePubSubMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed with deprecated field
	{nil, map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// all properly formed
	{nil, map[string]string{"subscriptionName": "mysubscription", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS", "activationValue": "5"}, false},
	// all properly formed with oldest unacked message age mode
	{nil, map[string]string{"subscriptionName": "mysubscription", "mode": pubsubModeOldestUnackedMessageAge, "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// missing subscriptionName
	{nil, map[string]string{"subscriptionName": "", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// missing credentials
	{nil, map[string]string{"subscriptionName": "mysubscription", "value": "7", "credentialsFromEnv": ""}, true},
	// malformed subscriptionSize
	{nil, map[string]string{"subscriptionName": "mysubscription", "value": "AA", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// malformed mode
	{nil, map[string]string{"subscriptionName": "", "mode": "AA", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// malformed activationTargetValue
	{nil, map[string]string{"subscriptionName": "mysubscription", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS", "activationValue": "AA"}, true},
	// Credentials from AuthParams
	{map[string]string{"GoogleApplicationCredentials": "Creds", "podIdentityOwner": ""}, map[string]string{"subscriptionName": "mysubscription", "value": "7"}, false},
	// Credentials from AuthParams with empty creds
	{map[string]string{"GoogleApplicationCredentials": "", "podIdentityOwner": ""}, map[string]string{"subscriptionName": "mysubscription", "subscriptionSize": "7"}, true},
	// with full link to subscription
	{nil, map[string]string{"subscriptionName": "projects/myproject/subscriptions/mysubscription", "subscriptionSize": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// with full (bad) link to subscription
	{nil, map[string]string{"subscriptionName": "projects/myproject/mysubscription", "subscriptionSize": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// properly formed float value and activationTargetValue
	{nil, map[string]string{"subscriptionName": "mysubscription", "value": "7.1", "credentialsFromEnv": "SAMPLE_CREDS", "activationValue": "2.1"}, false},
}

var gcpPubSubMetricIdentifiers = []gcpPubSubMetricIdentifier{
	{&testPubSubMetadata[1], 0, "s0-gcp-ps-mysubscription"},
	{&testPubSubMetadata[1], 1, "s1-gcp-ps-mysubscription"},
}

var gcpSubscriptionNameTests = []gcpPubSubSubscription{
	{&testPubSubMetadata[11], 1, "mysubscription", "myproject"},
	{&testPubSubMetadata[12], 1, "projects/myproject/mysubscription", ""},
}

func TestPubSubParseMetadata(t *testing.T) {
	for _, testData := range testPubSubMetadata {
		_, err := parsePubSubMetadata(&ScalerConfig{AuthParams: testData.authParams, TriggerMetadata: testData.metadata, ResolvedEnv: testPubSubResolvedEnv}, logr.Discard())
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
		meta, err := parsePubSubMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testPubSubResolvedEnv, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGcpPubSubScaler := pubsubScaler{nil, "", meta, logr.Discard()}

		metricSpec := mockGcpPubSubScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestGcpPubSubSubscriptionName(t *testing.T) {
	for _, testData := range gcpSubscriptionNameTests {
		meta, err := parsePubSubMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testPubSubResolvedEnv, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGcpPubSubScaler := pubsubScaler{nil, "", meta, logr.Discard()}
		subscriptionID, projectID := getSubscriptionData(&mockGcpPubSubScaler)

		if subscriptionID != testData.name || projectID != testData.projectID {
			t.Error("Wrong Subscription parsing:", subscriptionID, projectID)
		}
	}
}
