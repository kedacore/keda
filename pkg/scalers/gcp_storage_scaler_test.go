package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var testGcsResolvedEnv = map[string]string{
	"SAMPLE_CREDS": "",
}

type parseGcsMetadataTestData struct {
	authParams map[string]string
	metadata   map[string]string
	isError    bool
}

type gcpGcsMetricIdentifier struct {
	metadataTestData *parseGcsMetadataTestData
	triggerIndex     int
	name             string
}

var testGcsMetadata = []parseGcsMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed
	{nil, map[string]string{"bucketName": "test-bucket", "targetObjectCount": "7", "maxBucketItemsToScan": "100", "credentialsFromEnv": "SAMPLE_CREDS", "activationTargetObjectCount": "5", "blobPrefix": "blobsubpath", "blobDelimiter": "/"}, false},
	// all properly formed while using defaults
	{nil, map[string]string{"bucketName": "test-bucket", "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// missing bucketName
	{nil, map[string]string{"bucketName": "", "targetObjectCount": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// missing credentials
	{nil, map[string]string{"bucketName": "test-bucket", "targetObjectCount": "7", "credentialsFromEnv": ""}, true},
	// malformed targetObjectCount
	{nil, map[string]string{"bucketName": "test-bucket", "targetObjectCount": "AA", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// malformed maxBucketItemsToScan
	{nil, map[string]string{"bucketName": "test-bucket", "targetObjectCount": "7", "maxBucketItemsToScan": "AA", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// malformed activationTargetObjectCount
	{nil, map[string]string{"bucketName": "test-bucket", "credentialsFromEnv": "SAMPLE_CREDS", "activationTargetObjectCount": "A"}, true},
	// Credentials from AuthParams
	{map[string]string{"GoogleApplicationCredentials": "Creds"}, map[string]string{"bucketName": "test-bucket", "targetLength": "7"}, false},
	// Credentials from AuthParams with empty creds
	{map[string]string{"GoogleApplicationCredentials": ""}, map[string]string{"bucketName": "test-bucket", "subscriptionSize": "7"}, true},
}

var gcpGcsMetricIdentifiers = []gcpGcsMetricIdentifier{
	{&testGcsMetadata[1], 0, "s0-gcp-storage-test-bucket"},
	{&testGcsMetadata[1], 1, "s1-gcp-storage-test-bucket"},
}

func TestGcsParseMetadata(t *testing.T) {
	for _, testData := range testGcsMetadata {
		_, err := parseGcsMetadata(&scalersconfig.ScalerConfig{AuthParams: testData.authParams, TriggerMetadata: testData.metadata, ResolvedEnv: testGcsResolvedEnv})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestGcsGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range gcpGcsMetricIdentifiers {
		meta, err := parseGcsMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testGcsResolvedEnv, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGcsScaler := gcsScaler{nil, nil, "", meta, logr.Discard()}

		metricSpec := mockGcsScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
