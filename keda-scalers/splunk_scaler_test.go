package scalers

import (
	"context"
	"testing"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

type parseSplunkMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type SplunkMetricIdentifier struct {
	metadataTestData *parseSplunkMetadataTestData
	triggerIndex     int
	name             string
}

var validSplunkAuthParams = map[string]string{
	"username": "fake",
}

var validSplunkMetadata = map[string]string{
	"host":            "https://localhost:8089",
	"unsafeSsl":       "false",
	"targetValue":     "1",
	"activationValue": "5",
	"savedSearchName": "fakeSavedSearchName",
	"valueField":      "count",
}

var testSplunkMetadata = []parseSplunkMetadataTestData{
	// Valid metadata for api token auth, pass.
	{validSplunkMetadata, map[string]string{"username": "fake", "apiToken": "fake"}, false},
	// Valid metadata for basic auth, pass.
	{validSplunkMetadata, map[string]string{"username": "fake", "password": "fake"}, false},
	// No params, missing username, fail.
	{map[string]string{}, map[string]string{}, true},
	// No params, missing host, fail.
	{map[string]string{}, validSplunkAuthParams, true},
	// Invalid host, fail.
	{map[string]string{"host": "missinghttpURIScheme:8089"}, validSplunkAuthParams, true},
	// Invalid unsafeSsl value, fail.
	{map[string]string{"host": "https://localhost:8089", "unsafeSsl": "invalid"}, validSplunkAuthParams, true},
	// Missing targetValue, fail.
	{map[string]string{"host": "https://localhost:8089", "unsafeSsl": "false"}, validSplunkAuthParams, true},
	// Invalid targetValue, fail.
	{map[string]string{"host": "https://localhost:8089", "unsafeSsl": "false", "targetValue": "invalid"}, validSplunkAuthParams, true},
	// Missing activationValue, fail.
	{map[string]string{"host": "https://localhost:8089", "unsafeSsl": "false", "targetValue": "1"}, validSplunkAuthParams, true},
	// Invalid activationValue, fail.
	{map[string]string{"host": "https://localhost:8089", "unsafeSsl": "false", "targetValue": "1", "activationValue": "invalid"}, validSplunkAuthParams, true},
	// Missing savedSearchName, fail.
	{map[string]string{"host": "https://localhost:8089", "unsafeSsl": "false", "targetValue": "1", "activationValue": "5"}, validSplunkAuthParams, true},
	// Missing valueField, fail.
	{map[string]string{"host": "https://localhost:8089", "unsafeSsl": "false", "targetValue": "1", "activationValue": "5", "savedSearchName": "fakeSavedSearchName"}, validSplunkAuthParams, true},
}

var SplunkMetricIdentifiers = []SplunkMetricIdentifier{
	{&testSplunkMetadata[0], 0, "s0-splunk-fakeSavedSearchName"},
	{&testSplunkMetadata[0], 1, "s1-splunk-fakeSavedSearchName"},
}

func TestSplunkParseMetadata(t *testing.T) {
	for _, testData := range testSplunkMetadata {
		_, err := parseSplunkMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestSplunkGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range SplunkMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseSplunkMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validSplunkAuthParams, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockSplunkScaler := SplunkScaler{
			metadata: *meta,
		}

		metricSpec := mockSplunkScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
