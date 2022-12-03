package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
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

var sdFilter = "metric.type=\"storage.googleapis.com/storage/object_count\" resource.type=\"gcs_bucket\""

var testStackdriverMetadata = []parseStackdriverMetadataTestData{
	{map[string]string{}, map[string]string{}, true},
	// all properly formed
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "targetValue": "7", "credentialsFromEnv": "SAMPLE_CREDS", "activationTargetValue": "5"}, false},
	// all required properly formed
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "credentialsFromEnv": "SAMPLE_CREDS"}, false},
	// missing projectId
	{nil, map[string]string{"filter": sdFilter, "targetValue": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// missing filter
	{nil, map[string]string{"projectId": "myProject", "targetValue": "7", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// missing credentials
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "targetValue": "7"}, true},
	// malformed targetValue
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "targetValue": "aa", "credentialsFromEnv": "SAMPLE_CREDS"}, true},
	// malformed activationTargetValue
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "credentialsFromEnv": "SAMPLE_CREDS", "activationTargetValue": "a"}, true},
	// Credentials from AuthParams
	{map[string]string{"GoogleApplicationCredentials": "Creds", "podIdentityOwner": ""}, map[string]string{"projectId": "myProject", "filter": sdFilter}, false},
	// Credentials from AuthParams with empty creds
	{map[string]string{"GoogleApplicationCredentials": "", "podIdentityOwner": ""}, map[string]string{"projectId": "myProject", "filter": sdFilter}, true},
	// With aggregation info
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "credentialsFromEnv": "SAMPLE_CREDS", "alignmentPeriodSeconds": "120", "alignmentAligner": "sum", "alignmentReducer": "percentile_99"}, false},
	// With minimal aggregation info
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "credentialsFromEnv": "SAMPLE_CREDS", "alignmentPeriodSeconds": "120"}, false},
	// With too short alignment period
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "credentialsFromEnv": "SAMPLE_CREDS", "alignmentPeriodSeconds": "30"}, true},
	// With bad alignment period
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "credentialsFromEnv": "SAMPLE_CREDS", "alignmentPeriodSeconds": "a"}, true},
	// properly formed float targetValue and activationTargetValue
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "credentialsFromEnv": "SAMPLE_CREDS", "targetValue": "1.1", "activationTargetValue": "2.1"}, false},
}

var gcpStackdriverMetricIdentifiers = []gcpStackdriverMetricIdentifier{
	{&testStackdriverMetadata[1], 0, "s0-gcp-stackdriver-myProject"},
	{&testStackdriverMetadata[1], 1, "s1-gcp-stackdriver-myProject"},
}

func TestStackdriverParseMetadata(t *testing.T) {
	for _, testData := range testStackdriverMetadata {
		_, err := parseStackdriverMetadata(&ScalerConfig{AuthParams: testData.authParams, TriggerMetadata: testData.metadata, ResolvedEnv: testStackdriverResolvedEnv}, logr.Discard())
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
		meta, err := parseStackdriverMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testStackdriverResolvedEnv, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockGcpStackdriverScaler := stackdriverScaler{nil, "", meta, logr.Discard()}

		metricSpec := mockGcpStackdriverScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
