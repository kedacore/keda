package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	triggerIndex     int
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
	{map[string]string{"GoogleApplicationCredentials": "Creds"}, map[string]string{"projectId": "myProject", "filter": sdFilter}, false},
	// Credentials from AuthParams with empty creds
	{map[string]string{"GoogleApplicationCredentials": ""}, map[string]string{"projectId": "myProject", "filter": sdFilter}, true},
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
	// properly formed float valueIfNull
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "credentialsFromEnv": "SAMPLE_CREDS", "targetValue": "1.1", "activationTargetValue": "2.1", "valueIfNull": "1.0"}, false},
	// With bad valueIfNull
	{nil, map[string]string{"projectId": "myProject", "filter": sdFilter, "credentialsFromEnv": "SAMPLE_CREDS", "targetValue": "1.1", "activationTargetValue": "2.1", "valueIfNull": "toto"}, true},
}

var gcpStackdriverMetricIdentifiers = []gcpStackdriverMetricIdentifier{
	{&testStackdriverMetadata[1], 0, "s0-gcp-stackdriver-myProject"},
	{&testStackdriverMetadata[1], 1, "s1-gcp-stackdriver-myProject"},
}

func TestStackdriverParseMetadata(t *testing.T) {
	for _, testData := range testStackdriverMetadata {
		_, err := parseStackdriverMetadata(&scalersconfig.ScalerConfig{AuthParams: testData.authParams, TriggerMetadata: testData.metadata, ResolvedEnv: testStackdriverResolvedEnv}, logr.Discard())
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
		meta, err := parseStackdriverMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testStackdriverResolvedEnv, TriggerIndex: testData.triggerIndex}, logr.Discard())
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
