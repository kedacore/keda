package scalers

import (
	"context"
	"fmt"
	"testing"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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

var validSplunkMetadata = map[string]string{
	"username":        "admin",
	"host":            "https://localhost:8089",
	"httpTimeout":     "15s",
	"verifyTLS":       "false",
	"targetValue":     "1",
	"activationValue": "5",
	"savedSearchName": "fakeSavedSearchName",
	"valueField":      "count",
}

var testSplunkMetadata = []parseSplunkMetadataTestData{
	// Valid metadata for api token auth, pass.
	{validSplunkMetadata, map[string]string{"apiToken": "fake"}, false},
	// Valid metadata for basic auth, pass.
	{validSplunkMetadata, map[string]string{"password": "fake"}, false},
	// No params, missing username + host, fail.
	{map[string]string{}, map[string]string{}, true},
	// Invalid host, fail.
	{map[string]string{"username": "admin", "host": "missinghttpURIScheme:8089"}, map[string]string{}, true},
	// Invalid httpTimeout, fail.
	{map[string]string{"username": "admin", "host": "https://localhost:8089", "httpTimeout": "invalid"}, map[string]string{}, true},
	// Invalid verifyTLS value, fail.
	{map[string]string{"username": "admin", "host": "https://localhost:8089", "httpTimeout": "10s", "verifyTLS": "invalid"}, map[string]string{}, true},
	// Missing targetValue, fail.
	{map[string]string{"username": "admin", "apiToken": "fake", "host": "https://localhost:8089", "httpTimeout": "15s", "verifyTLS": "false"}, map[string]string{}, true},
	// Invalid targetValue, fail.
	{map[string]string{"username": "admin", "apiToken": "fake", "host": "https://localhost:8089", "httpTimeout": "15s", "verifyTLS": "false", "targetValue": "invalid"}, map[string]string{}, true},
	// Missing activationValue, fail.
	{map[string]string{"username": "admin", "apiToken": "fake", "host": "https://localhost:8089", "httpTimeout": "15s", "verifyTLS": "false", "targetValue": "1"}, map[string]string{}, true},
	// Invalid activationValue, fail.
	{map[string]string{"username": "admin", "apiToken": "fake", "host": "https://localhost:8089", "httpTimeout": "15s", "verifyTLS": "false", "targetValue": "1", "activationValue": "invalid"}, map[string]string{}, true},
	// Missing savedSearchName, fail.
	{map[string]string{"username": "admin", "apiToken": "fake", "host": "https://localhost:8089", "httpTimeout": "15s", "verifyTLS": "false", "targetValue": "1", "activationValue": "5"}, map[string]string{}, true},
	// Missing valueField, fail.
	{map[string]string{"username": "admin", "apiToken": "fake", "host": "https://localhost:8089", "httpTimeout": "15s", "verifyTLS": "false", "targetValue": "1", "activationValue": "5", "savedSearchName": "fakeSavedSearchName"}, map[string]string{}, true},
}

var SplunkMetricIdentifiers = []SplunkMetricIdentifier{
	{&testSplunkMetadata[0], 0, "s0-splunk-fakeSavedSearchName"},
	{&testSplunkMetadata[0], 1, "s1-splunk-fakeSavedSearchName"},
}

func TestSplunkParseMetadata(t *testing.T) {
	for index, testData := range testSplunkMetadata {
		_, err := parseSplunkMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			fmt.Println(index)
			t.Error("Expected success but got error", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestSplunkGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range SplunkMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseSplunkMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
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
