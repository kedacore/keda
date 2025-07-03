package scalers

import (
	"context"
	"fmt"
	"testing"

	v2 "k8s.io/api/autoscaling/v2"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseHoneycombMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type honeycombMetricIdentifier struct {
	metadataTestData *parseHoneycombMetadataTestData
	triggerIndex     int
	name             string
}

var testHoneycombMetadata = []parseHoneycombMetadataTestData{
	// missing everything
	{map[string]string{}, map[string]string{}, true},
	// minimal valid
	{map[string]string{"apiKey": "abc", "dataset": "ds", "threshold": "10"}, map[string]string{}, false},
	// with calculation and timeRange
	{map[string]string{"apiKey": "abc", "dataset": "ds", "threshold": "10", "calculation": "SUM", "timeRange": "120"}, map[string]string{}, false},
	// missing apiKey
	{map[string]string{"dataset": "ds", "threshold": "10"}, map[string]string{}, true},
	// missing dataset
	{map[string]string{"apiKey": "abc", "threshold": "10"}, map[string]string{}, true},
	// missing threshold
	{map[string]string{"apiKey": "abc", "dataset": "ds"}, map[string]string{}, true},
	// invalid threshold
	{map[string]string{"apiKey": "abc", "dataset": "ds", "threshold": "notanumber"}, map[string]string{}, true},
}

var honeycombMetricIdentifiers = []honeycombMetricIdentifier{
	{&testHoneycombMetadata[1], 0, "s0-honeycomb"},
	{&testHoneycombMetadata[2], 1, "s1-honeycomb"},
}

func TestHoneycombParseMetadata(t *testing.T) {
	for i, testData := range testHoneycombMetadata {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			cfg := &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				AuthParams:      testData.authParams,
			}
			_, err := parseHoneycombMetadata(cfg)
			if err != nil && !testData.isError {
				t.Errorf("Test case %d: Expected success but got error: %v\nMetadata: %v\nAuthParams: %v", i, err, testData.metadata, testData.authParams)
			}
			if testData.isError && err == nil {
				t.Errorf("Test case %d: Expected error but got success\nMetadata: %v\nAuthParams: %v", i, testData.metadata, testData.authParams)
			}
		})
	}
}

func TestHoneycombGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range honeycombMetricIdentifiers {
		cfg := &scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadataTestData.metadata,
			AuthParams:      testData.metadataTestData.authParams,
			TriggerIndex:    testData.triggerIndex,
		}
		meta, err := parseHoneycombMetadata(cfg)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockScaler := honeycombScaler{
			metadata:   meta,
			logger:     logr.Discard(),
			metricType: v2.AverageValueMetricType,
			httpClient: nil,
		}
		metricSpec := mockScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
