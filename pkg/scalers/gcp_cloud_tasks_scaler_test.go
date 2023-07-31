package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
)

var testCloudTasksResolvedEnv = map[string]string{
	"SAMPLE_CREDS": "{}",
}

type parseCloudTasksMetadataTestData struct {
	authParams map[string]string
	metadata   map[string]string
	isError    bool
}

type gcpCloudTasksMetricIdentifier struct {
	metadataTestData *parseCloudTasksMetadataTestData
	scalerIndex      int
	name             string
}

var testCloudTasksMetadata = []parseCloudTasksMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed
	{nil, map[string]string{"queueName": "myQueue", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS", "projectId": "myproject", "activationValue": "5"}, false},
	// missing subscriptionName
	{nil, map[string]string{"queueName": "", "value": "7", "projectId": "myproject", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// missing credentials
	{nil, map[string]string{"queueName": "myQueue", "value": "7", "projectId": "myproject", "credentialsFromEnv": ""}, true},
	// malformed subscriptionSize
	{nil, map[string]string{"queueName": "myQueue", "value": "AA", "projectId": "myproject", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// malformed mode
	{nil, map[string]string{"queueName": "", "mode": "AA", "value": "7", "projectId": "myproject", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// malformed activationTargetValue
	{nil, map[string]string{"queueName": "myQueue", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS", "projectId": "myproject", "activationValue": "AA"}, true},
	// Credentials from AuthParams
	{map[string]string{"GoogleApplicationCredentials": "Creds"}, map[string]string{"queueName": "myQueue", "value": "7", "projectId": "myproject"}, false},
	// Credentials from AuthParams with empty creds
	{map[string]string{"GoogleApplicationCredentials": ""}, map[string]string{"queueName": "myQueue", "subscriptionSize": "7", "projectId": "myproject"}, true},
	// properly formed float value and activationTargetValue
	{nil, map[string]string{"queueName": "mysubscription", "value": "7.1", "credentialsFromEnv": "SAMPLE_CREDS", "activationValue": "2.1", "projectId": "myproject"}, false},
}

var gcpCloudTasksMetricIdentifiers = []gcpCloudTasksMetricIdentifier{
	{&testCloudTasksMetadata[1], 0, "s0-gcp-ct-myQueue"},
	{&testCloudTasksMetadata[1], 1, "s1-gcp-ct-myQueue"},
}

func TestCloudTasksParseMetadata(t *testing.T) {
	for _, testData := range testCloudTasksMetadata {
		_, err := parseCloudTasksMetadata(&ScalerConfig{AuthParams: testData.authParams, TriggerMetadata: testData.metadata, ResolvedEnv: testCloudTasksResolvedEnv}, logr.Discard())
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestGcpCloudTasksGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range gcpCloudTasksMetricIdentifiers {
		meta, err := parseCloudTasksMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testCloudTasksResolvedEnv, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGcpCloudTasksScaler := cloudTasksScaler{nil, "", meta, logr.Discard()}

		metricSpec := mockGcpCloudTasksScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
