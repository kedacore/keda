package scalers

import (
	"context"
	"testing"
)

var testStackdriverResolvedEnv = map[string]string{
	"SAMPLE_CREDS": "{}",
}

type parseStackdriverMetadataTestData struct {
	authParams map[string]string
	metadata   map[string]string
	isError    bool
}

type gcpStackdriverMetricIdentifier struct {
	metadataTestData *parseStackdriverMetadataTestData
	scalerIndex      int
	name             string
}

var filter = "metric.type=\"storage.googleapis.com/storage/object_count\" resource.type=\"gcs_bucket\""

var testStackdriverMetadata = []parseStackdriverMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed
	{nil, map[string]string{"projectId": "myProject", "filter": filter, "targetValue": "7", "metricName": "gcs_bucket", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// all required properly formed
	{nil, map[string]string{"projectId": "myProject", "filter": filter, "metricName": "gcs_bucket", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// missing projectId
	{nil, map[string]string{"filter": filter, "targetValue": "7", "metricName": "gcs_bucket", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// missing filter
	{nil, map[string]string{"projectId": "myProject", "targetValue": "7", "metricName": "gcs_bucket", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// missing metric name
	{nil, map[string]string{"projectId": "myProject", "filter": filter, "targetValue": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// missing credentials
	{nil, map[string]string{"projectId": "myProject", "filter": filter, "targetValue": "7", "metricName": "gcs_bucket"}, true},
	// malformed targetValue
	{nil, map[string]string{"projectId": "myProject", "filter": filter, "targetValue": "aa", "metricName": "gcs_bucket", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// Credentials from AuthParams
	{map[string]string{"GoogleApplicationCredentials": "Creds", "podIdentityOwner": ""}, map[string]string{"projectId": "myProject", "filter": filter, "metricName": "gcs_bucket"}, false},
	// Credentials from AuthParams with empty creds
	{map[string]string{"GoogleApplicationCredentials": "", "podIdentityOwner": ""}, map[string]string{"projectId": "myProject", "filter": filter, "metricName": "gcs_bucket"}, true},
}

var gcpStackdriverMetricIdentifiers = []gcpStackdriverMetricIdentifier{
	{&testStackdriverMetadata[1], 0, "s0-gcp-stackdriver-gcs_bucket"},
	{&testStackdriverMetadata[1], 1, "s1-gcp-stackdriver-gcs_bucket"},
}

func TestStackdriverParseMetadata(t *testing.T) {
	for _, testData := range testStackdriverMetadata {
		_, err := parseStackdriverMetadata(&ScalerConfig{AuthParams: testData.authParams, TriggerMetadata: testData.metadata, ResolvedEnv: testStackdriverResolvedEnv})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestGcpStackdriverGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range gcpStackdriverMetricIdentifiers {
		meta, err := parseStackdriverMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testStackdriverResolvedEnv, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGcpStackdriverScaler := stackdriverScaler{nil, meta}

		metricSpec := mockGcpStackdriverScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
