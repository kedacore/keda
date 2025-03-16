package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scalers/sumologic"
)

type parseSumologicMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type sumologicMetricIdentifier struct {
	metadataTestData *parseSumologicMetadataTestData
	triggerIndex     int
	name             string
}

var validSumologicAuthParams = map[string]string{
	"accessID":  "fakeAccessID",
	"accessKey": "fakeAccessKey",
}

var validSumologicMetadata = map[string]string{
	"host":                 "https://api.sumologic.com",
	"unsafeSsl":            "false",
	"query":                "fakeQuery",
	"queryType":            "logs",
	"dimension":            "fakeDimension",
	"timerange":            "5",
	"timezone":             "UTC",
	"quantization":         "1",
	"activationQueryValue": "5",
	"targetValue":          "1",
	"queryAggregator":      "average",
}

var testSumologicMetadata = []parseSumologicMetadataTestData{
	// Valid metadata, pass.
	{validSumologicMetadata, validSumologicAuthParams, false},
	// Missing host, fail.
	{map[string]string{"query": "fakeQuery"}, validSumologicAuthParams, true},
	// Missing accessID, fail.
	{validSumologicMetadata, map[string]string{"accessKey": "fakeAccessKey"}, true},
	// Missing accessKey, fail.
	{validSumologicMetadata, map[string]string{"accessID": "fakeAccessID"}, true},
	// Invalid queryType, fail.
	{map[string]string{"host": "https://api.sumologic.com", "query": "fakeQuery", "queryType": "invalid"}, validSumologicAuthParams, true},
	// Missing query, fail.
	{map[string]string{"host": "https://api.sumologic.com", "queryType": "logs"}, validSumologicAuthParams, true},
	// Missing timerange, fail.
	{map[string]string{"host": "https://api.sumologic.com", "query": "fakeQuery", "queryType": "logs"}, validSumologicAuthParams, true},
	// Invalid timerange, fail.
	{map[string]string{"host": "https://api.sumologic.com", "query": "fakeQuery", "queryType": "logs", "timerange": "invalid"}, validSumologicAuthParams, true},
	// Missing dimension, fail.
	{map[string]string{"host": "https://api.sumologic.com", "query": "fakeQuery", "queryType": "logs", "timerange": "5"}, validSumologicAuthParams, true},
	// Missing quantization for metrics query, fail.
	{map[string]string{"host": "https://api.sumologic.com", "query": "fakeQuery", "queryType": "metrics", "timerange": "5", "dimension": "fakeDimension"}, validSumologicAuthParams, true},
}

var sumologicMetricIdentifiers = []sumologicMetricIdentifier{
	{&testSumologicMetadata[0], 0, "s0-sumologic-logs"},
	{&testSumologicMetadata[0], 1, "s1-sumologic-logs"},
}

func TestSumologicParseMetadata(t *testing.T) {
	for _, testData := range testSumologicMetadata {
		_, err := parseSumoMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams}, logr.Discard())
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestSumologicGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range sumologicMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseSumoMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validSumologicAuthParams, TriggerIndex: testData.triggerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockSumologicScaler := sumologicScaler{
			metadata: meta,
		}

		metricSpec := mockSumologicScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestSumologicScalerGetMetricsAndActivity(t *testing.T) {
	ctx := context.Background()
	meta, err := parseSumoMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: validSumologicMetadata, AuthParams: validSumologicAuthParams}, logr.Discard())
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	mockSumologicScaler := sumologicScaler{
		metadata: meta,
		client:   &sumologic.Client{
			// Mock the client methods as needed
		},
		logger: logr.Discard(),
	}

	metrics, isActive, err := mockSumologicScaler.GetMetricsAndActivity(ctx, "sumologic-metric")
	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if len(metrics) == 0 {
		t.Error("Expected metrics but got none")
	}
	if !isActive {
		t.Error("Expected scaler to be active but it was not")
	}
}
