package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var testGcpCloudTasksResolvedEnv = map[string]string{
	"SAMPLE_CREDS": "{}",
}

type parseGcpCloudTasksMetadataTestData struct {
	authParams map[string]string
	metadata   map[string]string
	isError    bool
}

type gcpCloudTasksMetricIdentifier struct {
	metadataTestData *parseGcpCloudTasksMetadataTestData
	triggerIndex     int
	name             string
}

var testGcpCloudTasksMetadata = []parseGcpCloudTasksMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed
	{nil, map[string]string{"queueName": "myQueue", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS", "projectID": "myproject", "activationValue": "5"}, false},
	// missing subscriptionName
	{nil, map[string]string{"queueName": "", "value": "7", "projectID": "myproject", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// missing credentials
	{nil, map[string]string{"queueName": "myQueue", "value": "7", "projectID": "myproject", "credentialsFromEnv": ""}, true},
	// malformed subscriptionSize
	{nil, map[string]string{"queueName": "myQueue", "value": "AA", "projectID": "myproject", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// malformed mode
	{nil, map[string]string{"queueName": "", "mode": "AA", "value": "7", "projectID": "myproject", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// malformed activationTargetValue
	{nil, map[string]string{"queueName": "myQueue", "value": "7", "credentialsFromEnv": "SAMPLE_CREDS", "projectID": "myproject", "activationValue": "AA"}, true},
	// Credentials from AuthParams
	{map[string]string{"GoogleApplicationCredentials": "Creds"}, map[string]string{"queueName": "myQueue", "value": "7", "projectID": "myproject"}, false},
	// Credentials from AuthParams with empty creds
	{map[string]string{"GoogleApplicationCredentials": ""}, map[string]string{"queueName": "myQueue", "subscriptionSize": "7", "projectID": "myproject"}, true},
	// properly formed float value and activationTargetValue
	{nil, map[string]string{"queueName": "mysubscription", "value": "7.1", "credentialsFromEnv": "SAMPLE_CREDS", "activationValue": "2.1", "projectID": "myproject"}, false},
}

var gcpCloudTasksMetricIdentifiers = []gcpCloudTasksMetricIdentifier{
	{&testGcpCloudTasksMetadata[1], 0, "s0-gcp-ct-myQueue"},
	{&testGcpCloudTasksMetadata[1], 1, "s1-gcp-ct-myQueue"},
}

func TestGcpCloudTasksParseMetadata(t *testing.T) {
	for _, testData := range testGcpCloudTasksMetadata {
		_, err := parseGcpCloudTasksMetadata(&scalersconfig.ScalerConfig{AuthParams: testData.authParams, TriggerMetadata: testData.metadata, ResolvedEnv: testGcpCloudTasksResolvedEnv})
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
		meta, err := parseGcpCloudTasksMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testGcpCloudTasksResolvedEnv, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGcpCloudTasksScaler := gcpCloudTasksScaler{nil, "", meta, logr.Discard()}

		metricSpec := mockGcpCloudTasksScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
