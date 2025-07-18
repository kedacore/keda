package scalers

import (
	"context"
	"testing"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	"host":                "https://api.sumologic.com",
	"unsafeSsl":           "false",
	"query":               "fakeQuery",
	"queryType":           "logs",
	"timerange":           "5",
	"resultField":         "fakeResultField",
	"timezone":            "UTC",
	"quantization":        "1",
	"activationThreshold": "5",
	"threshold":           "1",
	"queryAggregator":     "Avg",
}

var validSumologicMetricsMetadata = map[string]string{
	"host":                "https://api.sumologic.com",
	"unsafeSsl":           "false",
	"query":               "fakeQuery",
	"queryType":           "metrics",
	"timerange":           "5",
	"timezone":            "UTC",
	"quantization":        "1",
	"activationThreshold": "5",
	"threshold":           "1",
	"queryAggregator":     "Avg",
	"rollup":              "Sum",
}

var validSumologicMultiMetricsMetadata = map[string]string{
	"host":                "https://api.sumologic.com",
	"unsafeSsl":           "false",
	"queryType":           "metrics",
	"timerange":           "5m",
	"timezone":            "Asia/Kolkata",
	"quantization":        "60s",
	"activationThreshold": "5",
	"threshold":           "1",
	"queryAggregator":     "Avg",
	"rollup":              "Sum",
	"query.A":             "fakeQueryA",
	"query.B":             "fakeQueryB",
	"query.C":             "fakeQueryC",
	"resultQueryRowID":    "C",
}

var testSumologicMetadata = []parseSumologicMetadataTestData{
	// Valid metadata, pass.
	{validSumologicMetadata, validSumologicAuthParams, false},
	// Valid metrics metadata with rollup, pass.
	{validSumologicMetricsMetadata, validSumologicAuthParams, false},
	// Valid multi-metrics metadata, pass.
	{validSumologicMultiMetricsMetadata, validSumologicAuthParams, false},
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
	// Missing resultField for logs query, fail.
	{map[string]string{"host": "https://api.sumologic.com", "query": "fakeQuery", "queryType": "logs", "timerange": "5"}, validSumologicAuthParams, true},
	// Missing quantization for metrics query, fail.
	{map[string]string{"host": "https://api.sumologic.com", "query": "fakeQuery", "queryType": "metrics", "timerange": "5"}, validSumologicAuthParams, true},
	// Invalid rollup for metrics query, fail.
	{map[string]string{"host": "https://api.sumologic.com", "query": "fakeQuery", "queryType": "metrics", "quantization": "1", "timerange": "5", "rollup": "fake"}, validSumologicAuthParams, true},
}

var sumologicMetricIdentifiers = []sumologicMetricIdentifier{
	{&testSumologicMetadata[0], 0, "s0-sumologic-logs"},
	{&testSumologicMetadata[1], 0, "s0-sumologic-metrics"},
	{&testSumologicMetadata[2], 0, "s0-sumologic-metrics"},
}

func TestSumologicParseMetadata(t *testing.T) {
	for _, testData := range testSumologicMetadata {
		_, err := parseSumoMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error: %v", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestSumologicGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range sumologicMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseSumoMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validSumologicAuthParams, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatalf("Could not parse metadata: %v", err)
		}
		mockSumologicScaler := sumologicScaler{
			metadata: meta,
		}

		metricSpec := mockSumologicScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("Wrong External metric source name: got %s, expected %s", metricName, testData.name)
		}
	}
}
