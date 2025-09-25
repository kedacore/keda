package scalers

import (
	"context"
	"testing"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseSplunkObservabilityMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type SplunkObservabilityMetricIdentifier struct {
	metadataTestData *parseSplunkObservabilityMetadataTestData
	triggerIndex     int
	metricName       string
}

var validSplunkObservabilityAuthParams = map[string]string{
	"accessToken": "my-super-secret-access-token",
	"realm":       "my-realm",
}

var invalidSplunkObservabilityAuthParams = map[string]string{
	"accessToken": "",
	"realm":       "my-realm",
}

var validSplunkObservabilityMetadata = map[string]string{
	"query":                 "data('demo.trans.latency').max().publish()",
	"duration":              "10",
	"targetValue":           "200.0",
	"queryAggregator":       "avg",
	"activationTargetValue": "1.1",
}

var testSplunkObservabilityMetadata = []parseSplunkObservabilityMetadataTestData{
	// Valid metadata and valid auth params, pass.
	{validSplunkObservabilityMetadata, validSplunkObservabilityAuthParams, false},
	// no params at all, fail
	{map[string]string{}, map[string]string{}, true},
	// No metadada but valid auth, fail.
	{map[string]string{}, validSplunkObservabilityAuthParams, true},
	// Valid metadada but no auth params, fail.
	{validSplunkObservabilityMetadata, map[string]string{}, true},
	// Missing 'query' field, fail
	{map[string]string{"duration": "10", "targetValue": "200.0", "queryAggregator": "avg", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'duration' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "targetValue": "200.0", "queryAggregator": "avg", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'targetValue' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "queryAggregator": "avg", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'queryAggregator' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'activationTargetValue' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "queryAggregator": "avg"}, validSplunkObservabilityAuthParams, true},
	// Empty 'accessToken' field
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "queryAggregator": "avg"}, invalidSplunkObservabilityAuthParams, true},
}

var SplunkObservabilityMetricIdentifiers = []SplunkObservabilityMetricIdentifier{
	{&testSplunkObservabilityMetadata[0], 0, "s0-signalfx"},
	{&testSplunkObservabilityMetadata[0], 1, "s1-signalfx"},
}

func TestSplunkObservabilityParseMetadata(t *testing.T) {
	for _, testData := range testSplunkObservabilityMetadata {
		_, err := parseSplunkObservabilityMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestSplunkObservabilityGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range SplunkObservabilityMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseSplunkObservabilityMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validSplunkObservabilityAuthParams, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse Splunk Observability metadata:", err)
		}
		mockSplunkObservabilityScaler := splunkObservabilityScaler{
			metadata: meta,
		}

		metricSpec := mockSplunkObservabilityScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.metricName {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
